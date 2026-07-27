package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/token"
)

// ─── BE-11: CGNAT ostida auth rate-limit ─────────────────────────────────────
//
// MUAMMO: `/auth/*` uchun yagona "60 so'rov/min/IP" cheklovi mobil operator (Ucell/
// Beeline) CGNAT'i ortida yashovchi MINGLAB abonentni bitta hisoblagichga tiqadi.
// Ertalab ommaviy login/refresh → 429 → o'qituvchilar tizimga kira olmaydi. IP mobil
// klient uchun noto'g'ri o'lcham.
//
// YECHIM: har endpoint uchun MA'NOLI o'lcham + IP shifti (ceiling):
//
//	/auth/login   → (IP+email) qattiq · email (har qanday IP) o'rtacha · IP shifti keng
//	/auth/refresh → sid (sessiya) bo'yicha · IP shifti keng
//
// BRUTE-FORCE HIMOYASI KUCHSIZLANMAYDI, aksincha kuchayadi — asos:
//
//  1. Nishonli parol-tanlash (bitta akkaunt): avval 60/min/IP edi, endi (IP+email)
//     bo'yicha 10/min — ya'ni bir akkauntga urinish 6 BARAVAR QATTIQ cheklangan.
//  2. Taqsimlangan hujum (ko'p IP, bitta akkaunt): avval umuman cheklanmagan edi
//     (har IP o'z 60 tasini olardi), endi email bo'yicha IP'dan QAT'I NAZAR 30/min —
//     ya'ni butunlay YANGI himoya qatlami.
//  3. Credential stuffing (bitta IP, ko'p email): per-email kalit yolg'iz qolsa bu
//     chetlab o'tilardi, shuning uchun IP shifti (200/min) SAQLANADI. U CGNAT uchun
//     saxiy, lekin bitta manbadan cheksiz urinishni baribir to'sadi.
//  4. Refresh brute-force MAVJUD EMAS: refresh token — HS256 imzolangan 256-bitli sir,
//     uni taxmin qilib bo'lmaydi. U yerdagi haqiqiy xavf — resurs DoS, uni IP shifti
//     (600/min) qoplaydi. Shuning uchun refresh'ni sessiya bo'yicha cheklash to'g'ri:
//     bir foydalanuvchining tsikli boshqalarni bloklamaydi.
//
// Barcha limitlar env'dan sozlanadi (router.go), Redis-backed → ko'p-instansda umumiy.

// maxAuthBodyPeek — rate-limit kaliti uchun o'qiladigan body'ning maksimal hajmi.
// Auth body'lari kichik; kattasi bo'lsa kalit ajratilmaydi (IP shifti qoplaydi).
const maxAuthBodyPeek = 8 << 10 // 8 KB

const ctxAuthBodyKey = "middleware.authBody"

// authBody — /auth/* body'sidan rate-limit uchun kerakli maydonlar.
type authBody struct {
	Email        string `json:"email"`
	RefreshToken string `json:"refresh_token"`
}

// peekAuthBody body'ni O'QIYDI va handler uchun QAYTA TIKLAYDI (aks holda
// ShouldBindJSON bo'sh body ko'rardi). Natija kontekstda keshlanadi — bir necha
// limiter zanjirlanganda body faqat bir marta o'qiladi.
func peekAuthBody(c *gin.Context) authBody {
	if v, ok := c.Get(ctxAuthBodyKey); ok {
		if ab, ok := v.(authBody); ok {
			return ab
		}
	}

	var ab authBody
	if c.Request == nil || c.Request.Body == nil {
		c.Set(ctxAuthBodyKey, ab)
		return ab
	}

	peek, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAuthBodyPeek+1))
	if err != nil {
		// Body'ni o'qib bo'lmadi — o'qilgan qismini oldiga qo'yib tiklaymiz.
		c.Request.Body = restoreBody(peek, c.Request.Body)
		c.Set(ctxAuthBodyKey, ab)
		return ab
	}

	if len(peek) > maxAuthBodyPeek {
		// Juda katta body — kalit ajratmaymiz, lekin oqimni buzmaymiz.
		c.Request.Body = restoreBody(peek, c.Request.Body)
		c.Set(ctxAuthBodyKey, ab)
		return ab
	}

	// Body butunlay o'qildi — handler uchun to'liq tiklaymiz.
	_ = c.Request.Body.Close()
	c.Request.Body = io.NopCloser(bytes.NewReader(peek))

	// Xato bo'lsa e'tiborsiz: bu faqat rate-limit kaliti, validatsiya emas
	// (buzuq JSON'ni handler o'zi 400 bilan rad etadi).
	_ = json.Unmarshal(peek, &ab)

	c.Set(ctxAuthBodyKey, ab)
	return ab
}

// restoreBody o'qilgan bo'lakni qolgan oqim oldiga ulaydi (Close ishlashi uchun
// asl Closer saqlanadi).
func restoreBody(read []byte, rest io.ReadCloser) io.ReadCloser {
	return struct {
		io.Reader
		io.Closer
	}{
		Reader: io.MultiReader(bytes.NewReader(read), rest),
		Closer: rest,
	}
}

// normalizeEmail — kalit uchun email'ni bir shaklga keltiradi, aks holda hujumchi
// harf registrini/bo'shliqni o'zgartirib har safar yangi bucket olardi.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// hashKey uzun/maxfiy qiymatni qisqa va xavfsiz kalitga aylantiradi — Redis kalitida
// email yoki sessiya id'si ochiq yotmasin (Redis dump ham PII manbasi).
func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16]) // 128 bit — to'qnashuv amalda yo'q
}

// RateLimitLogin — /auth/login uchun uch o'lchamli cheklov zanjiri.
// Qaytarilgan handler'lar router'da ketma-ket qo'yiladi.
//
//	ipEmail — (IP + email) bo'yicha: "shu joydan shu akkauntga" nishonli urinish
//	email   — email bo'yicha (IP'dan qat'i nazar): taqsimlangan hujum
//	ip      — IP shifti: credential stuffing / DoS chegarasi (CGNAT uchun saxiy)
func RateLimitLogin(cache redis.Cache, ipEmail, email, ip int64, window time.Duration) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		// 1) IP+email — eng qattiq. Email bo'lmasa kalit yo'q (IP shifti qoplaydi).
		RateLimitRedisKeyed(cache, "login:ip-email", ipEmail, window, func(c *gin.Context) string {
			e := normalizeEmail(peekAuthBody(c).Email)
			if e == "" {
				return ""
			}
			return hashKey(c.ClientIP() + "|" + e)
		}),
		// 2) Email — IP almashtirib qutulmaslik uchun.
		RateLimitRedisKeyed(cache, "login:email", email, window, func(c *gin.Context) string {
			e := normalizeEmail(peekAuthBody(c).Email)
			if e == "" {
				return ""
			}
			return hashKey(e)
		}),
		// 3) IP shifti — har doim ishlaydi (email bo'lmagan/buzuq body holatlarini ham
		//    qoplaydi), shuning uchun cheklovsiz teshik qolmaydi.
		RateLimitRedis(cache, "login:ip", ip, window),
	}
}

// RateLimitRefresh — /auth/refresh uchun sessiya (sid) bo'yicha cheklov + IP shifti.
//
// ⚠️ sid FAQAT IMZO TEKSHIRILGANDAN KEYIN kalit sifatida ishlatiladi (maker.SessionFromRefresh).
//
// Nega bu kritik: `refresh:sid` bucket'i ataylab IP'dan MUSTAQIL (CGNAT'ning butun
// maqsadi shu). Demak sid'ni tekshirmasdan qabul qilsak, zarar hujumchining emas,
// QURBONNING bucket'iga tushadi: hujumchi soxta imzoli token ichiga qurbonning sid'ini
// yozib (sid muddati tugagan access token'dan ham o'qiladi — log/skrinshot artefakti)
// arzon so'rovlar bilan qurbonning hisoblagichini to'ldiradi. Natijada qurbonning
// haqiqiy refresh'i 429 oladi, frontend esa refresh muvaffaqiyatsiz bo'lsa foydalanuvchini
// tizimdan chiqaradi — dars o'rtasidagi ustoz uchun halokatli.
//
// Imzo tekshiruvi (bitta HMAC-SHA256, ~mikrosekund) ildizni yopadi: soxta token'ning
// sid'i bucket'ga UMUMAN kirmaydi va chaqiruvchi IP shiftiga — ya'ni O'Z IP'siga — tushadi.
// Qurbonning bucket'iga faqat o'sha sessiyaning haqiqiy, imzolangan tokenini ushlab
// turgan kishi tega oladi; unday kishi allaqachon sessiyani egallagan bo'ladi va bu
// holatni refresh-reuse detektori (RevokeAllUserSessions) qamrab oladi.
func RateLimitRefresh(cache redis.Cache, maker token.Maker, perSID, perIP int64, window time.Duration) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		RateLimitRedisKeyed(cache, "refresh:sid", perSID, window, func(c *gin.Context) string {
			raw := peekAuthBody(c).RefreshToken
			if raw == "" {
				return ""
			}
			// Imzo/tur yaroqsiz (yoki muddati tugagan) → sessiya kaliti YO'Q.
			// Bunday so'rov IP shifti bilan cheklanadi; u baribir 401 oladi.
			sid, err := maker.SessionFromRefresh(raw)
			if err != nil || sid == "" {
				return ""
			}
			return hashKey(sid)
		}),
		RateLimitRedis(cache, "refresh:ip", perIP, window),
	}
}
