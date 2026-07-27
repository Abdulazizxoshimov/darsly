package token

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// Maker is the interface for token generation and validation.
type Maker interface {
	Generate(ctx context.Context, sub, sessionID, role string) (access, refresh string, err error)
	ValidateAccess(ctx context.Context, token string) (*Claims, error)
	Rotate(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error)
	SessionFromRefresh(refreshToken string) (sessionID string, err error)
	Revoke(ctx context.Context, jti string) error
	RevokeRefresh(ctx context.Context, refreshToken string) error
	RevokeAllUserSessions(ctx context.Context, userID string) error
	StoreSession(ctx context.Context, sessionID, payload string, ttl time.Duration) error
	RevokeSession(ctx context.Context, sessionID string) error
}

// DefaultRefreshGrace — refresh rotatsiyasining idempotentlik oynasi (grace).
// Config'da JWT_REFRESH_GRACE berilmasa shu ishlatiladi.
const DefaultRefreshGrace = 60 * time.Second

type JWTMaker struct {
	signingKey  []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
	graceTTL    time.Duration
	redis       *redis.Client
	redisPrefix string
	log         logger.Logger
}

// gracePair — rotatsiya natijasining grace oynasidagi nusxasi. Eski refresh JTI
// bilan qayta so'rov kelsa xuddi shu juftlik qaytariladi (idempotent rotatsiya).
//
// XAVFSIZLIK MUROSASI: bu yozuv Redis'da tayyor access+refresh tokenlarni saqlaydi,
// ya'ni Redis'ga o'qish huquqi bo'lgan hujumchi ularni o'qiy oladi. Qabul qilinadi,
// chunki (a) TTL juda qisqa (default 60s), (b) Redis'da sessiya kalitlari allaqachon
// bor va u ishonchli perimetr ichida, (c) alternativasi — mobil tarmoq uzilishida
// foydalanuvchining barcha qurilmasidan chiqib ketishi — ancha og'ir zarar.
type gracePair struct {
	SID     string `json:"sid"`
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

func NewJWTMaker(
	signingKey []byte,
	accessTTL, refreshTTL, refreshGrace time.Duration,
	rdb *redis.Client,
	prefix string,
	log logger.Logger,
) Maker {
	if prefix == "" {
		prefix = "auth"
	}
	if refreshGrace < 0 {
		refreshGrace = 0
	}
	return &JWTMaker{
		signingKey:  signingKey,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
		graceTTL:    refreshGrace,
		redis:       rdb,
		redisPrefix: prefix,
		log:         log,
	}
}

// Generate issues a new access+refresh token pair for the given subject/session/role.
// The refresh token's JTI is stored in Redis so it can be rotated or revoked.
func (m *JWTMaker) Generate(ctx context.Context, sub, sessionID, role string) (string, string, error) {
	now := time.Now().UTC()

	access, err := m.sign(jwt.MapClaims{
		"sub":  sub,
		"sid":  sessionID,
		"role": role,
		"type": "access",
		"iat":  now.Unix(),
		"exp":  now.Add(m.accessTTL).Unix(),
		"jti":  uuid.NewString(),
	})
	if err != nil {
		m.log.Error(ctx, "failed to sign access token", logger.Error(err))
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	jti := uuid.NewString()
	refresh, err := m.sign(jwt.MapClaims{
		"sub":  sub,
		"sid":  sessionID,
		"role": role,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(m.refreshTTL).Unix(),
		"jti":  jti,
	})
	if err != nil {
		m.log.Error(ctx, "failed to sign refresh token", logger.Error(err))
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	if err := m.redis.Set(ctx, m.refreshKey(jti), sessionID, m.refreshTTL).Err(); err != nil {
		m.log.Error(ctx, "failed to persist refresh jti", logger.Error(err))
		return "", "", fmt.Errorf("persist refresh jti: %w", err)
	}

	return access, refresh, nil
}

// ValidateAccess parses and validates an access token, then confirms the session
// is still active in Redis. Returns parsed Claims on success.
func (m *JWTMaker) ValidateAccess(ctx context.Context, tokenStr string) (*Claims, error) {
	raw, err := m.parse(tokenStr, "access")
	if err != nil {
		return nil, err
	}

	sid, _ := raw["sid"].(string)
	if sid == "" {
		return nil, errors.New("token: missing sid claim")
	}

	exists, err := m.redis.Exists(ctx, m.sessionKey(sid)).Result()
	if err != nil {
		// Redis temporarily unavailable: degrade gracefully — trust JWT signature/expiry
		// rather than returning 401 to all users and causing a full outage.
		m.log.Error(ctx, "redis: session check failed — degrading to JWT-only validation", logger.Error(err))
	} else if exists == 0 {
		return nil, errors.New("token: session revoked or not found")
	}

	return mapToClaims(raw), nil
}

// Rotate validates a refresh token, atomically swaps the JTI in Redis, and returns
// a new access+refresh pair. Signing happens before Redis mutation to avoid partial state.
func (m *JWTMaker) Rotate(ctx context.Context, oldRefresh string) (string, string, error) {
	raw, err := m.parse(oldRefresh, "refresh")
	if err != nil {
		return "", "", err
	}

	jti, _ := raw["jti"].(string)
	sid, _ := raw["sid"].(string)
	sub, _ := raw["sub"].(string)
	role, _ := raw["role"].(string)

	if jti == "" || sid == "" || sub == "" {
		return "", "", errors.New("token: refresh token missing required claims")
	}

	storedSid, err := m.redis.Get(ctx, m.refreshKey(jti)).Result()
	if errors.Is(err, redis.Nil) {
		// JTI yo'q — ikki xil sabab bo'lishi mumkin:
		//
		//  1) GRACE OYNASI: rotatsiya server tomonda BAJARILGAN, lekin javob klientga
		//     yetmagan (mobil 4G uzilishi) yoki ilovada ikki parallel 401-retry ketgan.
		//     Klient aynan shu eski refresh bilan qayta uradi. Bu o'g'irlik EMAS —
		//     shuning uchun grace oynasida saqlangan AYNI o'sha juftlikni qaytaramiz
		//     (idempotent rotatsiya) va sessiyani o'ldirmaymiz. Aks holda dars o'rtasida
		//     foydalanuvchining telefon+planshet+web sessiyalari birdaniga uzilardi.
		//
		//  2) GRACE TASHQARISI: klassik refresh-token reuse. OAuth rotatsiya
		//     ko'rsatmasiga ko'ra o'g'irlik deb qaraladi va butun sessiya-oilasi bekor
		//     qilinadi (na hujumchi, na qurbon kirish saqlamaydi).
		if pair, ok := m.graceLookup(ctx, jti, sid); ok {
			m.log.Info(ctx, "token: refresh grace hit — idempotent rotatsiya qaytarildi",
				logger.String("sub", sub), logger.String("sid", sid))
			return pair.Access, pair.Refresh, nil
		}
		m.log.Warn(ctx, "token: refresh reuse detected — revoking all user sessions",
			logger.String("sub", sub), logger.String("sid", sid))
		if rErr := m.RevokeAllUserSessions(ctx, sub); rErr != nil {
			m.log.Error(ctx, "token: revoke all sessions after reuse failed", logger.Error(rErr))
		}
		return "", "", errors.New("token: refresh token already used or revoked")
	}
	if err != nil {
		m.log.Error(ctx, "redis: get refresh jti failed", logger.Error(err))
		return "", "", fmt.Errorf("redis get: %w", err)
	}
	if storedSid != sid {
		// sid mismatch is a sign of token theft — log it prominently and kill the whole
		// session family (same rationale as reuse above).
		m.log.Warn(ctx, "token: jti/sid mismatch — possible token theft",
			logger.String("sub", sub), logger.String("sid", sid))
		if rErr := m.RevokeAllUserSessions(ctx, sub); rErr != nil {
			m.log.Error(ctx, "token: revoke all sessions after sid mismatch failed", logger.Error(rErr))
		}
		return "", "", errors.New("token: session mismatch")
	}

	// Sessiya kaliti mavjudligini tekshiramiz. Ilgari Rotate faqat refresh JTI'ni
	// ko'rardi: sessiya kaliti (sess:<sid>) muddati tugagach ValidateAccess 401 berardi,
	// lekin Rotate baribir "yaroqli" juftlik qaytarardi → klient CHEKSIZ refresh siklida
	// qolib, hech qachon toza logout'ga tushmasdi. Endi sessiya yo'q bo'lsa xato
	// qaytaramiz va klient toza logout qiladi.
	sessExists, sErr := m.redis.Exists(ctx, m.sessionKey(sid)).Result()
	if sErr != nil {
		// Redis vaqtincha ishlamayapti — ValidateAccess bilan bir xil siyosat:
		// hammani 401 qilib to'liq uzilish yasagandan ko'ra JWT imzosiga tayanamiz.
		m.log.Error(ctx, "redis: rotate session check failed — degrading to JWT-only validation", logger.Error(sErr))
	} else if sessExists == 0 {
		m.log.Warn(ctx, "token: session expired/revoked — rotation refused",
			logger.String("sub", sub), logger.String("sid", sid))
		// Yetim refresh JTI'ni tozalaymiz (aks holda keyingi urinish "reuse" deb
		// baholanib butun oilani bekor qilardi).
		_ = m.redis.Del(ctx, m.refreshKey(jti)).Err()
		return "", "", errors.New("token: session expired or revoked")
	}

	// Sign before touching Redis: if signing fails, Redis state stays consistent.
	now := time.Now().UTC()
	newJTI := uuid.NewString()

	access, err := m.sign(jwt.MapClaims{
		"sub":  sub,
		"sid":  sid,
		"role": role,
		"type": "access",
		"iat":  now.Unix(),
		"exp":  now.Add(m.accessTTL).Unix(),
		"jti":  uuid.NewString(),
	})
	if err != nil {
		return "", "", fmt.Errorf("sign access token: %w", err)
	}

	refresh, err := m.sign(jwt.MapClaims{
		"sub":  sub,
		"sid":  sid,
		"role": role,
		"type": "refresh",
		"iat":  now.Unix(),
		"exp":  now.Add(m.refreshTTL).Unix(),
		"jti":  newJTI,
	})
	if err != nil {
		return "", "", fmt.Errorf("sign refresh token: %w", err)
	}

	// Grace yozuvi: eski JTI bilan kelgan takroriy so'rovga ayni shu juftlikni
	// qaytarish uchun. Marshal xato bersa grace'siz davom etamiz (rotatsiya
	// muhimroq) — bu holda eski xatti-harakat (reuse → revoke) qoladi.
	var graceVal []byte
	if m.graceTTL > 0 {
		if b, mErr := json.Marshal(gracePair{SID: sid, Access: access, Refresh: refresh}); mErr == nil {
			graceVal = b
		} else {
			m.log.Error(ctx, "token: grace pair marshal failed", logger.Error(mErr))
		}
	}

	// Rotatsiyani ATOMIK compare-and-swap bilan yozamiz (rotateScript).
	//
	// Nega TxPipelined YARAMAYDI: MULTI/EXEC WATCH'siz — bu CAS emas. Yuqoridagi
	// Get bilan yozuv orasi check-then-act oynasi edi: bitta refresh bilan kelgan N ta
	// parallel so'rov hammasi tekshiruvdan o'tib, HAR BIRI o'z yangi JTI'sini yozardi
	// ("sessiya forking"): bitta sessiyada N ta tirik refresh zanjiri, hech biri bekor
	// qilinmaydi. Bu rotatsiyaning asosiy invariantini buzardi va refresh-reuse o'g'irlik
	// detektorini butunlay chetlab o'tish imkonini berardi.
	//
	// Skript eski JTI'ni faqat qiymati hamon kutilgan sid bo'lsa iste'mol qiladi va
	// ayni o'sha atomik qadamda grace yozuvini ham yozadi. Redis bir oqimli va skript
	// bo'linmas bajarilgani uchun: g'olib aniq bitta bo'ladi, mag'lublar esa g'olibning
	// grace yozuvini ALBATTA topadi (u allaqachon yozilgan) va o'sha juftlikni oladi.
	graceMS := int64(0)
	if graceVal != nil {
		graceMS = m.graceTTL.Milliseconds()
	}
	won, err := rotateScript.Run(ctx, m.redis,
		[]string{m.refreshKey(jti), m.refreshKey(newJTI), m.graceKey(jti), m.sessionKey(sid)},
		sid, int64(m.refreshTTL.Seconds()), graceMS, string(graceVal), sub,
	).Int64()
	if err != nil {
		m.log.Error(ctx, "redis: jti rotation failed", logger.Error(err))
		return "", "", fmt.Errorf("jti rotation: %w", err)
	}

	if won == 0 {
		// Poygada yutqazdik: boshqa parallel so'rov shu JTI'ni allaqachon iste'mol qilgan.
		// Bu o'g'irlik EMAS — g'olibning juftligini qaytaramiz (foydalanuvchi zarar ko'rmaydi)
		// va yasagan juftligimizni tashlab yuboramiz (u Redis'ga yozilmagan).
		if pair, ok := m.graceLookup(ctx, jti, sid); ok {
			m.log.Info(ctx, "token: parallel rotatsiya poygasi — g'olibning juftligi qaytarildi",
				logger.String("sub", sub), logger.String("sid", sid))
			return pair.Access, pair.Refresh, nil
		}
		// Grace o'chirilgan (yoki yozuv yo'q): bitta g'olib qoladi, qolganlari xato oladi.
		// Sessiya-oilasini bekor QILMAYMIZ — bu ketma-ket reuse emas, parallel poyga.
		m.log.Warn(ctx, "token: parallel rotatsiya poygasida yutqazildi (grace yo'q)",
			logger.String("sub", sub), logger.String("sid", sid))
		return "", "", errors.New("token: concurrent rotation, retry")
	}

	return access, refresh, nil
}

// rotateScript — refresh rotatsiyasining atomik compare-and-swap qadami.
//
//	KEYS[1] eski refresh JTI kaliti   KEYS[2] yangi refresh JTI kaliti
//	KEYS[3] grace kaliti (eski JTI)   KEYS[4] sessiya kaliti
//	ARGV[1] sid  ARGV[2] refreshTTL (s)  ARGV[3] graceTTL (ms, 0=o'chirilgan)
//	ARGV[4] grace payload (JSON)         ARGV[5] sessiya payload (userID)
//
// Qaytaradi: 1 — rotatsiya bajarildi (g'olib), 0 — JTI allaqachon iste'mol qilingan.
// Grace TTL millisekundda (PX) — sub-sekundli qiymatlar 0 ga aylanib ketmasin.
//
// ⚠️ REDIS CLUSTER: skript 4 ta kalitga tegadi va ular turli hash-slotlarda bo'lishi
// mumkin → cluster rejimida `CROSSSLOT Keys in request don't hash to the same slot`
// xatosi chiqadi. Hozir muammo yo'q: deploy BITTA standalone instans (`redis_mode:standalone`),
// va bu cheklov yangi emas — o'rnini bosgan MULTI/EXEC ham xuddi shu talabga bo'ysunardi.
// Gorizontal masshtablashda (Redis Cluster) kalitlarga hash-tag qo'shish kerak, masalan
// `session:{<sid>}:refresh:<jti>` — shunda bitta sessiyaning barcha kalitlari bir slotga
// tushadi. Bu kalit sxemasini o'zgartiradi, ya'ni deploy paytida mavjud sessiyalar
// yaroqsiz bo'ladi (hamma qayta login qiladi) — migratsiyani rejalashtirib bajarish kerak.
var rotateScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) ~= ARGV[1] then
  return 0
end
redis.call('DEL', KEYS[1])
redis.call('SET', KEYS[2], ARGV[1], 'EX', ARGV[2])
if tonumber(ARGV[3]) > 0 then
  redis.call('SET', KEYS[3], ARGV[4], 'PX', ARGV[3])
end
redis.call('SET', KEYS[4], ARGV[5], 'EX', ARGV[2])
return 1
`)

// graceLookup eski refresh JTI uchun grace oynasida saqlangan juftlikni qaytaradi.
// Grace yozuvi topilmasa/buzuq bo'lsa/sid mos kelmasa yoki sessiya bekor qilingan
// bo'lsa (false) — bunda chaqiruvchi oddiy reuse siyosatini qo'llaydi.
func (m *JWTMaker) graceLookup(ctx context.Context, jti, sid string) (gracePair, bool) {
	if m.graceTTL <= 0 {
		return gracePair{}, false
	}
	val, err := m.redis.Get(ctx, m.graceKey(jti)).Result()
	if err != nil {
		return gracePair{}, false
	}
	var g gracePair
	if err := json.Unmarshal([]byte(val), &g); err != nil {
		return gracePair{}, false
	}
	if g.SID != sid || g.Access == "" || g.Refresh == "" {
		return gracePair{}, false
	}
	// Logout / parol reset sessiyani o'chirgan bo'lsa grace ham ishlamasligi shart —
	// aks holda bekor qilingan sessiya grace orqali tirilardi.
	if exists, err := m.redis.Exists(ctx, m.sessionKey(sid)).Result(); err == nil && exists == 0 {
		return gracePair{}, false
	}
	return g, true
}

// SessionFromRefresh refresh token'ning IMZOSI va turini tekshiradi va sessiya
// id'sini (sid) qaytaradi. Redis'ga TEGMAYDI, tokenni iste'mol QILMAYDI, hech qanday
// holatni o'zgartirmaydi — ya'ni chaqirish arzon (bitta HMAC-SHA256) va xavfsiz.
//
// Nima uchun kerak: rate-limit kabi infratuzilma qatlamlari sessiyani o'lcham sifatida
// ishlatishi mumkin, LEKIN buni imzo tekshirilmagan sid bilan qilish yaroqsiz. Aks holda
// istalgan kishi soxta imzoli token ichiga QURBONNING sid'ini yozib, uning cheklov
// hisoblagichini to'ldirib qo'yadi (qurbon refresh qila olmaydi → tizimdan chiqariladi).
// "Tekshirilmagan ma'lumot rate-limit kaliti bo'lmaydi" printsipi shu metod bilan
// ta'minlanadi. Imzo yaroqsiz bo'lsa xato qaytadi va chaqiruvchi IP shiftiga tushadi.
func (m *JWTMaker) SessionFromRefresh(refreshToken string) (string, error) {
	claims, err := m.parse(refreshToken, "refresh")
	if err != nil {
		return "", err
	}
	sid, _ := claims["sid"].(string)
	if sid == "" {
		return "", errors.New("token: refresh token missing sid claim")
	}
	return sid, nil
}

// Revoke deletes a refresh token's JTI from Redis (use on logout).
func (m *JWTMaker) Revoke(ctx context.Context, jti string) error {
	return m.redis.Del(ctx, m.refreshKey(jti)).Err()
}

// RevokeRefresh fully invalidates a session from its refresh token: it parses the
// token to recover the JTI (refresh) and SID (session), then deletes both Redis
// keys so neither the access token nor the refresh token can be used again.
// Safe to call with an invalid/expired token — it becomes a no-op.
func (m *JWTMaker) RevokeRefresh(ctx context.Context, refreshToken string) error {
	claims, err := m.parse(refreshToken, "refresh")
	if err != nil {
		return nil // idempotent: already invalid
	}
	if jti, _ := claims["jti"].(string); jti != "" {
		_ = m.redis.Del(ctx, m.refreshKey(jti)).Err()
	}
	if sid, _ := claims["sid"].(string); sid != "" {
		_ = m.redis.Del(ctx, m.sessionKey(sid)).Err()
	}
	return nil
}

// RevokeAllUserSessions invalidates every active session and refresh token belonging
// to userID. It is called after a password reset or account deactivation so that any
// stolen or lingering session is killed immediately — deleting DB refresh rows alone
// is not enough because ValidateAccess/Rotate consult Redis, not the DB.
//
// The Redis schema is used as-is: session keys (prefix:sess:<sid>) store the userID as
// their value, and refresh keys (prefix:refresh:<jti>) store the sessionID. We first
// collect the user's session ids (deleting those session keys → access tokens die), then
// delete every refresh jti whose stored sid belongs to the user (→ refresh rotation dies).
func (m *JWTMaker) RevokeAllUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}

	sessPrefix := m.redisPrefix + ":sess:"
	userSids := map[string]struct{}{}
	if err := m.scanKeys(ctx, sessPrefix+"*", func(key string) {
		val, err := m.redis.Get(ctx, key).Result()
		if err != nil {
			return
		}
		if val == userID {
			userSids[strings.TrimPrefix(key, sessPrefix)] = struct{}{}
			_ = m.redis.Del(ctx, key).Err()
		}
	}); err != nil {
		return fmt.Errorf("revoke user sessions (scan sessions): %w", err)
	}

	if len(userSids) == 0 {
		return nil
	}

	gracePrefix := m.gracePrefix()
	if err := m.scanKeys(ctx, m.redisPrefix+":refresh:*", func(key string) {
		val, err := m.redis.Get(ctx, key).Result()
		if err != nil {
			return
		}
		// Oddiy refresh kaliti qiymati = sid; grace kaliti (refresh:used:<jti>) esa
		// JSON saqlaydi — sid uning ichidan olinadi. Grace yozuvlari ham o'chirilishi
		// SHART, aks holda bekor qilingan sessiya grace orqali tirilardi.
		sid := val
		if strings.HasPrefix(key, gracePrefix) {
			var g gracePair
			if json.Unmarshal([]byte(val), &g) != nil {
				return
			}
			sid = g.SID
		}
		if _, ok := userSids[sid]; ok {
			_ = m.redis.Del(ctx, key).Err()
		}
	}); err != nil {
		return fmt.Errorf("revoke user sessions (scan refresh): %w", err)
	}

	return nil
}

// scanKeys iterates all Redis keys matching pattern via SCAN, invoking fn for each.
func (m *JWTMaker) scanKeys(ctx context.Context, pattern string, fn func(key string)) error {
	var cursor uint64
	for {
		keys, next, err := m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		for _, k := range keys {
			fn(k)
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

// StoreSession persists a session marker in Redis with the given TTL.
func (m *JWTMaker) StoreSession(ctx context.Context, sessionID, payload string, ttl time.Duration) error {
	return m.redis.Set(ctx, m.sessionKey(sessionID), payload, ttl).Err()
}

// RevokeSession removes the session key from Redis, invalidating all access tokens
// that reference it.
func (m *JWTMaker) RevokeSession(ctx context.Context, sessionID string) error {
	return m.redis.Del(ctx, m.sessionKey(sessionID)).Err()
}

// sign creates and signs a JWT with HS256.
func (m *JWTMaker) sign(claims jwt.MapClaims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.signingKey)
}

// parse verifies signature, expiry, and token type.
func (m *JWTMaker) parse(tokenStr, wantType string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.signingKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token parse: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token: invalid")
	}

	if typ, _ := claims["type"].(string); typ != wantType {
		return nil, fmt.Errorf("token: expected type %q, got %q", wantType, typ)
	}

	return claims, nil
}

func (m *JWTMaker) refreshKey(jti string) string { return m.redisPrefix + ":refresh:" + jti }
func (m *JWTMaker) sessionKey(sid string) string { return m.redisPrefix + ":sess:" + sid }

// graceKey — ishlatilgan refresh JTI uchun idempotentlik yozuvi.
// Diqqat: refreshKey bilan bir xil `:refresh:` prefiksida (`refresh:used:<jti>`),
// shuning uchun refresh kalitlarini skanerlaydigan joylar ikkala shaklni ham
// hisobga olishi kerak — grace prefiksini gracePrefix() beradi.
func (m *JWTMaker) graceKey(jti string) string { return m.gracePrefix() + jti }
func (m *JWTMaker) gracePrefix() string        { return m.redisPrefix + ":refresh:used:" }
