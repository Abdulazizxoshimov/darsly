package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/token"
)

// ─── LiveKit (mock) ──────────────────────────────────────────────────────────

// FakeLiveKit — room/chat/recording usecase'larining LiveKit interfeyslarini
// qondiradi (xatti-harakatni assert qilish uchun, real serverga ulanmasdan).
type FakeLiveKit struct {
	mu           sync.Mutex
	IsEnabled    bool
	Participants []entity.RoomParticipant
	Calls        map[string]int
	// Sent — data-channel orqali tarqatilgan xabarlar (roomstate/chat testlari
	// "nima yuborildi" ni AYNAN tekshiradi: xabar shakli klient bilan shartnoma).
	Sent [][]byte
	// SentTo — har `SendDataTo` chaqiruvining manzil ro'yxati (`Sent` bilan
	// parallel emas: faqat manzilli yuborishlar).
	SentTo [][]string
	// TokenRoom — VerifyToken qaytaradigan xona nomi.
	TokenRoom string
	// Removed — `RemoveParticipant` bilan uzilgan identity'lar (ban qo'llanishini
	// tekshirish uchun: "chaqirildi" emas, "AYNAN KIM uzildi" muhim).
	Removed []string
	// Muted — `MuteParticipant` bilan mute qilingan identity'lar (ovoz siyosati
	// testlari "AYNAN KIM mute bo'ldi"ni tekshiradi).
	Muted []string
	// EgressErr — nil bo'lmasa `StartRoomRecording` shu xatoni qaytaradi.
	EgressErr error
}

func NewFakeLiveKit() *FakeLiveKit {
	return &FakeLiveKit{IsEnabled: true, Calls: map[string]int{}}
}

func (f *FakeLiveKit) inc(op string) {
	f.mu.Lock()
	f.Calls[op]++
	f.mu.Unlock()
}

func (f *FakeLiveKit) Enabled() bool { return f.IsEnabled }
func (f *FakeLiveKit) ClientWSURL(string) string { return "ws://fake-livekit" }
func (f *FakeLiveKit) AccessToken(_, identity, _ string, isHost bool) (string, error) {
	f.inc("AccessToken")
	if isHost {
		return "host-token-" + identity, nil
	}
	return "part-token-" + identity, nil
}
func (f *FakeLiveKit) EnsureRoom(_ context.Context, _ string, _ uint32) error {
	f.inc("EnsureRoom")
	return nil
}
func (f *FakeLiveKit) DeleteRoom(_ context.Context, _ string) error {
	f.inc("DeleteRoom")
	return nil
}
func (f *FakeLiveKit) ListParticipantViews(_ context.Context, _ string) ([]entity.RoomParticipant, error) {
	f.inc("ListParticipantViews")
	return f.Participants, nil
}
func (f *FakeLiveKit) MuteParticipant(_ context.Context, _, identity string, _ bool) error {
	f.inc("MuteParticipant")
	f.mu.Lock()
	f.Muted = append(f.Muted, identity)
	f.mu.Unlock()
	return nil
}

// MutedList — mute qilingan identity'lar nusxasi (poyga'siz o'qish uchun).
func (f *FakeLiveKit) MutedList() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.Muted...)
}
func (f *FakeLiveKit) RemoveParticipant(_ context.Context, _, identity string) error {
	f.inc("RemoveParticipant")
	f.mu.Lock()
	f.Removed = append(f.Removed, identity)
	f.mu.Unlock()
	return nil
}

// RemovedList — uzilgan identity'lar nusxasi (poyga'siz o'qish uchun).
func (f *FakeLiveKit) RemovedList() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.Removed...)
}
func (f *FakeLiveKit) SetParticipantPublish(_ context.Context, _, _ string, _ bool) error {
	f.inc("SetParticipantPublish")
	return nil
}
func (f *FakeLiveKit) SendData(_ context.Context, _ string, data []byte) error {
	f.inc("SendData")
	f.mu.Lock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.Sent = append(f.Sent, cp)
	f.mu.Unlock()
	return nil
}

// SendDataTo — manzilli yuborish. Kimga ketgani testda tekshirilsin deb
// `SentTo` ga yoziladi (shaxsiy xabar SIZMASLIGI shu bilan isbotlanadi).
func (f *FakeLiveKit) SendDataTo(_ context.Context, _ string, data []byte, identities []string) error {
	f.inc("SendDataTo")
	f.mu.Lock()
	cp := make([]byte, len(data))
	copy(cp, data)
	f.Sent = append(f.Sent, cp)
	ids := make([]string, len(identities))
	copy(ids, identities)
	f.SentTo = append(f.SentTo, ids)
	f.mu.Unlock()
	return nil
}

// VerifyToken — `AccessToken` ning teskarisi. Fake token shakli:
// "host-token-<identity>" yoki "part-token-<identity>". Xona nomi testda
// tekshirilishi uchun `SetTokenRoom` bilan beriladi (default: bo'sh).
func (f *FakeLiveKit) VerifyToken(token string) (identity, name, room string, err error) {
	f.inc("VerifyToken")
	for _, prefix := range []string{"host-token-", "part-token-"} {
		if strings.HasPrefix(token, prefix) {
			id := strings.TrimPrefix(token, prefix)
			return id, id, f.TokenRoom, nil
		}
	}
	return "", "", "", errors.New("invalid token")
}

// SetTokenRoom — VerifyToken qaytaradigan xona nomini belgilaydi.
func (f *FakeLiveKit) SetTokenRoom(room string) {
	f.mu.Lock()
	f.TokenRoom = room
	f.mu.Unlock()
}
// EgressErr o'rnatilgan bo'lsa `StartRoomRecording` shu xato bilan yiqiladi —
// "egress boshlanmadi" yo'lini (kompensatsiya, qulf bo'shatish) sinash uchun.
func (f *FakeLiveKit) StartRoomRecording(_ context.Context, _, _ string, _ livekit.S3Config) (string, error) {
	f.inc("StartRoomRecording")
	f.mu.Lock()
	err := f.EgressErr
	f.mu.Unlock()
	if err != nil {
		return "", err
	}
	return "egress-fake", nil
}
func (f *FakeLiveKit) StopRecording(_ context.Context, _ string) error {
	f.inc("StopRecording")
	return nil
}

// ─── UserRepo ────────────────────────────────────────────────────────────────

type FakeUserRepo struct {
	mu    sync.Mutex
	byID  map[string]*entity.User
	Calls map[string]int
}

func NewFakeUserRepo() *FakeUserRepo {
	return &FakeUserRepo{byID: map[string]*entity.User{}, Calls: map[string]int{}}
}

func (r *FakeUserRepo) Create(_ context.Context, u *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls["Create"]++
	for _, e := range r.byID {
		if e.Email == u.Email && e.DeletedAt == nil {
			return apperr.Conflict("email already exists")
		}
	}
	cp := *u
	r.byID[u.ID] = &cp
	return nil
}

func (r *FakeUserRepo) GetByID(_ context.Context, id string) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok && u.DeletedAt == nil {
		cp := *u
		return &cp, nil
	}
	return nil, apperr.NotFound("user")
}

func (r *FakeUserRepo) GetByEmail(_ context.Context, email string) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.byID {
		if u.Email == email && u.DeletedAt == nil {
			cp := *u
			return &cp, nil
		}
	}
	return nil, apperr.NotFound("user")
}

func (r *FakeUserRepo) List(_ context.Context, _ *entity.UserFilter) ([]*entity.User, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.User
	for _, u := range r.byID {
		if u.DeletedAt == nil {
			cp := *u
			out = append(out, &cp)
		}
	}
	return out, len(out), nil
}

func (r *FakeUserRepo) Update(_ context.Context, u *entity.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[u.ID]; !ok {
		return apperr.NotFound("user")
	}
	cp := *u
	r.byID[u.ID] = &cp
	return nil
}

func (r *FakeUserRepo) UpdatePassword(_ context.Context, userID, hash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[userID]; ok {
		u.PasswordHash = hash
		return nil
	}
	return apperr.NotFound("user")
}

func (r *FakeUserRepo) UpdateLastLogin(_ context.Context, _ string) error { return nil }

func (r *FakeUserRepo) SoftDelete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.byID[id]; ok {
		now := time.Now()
		u.DeletedAt = &now
	}
	return nil
}

func (r *FakeUserRepo) DeleteHard(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls["DeleteHard"]++
	delete(r.byID, id)
	return nil
}

func (r *FakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.byID {
		if u.Email == email && u.DeletedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

// ─── AuthRepo ────────────────────────────────────────────────────────────────

type FakeAuthRepo struct {
	mu                  sync.Mutex
	tokens              map[string]*entity.RefreshToken // by hash
	resets              map[string]*entity.PasswordReset
	FailNextCreateToken bool
	Calls               map[string]int
}

func NewFakeAuthRepo() *FakeAuthRepo {
	return &FakeAuthRepo{tokens: map[string]*entity.RefreshToken{}, resets: map[string]*entity.PasswordReset{}, Calls: map[string]int{}}
}

func (r *FakeAuthRepo) CreateRefreshToken(_ context.Context, rt *entity.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls["CreateRefreshToken"]++
	if r.FailNextCreateToken {
		r.FailNextCreateToken = false
		return apperr.Internal(context.DeadlineExceeded)
	}
	cp := *rt
	r.tokens[rt.TokenHash] = &cp
	return nil
}

func (r *FakeAuthRepo) GetRefreshTokenByHash(_ context.Context, hash string) (*entity.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rt, ok := r.tokens[hash]; ok {
		cp := *rt
		return &cp, nil
	}
	return nil, apperr.NotFound("refresh token")
}

func (r *FakeAuthRepo) RevokeRefreshToken(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls["RevokeRefreshToken"]++
	for h, rt := range r.tokens {
		if rt.ID == id {
			delete(r.tokens, h)
		}
	}
	return nil
}

func (r *FakeAuthRepo) RevokeAllUserTokens(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls["RevokeAllUserTokens"]++
	for h, rt := range r.tokens {
		if rt.UserID == userID {
			delete(r.tokens, h)
		}
	}
	return nil
}
func (r *FakeAuthRepo) DeleteExpiredTokens(_ context.Context) error { return nil }

func (r *FakeAuthRepo) CreatePasswordReset(_ context.Context, pr *entity.PasswordReset) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *pr
	r.resets[pr.TokenHash] = &cp
	return nil
}

func (r *FakeAuthRepo) GetPasswordResetByHash(_ context.Context, hash string) (*entity.PasswordReset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pr, ok := r.resets[hash]; ok {
		cp := *pr
		return &cp, nil
	}
	return nil, apperr.NotFound("password reset")
}

func (r *FakeAuthRepo) MarkPasswordResetUsed(_ context.Context, _ string) error { return nil }
func (r *FakeAuthRepo) DeleteExpiredPasswordResets(_ context.Context) error     { return nil }

// ─── TokenMaker ──────────────────────────────────────────────────────────────

type FakeTokenMaker struct {
	mu           sync.Mutex
	Sessions     map[string]bool // sessionID → active
	Revoked      []string        // refresh tokens passed to RevokeRefresh
	RevokedUsers []string        // userIDs passed to RevokeAllUserSessions
	// SessionUser — sessionID → userID (StoreSession payload'idan). "Bitta akkaunt
	// = bitta faol sessiya" testi uchun kerak: `RevokeUserSessionsExcept` qaysi
	// sessiyalar shu foydalanuvchiniki ekanini shu xarita orqali biladi.
	SessionUser map[string]string
	// RevokedExceptUsers — `RevokeUserSessionsExcept` chaqirilgan userID'lar.
	RevokedExceptUsers []string
	// RotateErr — nil bo'lmasa `Rotate` shu xatoni qaytaradi. Refresh xatosining
	// TURI usecase'da farqlanadi (`SESSION_REVOKED` va boshqalar), shuning uchun
	// testda uni boshqarish kerak.
	RotateErr error
	counter   int
}

func NewFakeTokenMaker() *FakeTokenMaker {
	return &FakeTokenMaker{Sessions: map[string]bool{}, SessionUser: map[string]string{}}
}

func (m *FakeTokenMaker) Generate(_ context.Context, _, sessionID, _ string) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counter++
	return "access-tok", "refresh-tok-" + sessionID, nil
}
func (m *FakeTokenMaker) ValidateAccess(_ context.Context, _ string) (*token.Claims, error) {
	return &token.Claims{}, nil
}
func (m *FakeTokenMaker) Rotate(_ context.Context, _ string) (string, string, error) {
	m.mu.Lock()
	err := m.RotateErr
	m.mu.Unlock()
	if err != nil {
		return "", "", err
	}
	return "new-access", "new-refresh", nil
}
func (m *FakeTokenMaker) SessionFromRefresh(refreshToken string) (string, error) {
	// Fake'da token "refresh-tok-<sessionID>" shaklida (Generate'ga qara).
	sid, ok := strings.CutPrefix(refreshToken, "refresh-tok-")
	if !ok || sid == "" {
		return "", errors.New("fake: invalid refresh token")
	}
	return sid, nil
}
func (m *FakeTokenMaker) Revoke(_ context.Context, _ string) error { return nil }
func (m *FakeTokenMaker) RevokeRefresh(_ context.Context, refreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Revoked = append(m.Revoked, refreshToken)
	return nil
}
func (m *FakeTokenMaker) RevokeAllUserSessions(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RevokedUsers = append(m.RevokedUsers, userID)
	return nil
}
func (m *FakeTokenMaker) RevokeUserSessionsExcept(_ context.Context, userID, keepSessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RevokedExceptUsers = append(m.RevokedExceptUsers, userID)
	for sid, uid := range m.SessionUser {
		if uid == userID && sid != keepSessionID {
			delete(m.Sessions, sid)
		}
	}
	return nil
}
func (m *FakeTokenMaker) StoreSession(_ context.Context, sessionID, payload string, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sessions[sessionID] = true
	if m.SessionUser == nil {
		m.SessionUser = map[string]string{}
	}
	m.SessionUser[sessionID] = payload
	return nil
}
func (m *FakeTokenMaker) RevokeSession(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Sessions, sessionID)
	return nil
}

// ─── Cache (in-memory) ───────────────────────────────────────────────────────

type FakeCache struct {
	mu       sync.Mutex
	kv       map[string]string
	counters map[string]int64
	// hashes — HSet/HGetAll/HDel uchun HAQIQIY xulq. Avval bu uchtasi no-op edi
	// (HSet nil, HGetAll nil), ya'ni hash'ga tayanadigan kod testda "hech narsa
	// saqlanmaydi" holatida sinalardi va bu jimgina yolg'on qamrov berardi.
	hashes map[string]map[string]string
}

func NewFakeCache() *FakeCache {
	return &FakeCache{
		kv:       map[string]string{},
		counters: map[string]int64{},
		hashes:   map[string]map[string]string{},
	}
}

// Set — `redis.redisCache.Set` bilan AYNAN bir xil: satr xom, qolgani JSON.
// Bu ikkisi ajralib qolsa test yashil bo'lib turib real Redis'dagi xatoni
// yashiradi (aynan shu bir marta sodir bo'lgan — izohi `redis/cache.go` da).
func (c *FakeCache) Set(_ context.Context, key string, value any, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch v := value.(type) {
	case string:
		c.kv[key] = v
	default:
		b, _ := json.Marshal(value)
		c.kv[key] = string(b)
	}
	return nil
}
func (c *FakeCache) Get(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.kv[key]; ok {
		return v, nil
	}
	return "", goredis.Nil
}
func (c *FakeCache) Del(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range keys {
		delete(c.kv, k)
		delete(c.counters, k)
		delete(c.hashes, k) // Redis'da DEL hash'ni ham o'chiradi
	}
	return nil
}
// SetNX — HAQIQIY "faqat mavjud bo'lmasa yoz" semantikasi.
//
// Avval bu doim `true` qaytarardi, ya'ni SetNX'ga tayanadigan har qanday qulf
// testda "hech qachon band emas" holatida sinalardi — dublikat-egress qulfi
// (`recording.startLockKey`) kabi poyga himoyalari yashil bo'lib turib
// umuman qoplanmagan bo'lardi.
func (c *FakeCache) SetNX(_ context.Context, key, value string, _ time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.kv[key]; exists {
		return false, nil
	}
	c.kv[key] = value
	return true, nil
}
func (c *FakeCache) Incr(_ context.Context, key string, _ time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := c.counters[key] + 1
	c.counters[key] = n
	return n, nil
}
func (c *FakeCache) Ping(_ context.Context) error                       { return nil }
func (c *FakeCache) Keys(_ context.Context, _ string) ([]string, error) { return nil, nil }
func (c *FakeCache) Scan(_ context.Context, _ uint64, _ string, _ int64) ([]string, uint64, error) {
	return nil, 0, nil
}
func (c *FakeCache) ScanDel(_ context.Context, _ string) error             { return nil }
func (c *FakeCache) MGet(_ context.Context, _ ...string) ([]string, error) { return nil, nil }
func (c *FakeCache) HSet(_ context.Context, key string, values map[string]any, _ time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	h, ok := c.hashes[key]
	if !ok {
		h = map[string]string{}
		c.hashes[key] = h
	}
	for f, v := range values {
		if s, isStr := v.(string); isStr {
			h[f] = s
			continue
		}
		b, _ := json.Marshal(v)
		h[f] = string(b)
	}
	return nil
}

// HGetAll — bo'sh/yo'q hash uchun XATO qaytaradi, xuddi `redisCache` kabi.
// Fake haqiqiy implementatsiyadan farq qilsa, test yashil bo'lib turib
// production'da yiqiladigan kodni o'tkazib yuborardi.
func (c *FakeCache) HGetAll(_ context.Context, key string) (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	h, ok := c.hashes[key]
	if !ok || len(h) == 0 {
		return nil, goredis.Nil
	}
	out := make(map[string]string, len(h))
	for f, v := range h {
		out[f] = v
	}
	return out, nil
}

func (c *FakeCache) HDel(_ context.Context, key string, fields ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	h, ok := c.hashes[key]
	if !ok {
		return nil
	}
	for _, f := range fields {
		delete(h, f)
	}
	return nil
}
func (c *FakeCache) Publish(_ context.Context, _ string, _ any) error         { return nil }
func (c *FakeCache) Subscribe(_ context.Context, _ ...string) *goredis.PubSub { return nil }
func (c *FakeCache) AcquireLock(_ context.Context, _, _ string, _ time.Duration) (bool, error) {
	return true, nil
}
func (c *FakeCache) ReleaseLock(_ context.Context, _, _ string) error { return nil }
func (c *FakeCache) Eval(_ context.Context, _ string, _ []string, _ ...any) (any, error) {
	return nil, nil
}
func (c *FakeCache) Client() *goredis.Client { return nil }

// ─── Minio ───────────────────────────────────────────────────────────────────

type FakeMinio struct {
	Objects map[string]bool
	// Contents — HAQIQIY bayt mazmuni. Ko'p testga kerak emas (obyekt bor-yo'qligi
	// yetarli), lekin transkod ishchisi faylni ffmpeg'ga uzatadi va u yerda
	// mazmun ahamiyatli. To'ldirilgan bo'lsa `Get` shu baytlarni beradi.
	Contents map[string][]byte
}

func NewFakeMinio() *FakeMinio {
	return &FakeMinio{Objects: map[string]bool{}, Contents: map[string][]byte{}}
}

// PutFile — diskdagi faylni obyekt sifatida joylaydi (transkod testlari uchun).
func (m *FakeMinio) PutFile(name, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	m.Objects[name] = true
	m.Contents[name] = b
	return nil
}

func (m *FakeMinio) Upload(_ context.Context, name, _ string, r io.Reader, _ int64) (string, error) {
	m.Objects[name] = true
	if r != nil {
		if b, err := io.ReadAll(r); err == nil {
			m.Contents[name] = b
		}
	}
	return name, nil
}
func (m *FakeMinio) PresignedURL(_ context.Context, name string, _ time.Duration) (string, error) {
	return "https://minio.test/" + name + "?sig=fake", nil
}
func (m *FakeMinio) Get(_ context.Context, name string) (io.ReadCloser, error) {
	if !m.Objects[name] {
		return nil, fmt.Errorf("minio(fake): %q not found", name)
	}
	if b, ok := m.Contents[name]; ok {
		return io.NopCloser(bytes.NewReader(b)), nil
	}
	// Mazmuni ahamiyatsiz — testlar faqat oqim borligini tekshiradi.
	return io.NopCloser(strings.NewReader("fake-video")), nil
}

func (m *FakeMinio) Delete(_ context.Context, name string) error {
	delete(m.Objects, name)
	delete(m.Contents, name)
	return nil
}
func (m *FakeMinio) EnsureBucket(_ context.Context) error        { return nil }

// ─── RoomUseCase ─────────────────────────────────────────────────────────────

type FakeRoomUC struct {
	ParticipantCalls int
	// EnforcedJoins — `EnforceJoin` chaqiruvlari: "room|identity|name".
	EnforcedJoins []string
	// AudioPolicies — `EnforceAudioPolicy` chaqiruvlari: "room|identity".
	AudioPolicies []string
	// EmptyNoted / OccupiedNoted — avto-yakun uchun bo'shlik signallari (xona nomlari).
	EmptyNoted    []string
	OccupiedNoted []string
}

func (r *FakeRoomUC) HostToken(_ context.Context, _, _ string) (*entity.RoomToken, error) {
	return &entity.RoomToken{Token: "host-tok", Role: entity.RoomRoleHost}, nil
}
func (r *FakeRoomUC) ParticipantToken(_ context.Context, _ *entity.Lesson, identity, name string) (*entity.RoomToken, error) {
	r.ParticipantCalls++
	return &entity.RoomToken{Token: "part-tok", Identity: identity, Role: entity.RoomRoleParticipant}, nil
}
func (r *FakeRoomUC) EndLesson(_ context.Context, _, _ string) error { return nil }
func (r *FakeRoomUC) EnforceJoin(_ context.Context, roomName, identity, name string) {
	r.EnforcedJoins = append(r.EnforcedJoins, roomName+"|"+identity+"|"+name)
}
func (r *FakeRoomUC) EnforceAudioPolicy(_ context.Context, roomName, identity string) {
	r.AudioPolicies = append(r.AudioPolicies, roomName+"|"+identity)
}
func (r *FakeRoomUC) ListParticipants(_ context.Context, _, _ string) ([]entity.RoomParticipant, error) {
	return nil, nil
}
func (r *FakeRoomUC) MuteParticipant(_ context.Context, _, _, _ string, _ bool) error { return nil }
func (r *FakeRoomUC) MuteAll(_ context.Context, _, _ string, _ *bool) error           { return nil }
func (r *FakeRoomUC) RemoveParticipant(_ context.Context, _, _, _, _ string) error    { return nil }
func (r *FakeRoomUC) SetSpeakPermission(_ context.Context, _, _, _ string, _ bool) error {
	return nil
}
func (r *FakeRoomUC) ListBlocklist(_ context.Context, _ string) ([]*entity.BlocklistEntry, error) {
	return nil, nil
}
func (r *FakeRoomUC) Unblock(_ context.Context, _, _ string) error { return nil }
func (r *FakeRoomUC) NoteRoomEmpty(_ context.Context, roomName string) {
	r.EmptyNoted = append(r.EmptyNoted, roomName)
}
func (r *FakeRoomUC) NoteRoomOccupied(_ context.Context, roomName string) {
	r.OccupiedNoted = append(r.OccupiedNoted, roomName)
}
func (r *FakeRoomUC) SweepAutoEnd(_ context.Context, _, _ time.Duration) int { return 0 }

// ─── LessonRepo ──────────────────────────────────────────────────────────────

type FakeLessonRepo struct {
	mu        sync.Mutex
	byID      map[string]*entity.Lesson
	takenSlug map[string]bool // SlugExists uchun majburiy collision
	reminded  map[string]bool // reminder_sent_at kuzatuvi (entity'da maydon yo'q)
}

func NewFakeLessonRepo() *FakeLessonRepo {
	return &FakeLessonRepo{byID: map[string]*entity.Lesson{}, takenSlug: map[string]bool{}, reminded: map[string]bool{}}
}

// ReserveSlug keyingi SlugExists chaqiruvida `slug`ni band deb ko'rsatadi (collision testi).
func (r *FakeLessonRepo) ReserveSlug(slug string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.takenSlug[slug] = true
}

func (r *FakeLessonRepo) Create(_ context.Context, l *entity.Lesson) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *l
	r.byID[l.ID] = &cp
	r.takenSlug[l.JoinSlug] = true
	return nil
}
func (r *FakeLessonRepo) GetByID(_ context.Context, id string) (*entity.Lesson, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.byID[id]; ok && l.DeletedAt == nil {
		cp := *l
		return &cp, nil
	}
	return nil, apperr.NotFound("lesson")
}
func (r *FakeLessonRepo) GetBySlug(_ context.Context, slug string) (*entity.Lesson, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.byID {
		if l.JoinSlug == slug && l.DeletedAt == nil {
			cp := *l
			return &cp, nil
		}
	}
	return nil, apperr.NotFound("lesson")
}
func (r *FakeLessonRepo) SlugExists(_ context.Context, slug string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.takenSlug[slug], nil
}
func (r *FakeLessonRepo) ListByMentor(_ context.Context, mentorID string, _ *entity.LessonFilter) ([]*entity.Lesson, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Lesson
	for _, l := range r.byID {
		if l.MentorID == mentorID && l.DeletedAt == nil {
			cp := *l
			out = append(out, &cp)
		}
	}
	return out, len(out), nil
}
func (r *FakeLessonRepo) Update(_ context.Context, l *entity.Lesson) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *l
	r.byID[l.ID] = &cp
	return nil
}
func (r *FakeLessonRepo) SoftDelete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.byID[id]; ok {
		now := time.Now()
		l.DeletedAt = &now
	}
	return nil
}

// ListLive — jonli darslar (avto-yakun ishchisi/sweep testlari uchun).
func (r *FakeLessonRepo) ListLive(_ context.Context) ([]*entity.Lesson, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Lesson
	for _, l := range r.byID {
		if l.DeletedAt == nil && l.Status == entity.LessonStatusLive {
			cp := *l
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ClaimEnd — atomik yakunlash imitatsiyasi: faqat dars hamon `live` bo'lsa true.
func (r *FakeLessonRepo) ClaimEnd(_ context.Context, id string, endedAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.byID[id]
	if !ok || l.DeletedAt != nil || l.Status != entity.LessonStatusLive {
		return false, nil
	}
	l.Status = entity.LessonStatusEnded
	ended := endedAt
	l.EndedAt = &ended
	return true, nil
}

func (r *FakeLessonRepo) ListUpcomingUnreminded(_ context.Context, from, to time.Time) ([]*entity.Lesson, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Lesson
	for _, l := range r.byID {
		if l.DeletedAt == nil && !r.reminded[l.ID] && l.Status == entity.LessonStatusScheduled &&
			l.ScheduledAt != nil && !l.ScheduledAt.Before(from) && !l.ScheduledAt.After(to) {
			cp := *l
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *FakeLessonRepo) ClaimReminder(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[id]; !ok || r.reminded[id] {
		return false, nil
	}
	r.reminded[id] = true
	return true, nil
}

// ─── BlocklistRepo ───────────────────────────────────────────────────────────

// FakeBlocklistRepo — mentor darajasidagi doimiy qora ro'yxat (in-memory).
// Ism mosligi haqiqiy repo bilan AYNAN bir xil: lower(trim(display_name)).
type FakeBlocklistRepo struct {
	mu      sync.Mutex
	entries map[string]*entity.BlocklistEntry // by ID
	// FailNext — nil bo'lmasa keyingi chaqiruv shu xato bilan yiqiladi
	// (fail-open yo'lini sinash uchun).
	FailNext error
}

func NewFakeBlocklistRepo() *FakeBlocklistRepo {
	return &FakeBlocklistRepo{entries: map[string]*entity.BlocklistEntry{}}
}

func normalizeBlockName(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (r *FakeBlocklistRepo) takeErr() error {
	err := r.FailNext
	r.FailNext = nil
	return err
}

func (r *FakeBlocklistRepo) Add(_ context.Context, e *entity.BlocklistEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.takeErr(); err != nil {
		return err
	}
	// ON CONFLICT DO NOTHING taqlidi: bir mentor uchun bir ism bir marta.
	if name := normalizeBlockName(e.DisplayName); name != "" {
		for _, x := range r.entries {
			if x.MentorID == e.MentorID && normalizeBlockName(x.DisplayName) == name {
				return nil
			}
		}
	}
	cp := *e
	cp.DisplayName = strings.TrimSpace(e.DisplayName)
	cp.CreatedAt = time.Now()
	r.entries[e.ID] = &cp
	return nil
}

func (r *FakeBlocklistRepo) IsBlocked(_ context.Context, mentorID, displayName string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.takeErr(); err != nil {
		return false, err
	}
	name := normalizeBlockName(displayName)
	if name == "" {
		return false, nil
	}
	for _, e := range r.entries {
		if e.MentorID == mentorID && normalizeBlockName(e.DisplayName) == name {
			return true, nil
		}
	}
	return false, nil
}

func (r *FakeBlocklistRepo) ListByMentor(_ context.Context, mentorID string) ([]*entity.BlocklistEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.takeErr(); err != nil {
		return nil, err
	}
	var out []*entity.BlocklistEntry
	for _, e := range r.entries {
		if e.MentorID == mentorID {
			cp := *e
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *FakeBlocklistRepo) Delete(_ context.Context, mentorID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.takeErr(); err != nil {
		return err
	}
	if e, ok := r.entries[id]; ok && e.MentorID == mentorID {
		delete(r.entries, id)
		return nil
	}
	return apperr.NotFound("blocklist entry")
}

// ─── WaitingRoomRepo ─────────────────────────────────────────────────────────

type FakeWaitingRepo struct {
	mu   sync.Mutex
	byID map[string]*entity.WaitingRoomRequest
	// LessonMentor — lessonID→mentorID (ListPendingByMentor filtri uchun). Bo'sh bo'lsa
	// snapshot barcha pending'ni qaytaradi (test soddaligi uchun).
	LessonMentor map[string]string
}

func NewFakeWaitingRepo() *FakeWaitingRepo {
	return &FakeWaitingRepo{byID: map[string]*entity.WaitingRoomRequest{}, LessonMentor: map[string]string{}}
}

func (r *FakeWaitingRepo) Create(_ context.Context, w *entity.WaitingRoomRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *w
	r.byID[w.ID] = &cp
	return nil
}
func (r *FakeWaitingRepo) GetByID(_ context.Context, id string) (*entity.WaitingRoomRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if w, ok := r.byID[id]; ok {
		cp := *w
		return &cp, nil
	}
	return nil, apperr.NotFound("waiting room request")
}
func (r *FakeWaitingRepo) ListPending(_ context.Context, lessonID string) ([]*entity.WaitingRoomRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.WaitingRoomRequest
	for _, w := range r.byID {
		if w.LessonID == lessonID && w.Status == entity.WaitingStatusPending {
			cp := *w
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *FakeWaitingRepo) ListPendingByMentor(_ context.Context, mentorID string) ([]*entity.WaitingRoomRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.WaitingRoomRequest
	for _, w := range r.byID {
		if w.Status != entity.WaitingStatusPending {
			continue
		}
		if len(r.LessonMentor) > 0 && r.LessonMentor[w.LessonID] != mentorID {
			continue
		}
		cp := *w
		out = append(out, &cp)
	}
	return out, nil
}

func (r *FakeWaitingRepo) TransitionFromPending(_ context.Context, id, newStatus string, decidedAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.byID[id]
	if !ok || w.Status != entity.WaitingStatusPending {
		return false, nil
	}
	w.Status = newStatus
	w.DecidedAt = &decidedAt
	return true, nil
}

// ─── NotificationRepo ────────────────────────────────────────────────────────

type FakeNotifRepo struct {
	mu    sync.Mutex
	items []*entity.Notification
}

func NewFakeNotifRepo() *FakeNotifRepo { return &FakeNotifRepo{} }

func (r *FakeNotifRepo) Create(_ context.Context, n *entity.Notification) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *n
	r.items = append(r.items, &cp)
	return nil
}
func (r *FakeNotifRepo) ListByUser(_ context.Context, userID string, f *entity.NotificationFilter) ([]*entity.Notification, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Notification
	for _, n := range r.items {
		if n.UserID == userID && (!f.UnreadOnly || n.ReadAt == nil) {
			cp := *n
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, len(out), nil
}
func (r *FakeNotifRepo) UnreadCount(_ context.Context, userID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, x := range r.items {
		if x.UserID == userID && x.ReadAt == nil {
			n++
		}
	}
	return n, nil
}
func (r *FakeNotifRepo) MarkRead(_ context.Context, id, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range r.items {
		if n.ID == id && n.UserID == userID && n.ReadAt == nil {
			now := time.Now()
			n.ReadAt = &now
		}
	}
	return nil
}
func (r *FakeNotifRepo) MarkAllRead(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, n := range r.items {
		if n.UserID == userID && n.ReadAt == nil {
			n.ReadAt = &now
		}
	}
	return nil
}

// ─── RecordingRepo ───────────────────────────────────────────────────────────

type FakeRecordingRepo struct {
	mu           sync.Mutex
	byID         map[string]*entity.Recording
	transcode    map[string]string
	originalSize map[string]int64
	// warned — `retention_warned_at` kuzatuvi (entity'da maydon yo'q, xuddi
	// FakeLessonRepo.reminded kabi).
	warned map[string]bool
	// tgNext — `telegram_next_attempt_at` (Telegram yuklash navbati). Entity'da
	// yo'q, chunki u sof ichki navbat maydoni va klientga chiqmaydi.
	tgNext map[string]time.Time
}

func NewFakeRecordingRepo() *FakeRecordingRepo {
	return &FakeRecordingRepo{
		byID:         map[string]*entity.Recording{},
		transcode:    map[string]string{},
		originalSize: map[string]int64{},
		warned:       map[string]bool{},
		tgNext:       map[string]time.Time{},
	}
}

// ─── Retention (PRODUCT.md №5) ───────────────────────────────────────────────

// recordingEnd — retention hisobining tayanch vaqti (postgres'dagi
// COALESCE(ended_at, created_at) bilan bir xil).
func recordingEnd(rec *entity.Recording) time.Time {
	if rec.EndedAt != nil {
		return *rec.EndedAt
	}
	return rec.CreatedAt
}

func (r *FakeRecordingRepo) ListExpired(_ context.Context, endedBefore time.Time, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == entity.RecordingStatusReady && !recordingEnd(rec).After(endedBefore) {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ListArchivable — postgres bilan bir xil: Telegramda TASDIQLANGAN va kesh
// BO'LMAGAN yozuvlar.
//
// Filtrni fake'da ham takrorlash SHART: aks holda test «tasdiqlanmagan yozuv
// o'chmaydi» degan asosiy kafolatni tekshira olmasdi — fake hammasini
// qaytarib, ishchi ularni o'chirib yuborardi va test baribir yashil bo'lardi.
func (r *FakeRecordingRepo) ListArchivable(_ context.Context, endedBefore time.Time, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == entity.RecordingStatusReady && rec.TelegramSentAt != nil &&
			rec.CachedUntil == nil && !recordingEnd(rec).After(endedBefore) {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *FakeRecordingRepo) ListExpiredUnconfirmed(_ context.Context, endedBefore time.Time, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == entity.RecordingStatusReady && rec.TelegramSentAt == nil &&
			rec.CachedUntil == nil && !recordingEnd(rec).After(endedBefore) {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *FakeRecordingRepo) ListExpiringUnwarned(_ context.Context, endedBefore time.Time, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == entity.RecordingStatusReady && !r.warned[rec.ID] && !recordingEnd(rec).After(endedBefore) {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}

// ClaimExpire — atomik `ready → expired` imitatsiyasi. `deleted_at` entity'da
// yo'q (faqat DB ustuni), shuning uchun fake'da holatning o'zi kifoya.
func (r *FakeRecordingRepo) ClaimExpire(_ context.Context, id string, _ time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusReady {
		return false, nil
	}
	rec.Status = entity.RecordingStatusExpired
	return true, nil
}

// ClaimArchive — `ready → archived`, FAQAT Telegramda tasdiqlangan bo'lsa
// (postgres'dagi WHERE sharti bilan bir xil).
func (r *FakeRecordingRepo) ClaimArchive(_ context.Context, id string, _ time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusReady || rec.TelegramSentAt == nil {
		return false, nil
	}
	rec.Status = entity.RecordingStatusArchived
	return true, nil
}

// ─── Telegram arxivi ─────────────────────────────────────────────────────────

func (r *FakeRecordingRepo) EnqueueTelegram(_ context.Context, egressID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for _, rec := range r.byID {
		if rec.EgressID == egressID && rec.Status == entity.RecordingStatusReady && rec.TelegramSentAt == nil {
			r.tgNext[rec.ID] = now
		}
	}
	return nil
}

func (r *FakeRecordingRepo) ClaimTelegramUpload(_ context.Context, now time.Time, maxAttempts int) (*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Barqaror tartib — `ClaimTranscode` bilan bir xil sabab.
	ids := make([]string, 0, len(r.tgNext))
	for id, at := range r.tgNext {
		rec, ok := r.byID[id]
		if !ok || rec.Status != entity.RecordingStatusReady || rec.TelegramSentAt != nil {
			continue
		}
		if rec.TelegramAttempts >= maxAttempts || at.After(now) {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	sort.Strings(ids)
	rec := r.byID[ids[0]]
	rec.TelegramAttempts++
	r.tgNext[rec.ID] = now.Add(30 * time.Minute)
	cp := *rec
	// Chaqiruvchi (`handleFailure`) urinish tartibini `TelegramAttempts + 1`
	// deb hisoblaydi, ya'ni unga SO'ROVDAN OLDINGI qiymat kerak.
	cp.TelegramAttempts = rec.TelegramAttempts - 1
	return &cp, nil
}

func (r *FakeRecordingRepo) MarkTelegramSent(_ context.Context, id string, chatID, messageID int64, fileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	rec.TelegramChatID = &chatID
	rec.TelegramMessageID = &messageID
	rec.TelegramFileID = &fileID
	rec.TelegramSentAt = &now
	rec.TelegramError = nil
	delete(r.tgNext, id)
	return nil
}

func (r *FakeRecordingRepo) MarkTelegramFailed(_ context.Context, id, errMsg string, nextAttempt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok {
		return nil
	}
	msg := errMsg
	rec.TelegramError = &msg
	if nextAttempt == nil {
		delete(r.tgNext, id)
	} else {
		r.tgNext[id] = *nextAttempt
	}
	return nil
}

// ─── Tiklash ─────────────────────────────────────────────────────────────────

func (r *FakeRecordingRepo) ClaimRestore(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusArchived || rec.TelegramFileID == nil {
		return false, nil
	}
	rec.Status = entity.RecordingStatusRestoring
	return true, nil
}

func (r *FakeRecordingRepo) FinishRestore(_ context.Context, id, objectKey string, cachedUntil time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusRestoring {
		return nil
	}
	rec.Status = entity.RecordingStatusReady
	rec.ObjectKey = objectKey
	rec.CachedUntil = &cachedUntil
	return nil
}

func (r *FakeRecordingRepo) FailRestore(_ context.Context, id, errMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusRestoring {
		return nil
	}
	rec.Status = entity.RecordingStatusArchived
	msg := errMsg
	rec.TelegramError = &msg
	return nil
}

func (r *FakeRecordingRepo) ListByStatus(_ context.Context, status string, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == status {
			cp := *rec
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (r *FakeRecordingRepo) ListCacheExpired(_ context.Context, now time.Time, _ uint64) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.Status == entity.RecordingStatusReady && rec.CachedUntil != nil && rec.CachedUntil.Before(now) {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *FakeRecordingRepo) ClaimCacheEvict(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec, ok := r.byID[id]
	if !ok || rec.Status != entity.RecordingStatusReady || rec.CachedUntil == nil ||
		!rec.CachedUntil.Before(time.Now().UTC()) {
		return false, nil
	}
	rec.Status = entity.RecordingStatusArchived
	rec.CachedUntil = nil
	return true, nil
}

func (r *FakeRecordingRepo) ClaimRetentionWarning(_ context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.warned[id] {
		return false, nil
	}
	r.warned[id] = true
	return true, nil
}

func (r *FakeRecordingRepo) Create(_ context.Context, rec *entity.Recording) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rec
	r.byID[rec.ID] = &cp
	return nil
}
func (r *FakeRecordingRepo) GetByID(_ context.Context, id string) (*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.byID[id]; ok {
		cp := *rec
		return &cp, nil
	}
	return nil, apperr.NotFound("recording")
}
func (r *FakeRecordingRepo) GetByEgressID(_ context.Context, egressID string) (*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.byID {
		if rec.EgressID == egressID {
			cp := *rec
			return &cp, nil
		}
	}
	return nil, apperr.NotFound("recording")
}
func (r *FakeRecordingRepo) ListByLesson(_ context.Context, lessonID string) ([]*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Recording
	for _, rec := range r.byID {
		if rec.LessonID == lessonID {
			cp := *rec
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *FakeRecordingRepo) UpdateStatus(_ context.Context, id, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec, ok := r.byID[id]; ok {
		rec.Status = status
	}
	return nil
}
func (r *FakeRecordingRepo) MarkReady(_ context.Context, egressID, objectKey string, dur int, size int64, endedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.byID {
		if rec.EgressID == egressID {
			rec.Status = entity.RecordingStatusReady
			rec.ObjectKey = objectKey
			rec.DurationSec = dur
			rec.SizeBytes = size
			rec.EndedAt = &endedAt
		}
	}
	return nil
}

// ── Qayta kodlash navbati ────────────────────────────────────────────────────

func (r *FakeRecordingRepo) EnqueueTranscode(_ context.Context, egressID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.byID {
		if rec.EgressID == egressID && rec.Status == entity.RecordingStatusReady {
			r.transcode[rec.ID] = entity.TranscodePending
		}
	}
	return nil
}

func (r *FakeRecordingRepo) ClaimTranscode(_ context.Context) (*entity.Recording, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Barqaror tartib: xarita bo'ylab yurish tasodifiy va test "qaysi yozuv
	// olindi" degan savolga javob bera olmasdi.
	ids := make([]string, 0, len(r.transcode))
	for id, st := range r.transcode {
		if st == entity.TranscodePending {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	sort.Strings(ids)
	id := ids[0]
	r.transcode[id] = entity.TranscodeRunning
	cp := *r.byID[id]
	return &cp, nil
}

func (r *FakeRecordingRepo) FinishTranscode(_ context.Context, id string, newSize, originalSize int64, durationSec, trimmedSec int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transcode[id] = entity.TranscodeDone
	if rec, ok := r.byID[id]; ok {
		rec.SizeBytes = newSize
		if durationSec > 0 {
			rec.DurationSec = durationSec
		}
		// Postgres kabi QO'SHILADI, ustiga yozilmaydi — qayta transkodda
		// siljish to'planishi testda ham aynan shunday ko'rinsin.
		if trimmedSec > 0 {
			rec.ContentOffsetSec += trimmedSec
		}
	}
	r.originalSize[id] = originalSize
	return nil
}

func (r *FakeRecordingRepo) FailTranscode(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.transcode[id] = entity.TranscodeFailed
	return nil
}

func (r *FakeRecordingRepo) RequeueStaleTranscodes(_ context.Context, _ time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for id, st := range r.transcode {
		if st == entity.TranscodeRunning {
			r.transcode[id] = entity.TranscodePending
			n++
		}
	}
	return n, nil
}

// TranscodeStatus — test uchun holatni o'qish.
func (r *FakeRecordingRepo) TranscodeStatus(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.transcode[id]
}

func (r *FakeRecordingRepo) MarkFailed(_ context.Context, egressID string, endedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.byID {
		if rec.EgressID == egressID {
			rec.Status = entity.RecordingStatusFailed
			rec.EndedAt = &endedAt
		}
	}
	return nil
}

// ─── ChatRepo ────────────────────────────────────────────────────────────────

type FakeChatRepo struct {
	mu    sync.Mutex
	items []*entity.ChatMessage
	// deleted — moderatsiya bilan o'chirilgan xabar ID lari. Entity'da bunday
	// maydon YO'Q (o'chirilgan xabar API'da umuman ko'rinmaydi), shuning uchun
	// fake buni yonida tutadi — xuddi haqiqiy repo `deleted_at` ustunida
	// tutgani kabi.
	deleted map[string]string // messageID → deletedBy
}

func NewFakeChatRepo() *FakeChatRepo { return &FakeChatRepo{deleted: map[string]string{}} }

func (r *FakeChatRepo) Create(_ context.Context, m *entity.ChatMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *m
	if m.File != nil {
		f := *m.File
		cp.File = &f
	}
	r.items = append(r.items, &cp)
	return nil
}

// DeletedBy — testlar uchun: xabarni kim o'chirgan ("" = o'chirilmagan).
func (r *FakeChatRepo) DeletedBy(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.deleted[id]
}

func (r *FakeChatRepo) SoftDelete(_ context.Context, lessonID, messageID, deletedBy string) (*entity.ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, gone := r.deleted[messageID]; gone {
		return nil, apperr.NotFound("chat message")
	}
	for _, m := range r.items {
		if m.ID != messageID || m.LessonID != lessonID {
			continue
		}
		r.deleted[messageID] = deletedBy
		cp := *m
		return &cp, nil
	}
	return nil, apperr.NotFound("chat message")
}
func (r *FakeChatRepo) ListByLesson(_ context.Context, lessonID, viewerIdentity string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	// Eng yangidan eskiga (repo semantikasi), before kursoridan eski xabarlar.
	var out []*entity.ChatMessage
	for i := len(r.items) - 1; i >= 0; i-- {
		m := r.items[i]
		if m.LessonID != lessonID {
			continue
		}
		// O'CHIRILGAN xabar hech kimga qaytmaydi (haqiqiy repo SQL'da shunday).
		if _, gone := r.deleted[m.ID]; gone {
			continue
		}
		// KO'RINUVCHANLIK — haqiqiy repo bilan bir xil. Fake buni qilmasa,
		// "begona shaxsiy xabar sizib chiqmaydi" degan testni yozib bo'lmasdi.
		if m.ToIdentity != nil {
			if viewerIdentity == "" {
				continue
			}
			if *m.ToIdentity != viewerIdentity && m.SenderIdentity != viewerIdentity {
				continue
			}
		}
		if before != nil && !m.CreatedAt.Before(*before) {
			continue
		}
		cp := *m
		out = append(out, &cp)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// ListAllByLesson — ESKIDAN YANGIGA, kursorsiz (arxiv/transkript yo'li).
// Ko'rinuvchanlik va o'chirilgan filtri `ListByLesson` bilan bir xil bo'lishi
// SHART — haqiqiy repozitoriyda ular bitta `chatVisible` dan keladi.
func (r *FakeChatRepo) ListAllByLesson(_ context.Context, lessonID, viewerIdentity string, max int) ([]*entity.ChatMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if max <= 0 {
		max = 10000
	}
	var out []*entity.ChatMessage
	for _, m := range r.items {
		if m.LessonID != lessonID {
			continue
		}
		if _, gone := r.deleted[m.ID]; gone {
			continue
		}
		if m.ToIdentity != nil {
			if viewerIdentity == "" {
				continue
			}
			if *m.ToIdentity != viewerIdentity && m.SenderIdentity != viewerIdentity {
				continue
			}
		}
		cp := *m
		out = append(out, &cp)
	}
	// `created_at ASC` — haqiqiy repo tartibi. Fake qo'shilish tartibini
	// qaytarganda "vaqt bo'yicha saralash" testlari yolg'on o'tib ketardi.
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	if len(out) > max {
		out = out[:max]
	}
	return out, nil
}

// ─── PollRepo ────────────────────────────────────────────────────────────────

type FakePollRepo struct {
	mu    sync.Mutex
	polls map[string]*entity.Poll
	votes map[string]map[string]int // pollID → identity → optionIndex
}

func NewFakePollRepo() *FakePollRepo {
	return &FakePollRepo{polls: map[string]*entity.Poll{}, votes: map[string]map[string]int{}}
}

func (r *FakePollRepo) Create(_ context.Context, p *entity.Poll) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.polls[p.ID] = &cp
	return nil
}
func (r *FakePollRepo) GetByID(_ context.Context, id string) (*entity.Poll, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.polls[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, apperr.NotFound("poll")
}
func (r *FakePollRepo) ListByLesson(_ context.Context, lessonID string) ([]*entity.Poll, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.Poll
	for _, p := range r.polls {
		if p.LessonID == lessonID {
			cp := *p
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (r *FakePollRepo) Close(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.polls[id]; ok {
		p.IsActive = false
	}
	return nil
}

func (r *FakePollRepo) Publish(_ context.Context, id string) (*entity.Poll, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.polls[id]
	if !ok {
		return nil, apperr.NotFound("poll")
	}
	// IDEMPOTENT — haqiqiy repo'dagi COALESCE bilan bir xil: takroriy e'lon
	// vaqtni surib yubormaydi.
	if p.ResultsPublishedAt == nil {
		now := time.Now().UTC()
		p.ResultsPublishedAt = &now
	}
	cp := *p
	return &cp, nil
}
func (r *FakePollRepo) Vote(_ context.Context, pollID, identity string, optionIndex int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.votes[pollID] == nil {
		r.votes[pollID] = map[string]int{}
	}
	r.votes[pollID][identity] = optionIndex // upsert
	return nil
}
func (r *FakePollRepo) Counts(_ context.Context, pollID string, numOptions int) ([]int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := make([]int, numOptions)
	for _, idx := range r.votes[pollID] {
		if idx >= 0 && idx < numOptions {
			counts[idx]++
		}
	}
	return counts, nil
}

// ─── FakeTxRunner ────────────────────────────────────────────────────────────

// FakeTxRunner — `auth.TxRunner` ning test implementatsiyasi.
//
// Haqiqiy tranzaksiyani REAL rollback bilan taqlid qiladi: `fn` xato qaytarsa,
// uning ichida yaratilgan yozuvlar bekor qilinadi. Bu muhim — agar fake
// shunchaki `fn` ni chaqirib qo'ya qolsa, "rollback bo'ldi" degan test
// hech nimani isbotlamasdi (yozuv baribir qolaverardi va test yolg'on-yashil
// bo'lardi).
//
// Taqlid usuli: `fn` dan OLDIN ikkala fake'ning holatidan snapshot olinadi,
// xatoda esa snapshot qaytariladi. Bu haqiqiy `ROLLBACK` bilan bir xil
// kuzatiladigan natijani beradi.
type FakeTxRunner struct {
	users *FakeUserRepo
	auths *FakeAuthRepo
	// Commits/Rollbacks — test tranzaksiya haqiqatan ishlaganini tekshira olsin.
	Commits   int
	Rollbacks int
}

func NewFakeTxRunner(users *FakeUserRepo, auths *FakeAuthRepo) *FakeTxRunner {
	return &FakeTxRunner{users: users, auths: auths}
}

func (f *FakeTxRunner) RunInTx(
	ctx context.Context,
	fn func(users repository.UserRepository, auth repository.AuthRepository) error,
) error {
	userSnap := f.users.snapshot()
	tokenSnap := f.auths.snapshot()

	if err := fn(f.users, f.auths); err != nil {
		f.users.restore(userSnap)
		f.auths.restore(tokenSnap)
		f.Rollbacks++
		return err
	}
	f.Commits++
	return nil
}

// snapshot/restore — rollback taqlidi uchun holat nusxasi.
func (r *FakeUserRepo) snapshot() map[string]*entity.User {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make(map[string]*entity.User, len(r.byID))
	for k, v := range r.byID {
		u := *v
		cp[k] = &u
	}
	return cp
}

func (r *FakeUserRepo) restore(snap map[string]*entity.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID = snap
}

func (r *FakeAuthRepo) snapshot() map[string]*entity.RefreshToken {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make(map[string]*entity.RefreshToken, len(r.tokens))
	for k, v := range r.tokens {
		t := *v
		cp[k] = &t
	}
	return cp
}

func (r *FakeAuthRepo) restore(snap map[string]*entity.RefreshToken) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = snap
}
