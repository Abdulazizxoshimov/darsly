package token_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"

	"github.com/zoom/darsly/internal/testutil"
)

// redisAddr — test Redis manzili (default dev porti 6399). Ulanib bo'lmasa test skip.
func redisAddr() string {
	if a := os.Getenv("TEST_REDIS_ADDR"); a != "" {
		return a
	}
	return "localhost:6399"
}

const testPrefix = "tokentest"

func newMaker(t *testing.T) (token.Maker, *goredis.Client) {
	return newMakerWithGrace(t, token.DefaultRefreshGrace)
}

// newMakerWithGrace grace oynasini test bo'yicha sozlash imkonini beradi
// (0 → grace o'chirilgan, ya'ni eski "reuse → butun oilani bekor qilish" xatti-harakati).
// Har chaqiruvda test prefiksidagi kalitlar tozalanadi — testlar ketma-ket ishlaydi,
// shuning uchun oldingi yugurishdan qolgan kalitlar natijaga ta'sir qilmaydi.
func newMakerWithGrace(t *testing.T, grace time.Duration) (token.Maker, *goredis.Client) {
	t.Helper()
	cli := goredis.NewClient(&goredis.Options{Addr: redisAddr()})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := cli.Ping(ctx).Err(); err != nil {
		testutil.SkipOrFail(t, "Redis mavjud emas (%s): %v", redisAddr(), err)
	}
	flushTestKeys(t, cli)
	log := logger.New("error", "test", "test")
	m := token.NewJWTMaker([]byte("test-secret-at-least-32-chars-long-000"), 15*time.Minute, 720*time.Hour, grace, cli, testPrefix, log)
	return m, cli
}

func flushTestKeys(t *testing.T, cli *goredis.Client) {
	t.Helper()
	ctx := context.Background()
	var cursor uint64
	for {
		keys, next, err := cli.Scan(ctx, cursor, testPrefix+":*", 200).Result()
		require.NoError(t, err)
		if len(keys) > 0 {
			require.NoError(t, cli.Del(ctx, keys...).Err())
		}
		cursor = next
		if cursor == 0 {
			return
		}
	}
}

func TestGenerateValidateRevoke(t *testing.T) {
	m, _ := newMaker(t)
	ctx := context.Background()

	access, refresh, err := m.Generate(ctx, "user1", "sid-abc", "student")
	require.NoError(t, err)
	// Haqiqiy oqimda caller (auth.Login) sessiyani saqlaydi — takrorlaymiz.
	require.NoError(t, m.StoreSession(ctx, "sid-abc", "user1", 720*time.Hour))

	claims, err := m.ValidateAccess(ctx, access)
	require.NoError(t, err)
	require.Equal(t, "user1", claims.Sub)
	require.Equal(t, "student", claims.Role)

	// Logout: RevokeRefresh sessiya + JTI'ni o'chiradi → access ham yaroqsiz bo'ladi.
	require.NoError(t, m.RevokeRefresh(ctx, refresh))

	_, err = m.ValidateAccess(ctx, access)
	require.Error(t, err, "logout'dan keyin access token yaroqsiz bo'lishi kerak")

	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "logout'dan keyin refresh ishlamasligi kerak")
}

// Parol reset / deaktivatsiyadan keyin foydalanuvchining BARCHA sessiyalari
// (access sessiya kaliti + refresh jti) o'chirilishi shart.
func TestRevokeAllUserSessions(t *testing.T) {
	m, _ := newMaker(t)
	ctx := context.Background()

	// Bitta foydalanuvchida ikkita alohida sessiya (masalan ikki qurilma).
	access1, refresh1, err := m.Generate(ctx, "victim", "sid-1", "student")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-1", "victim", 720*time.Hour))

	access2, refresh2, err := m.Generate(ctx, "victim", "sid-2", "student")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-2", "victim", 720*time.Hour))

	// Boshqa foydalanuvchi sessiyasi — buzilmasligi kerak.
	accessOther, _, err := m.Generate(ctx, "bystander", "sid-3", "student")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-3", "bystander", 720*time.Hour))

	require.NoError(t, m.RevokeAllUserSessions(ctx, "victim"))

	// Ikkala access token ham darhol yaroqsiz.
	_, err = m.ValidateAccess(ctx, access1)
	require.Error(t, err, "reset'dan keyin 1-access yaroqsiz bo'lishi kerak")
	_, err = m.ValidateAccess(ctx, access2)
	require.Error(t, err, "reset'dan keyin 2-access yaroqsiz bo'lishi kerak")

	// Ikkala refresh ham rotatsiya qila olmasligi kerak.
	_, _, err = m.Rotate(ctx, refresh1)
	require.Error(t, err, "reset'dan keyin 1-refresh ishlamasligi kerak")
	_, _, err = m.Rotate(ctx, refresh2)
	require.Error(t, err, "reset'dan keyin 2-refresh ishlamasligi kerak")

	// Boshqa foydalanuvchi sessiyasi tirik qolishi kerak.
	_, err = m.ValidateAccess(ctx, accessOther)
	require.NoError(t, err, "boshqa foydalanuvchi sessiyasi buzilmasligi kerak")
}

// Grace oynasi TASHQARISIDA (bu yerda grace butunlay o'chirilgan) ishlatilgan
// refresh qayta ishlatilishi rad etilishi shart.
func TestRotate_ReuseDetection(t *testing.T) {
	m, _ := newMakerWithGrace(t, 0)
	ctx := context.Background()

	_, refresh, err := m.Generate(ctx, "user2", "sid-xyz", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-xyz", "user2", 720*time.Hour))

	_, newRefresh, err := m.Rotate(ctx, refresh)
	require.NoError(t, err)
	require.NotEqual(t, refresh, newRefresh)

	// Eski refresh'ni qayta ishlatish → rad etilishi kerak (token theft himoyasi).
	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "ishlatilgan refresh token qayta ishlatilmasligi kerak")
}

// Refresh reuse aniqlansa (grace oynasi tashqarisida) — shu foydalanuvchining BUTUN
// sessiya-oilasi bekor bo'lishi kerak (hujumchi ham, qurbon ham qayta login qilishga majbur).
func TestRotate_ReuseRevokesFamily(t *testing.T) {
	m, _ := newMakerWithGrace(t, 0)
	ctx := context.Background()

	// Foydalanuvchining ikkinchi (aloqasiz) sessiyasi — tirik va yaroqli.
	access2, _, err := m.Generate(ctx, "victim2", "sid-live", "student")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-live", "victim2", 720*time.Hour))
	// Ushbu sessiya reuse'gacha yaroqli ekanini tasdiqlaymiz.
	_, err = m.ValidateAccess(ctx, access2)
	require.NoError(t, err)

	// O'g'irlangan sessiyaning refresh'i bir marta rotatsiya qilinadi (JTI Redis'dan o'chadi).
	_, stolen, err := m.Generate(ctx, "victim2", "sid-stolen", "student")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-stolen", "victim2", 720*time.Hour))
	_, _, err = m.Rotate(ctx, stolen)
	require.NoError(t, err)

	// Eski (o'g'irlangan) refresh qayta ishlatiladi → reuse aniqlanadi.
	_, _, err = m.Rotate(ctx, stolen)
	require.Error(t, err, "qayta ishlatilgan refresh rad etilishi kerak")

	// Reuse aniqlangach ikkinchi (aloqasiz) sessiya ham bekor bo'lishi shart.
	_, err = m.ValidateAccess(ctx, access2)
	require.Error(t, err, "reuse aniqlansa foydalanuvchining boshqa sessiyasi ham bekor bo'lishi kerak")
}

// ─── BE-2: grace oynasi (idempotent rotatsiya) ───────────────────────────────

// Mobil stsenariy: server rotatsiyani BAJARDI, lekin javob 4G'da yo'qoldi (yoki
// ilovada ikki parallel 401-retry ketdi). Klient o'sha eski refresh bilan qayta
// uradi → grace oynasi ichida AYNI o'sha juftlik qaytishi va sessiya TIRIK qolishi
// shart (avval bu holat butun sessiya-oilasini o'ldirardi).
func TestRotate_GraceWindow_IdempotentRetry(t *testing.T) {
	m, _ := newMakerWithGrace(t, 60*time.Second)
	ctx := context.Background()

	// Foydalanuvchining ikkinchi (aloqasiz) qurilmasi — buzilmasligi kerak.
	otherAccess, _, err := m.Generate(ctx, "mobile1", "sid-tablet", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-tablet", "mobile1", 720*time.Hour))

	_, refresh, err := m.Generate(ctx, "mobile1", "sid-phone", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-phone", "mobile1", 720*time.Hour))

	// 1-rotatsiya muvaffaqiyatli (javob klientga yetmagan deb tasavvur qilamiz).
	access1, refresh1, err := m.Rotate(ctx, refresh)
	require.NoError(t, err)
	require.NotEqual(t, refresh, refresh1)

	// Klient eski refresh bilan qayta uradi → xuddi shu juftlik (idempotent).
	access2, refresh2, err := m.Rotate(ctx, refresh)
	require.NoError(t, err, "grace oynasi ichida takroriy so'rov xato bermasligi kerak")
	require.Equal(t, access1, access2, "grace ayni o'sha access tokenni qaytarishi kerak")
	require.Equal(t, refresh1, refresh2, "grace ayni o'sha refresh tokenni qaytarishi kerak")

	// Uchinchi urinish ham (yana bir retry) xuddi shu natijani berishi kerak.
	access3, refresh3, err := m.Rotate(ctx, refresh)
	require.NoError(t, err)
	require.Equal(t, access1, access3)
	require.Equal(t, refresh1, refresh3)

	// Eng muhimi: hech qanday sessiya o'lmagan.
	_, err = m.ValidateAccess(ctx, otherAccess)
	require.NoError(t, err, "grace holatida boshqa qurilma sessiyasi o'lmasligi kerak")
	_, err = m.ValidateAccess(ctx, access1)
	require.NoError(t, err, "grace holatida yangi access token yaroqli qolishi kerak")

	// Grace'dan qaytgan refresh keyingi normal rotatsiyada ishlashi kerak.
	_, refresh4, err := m.Rotate(ctx, refresh1)
	require.NoError(t, err)
	require.NotEqual(t, refresh1, refresh4)
}

// Grace oynasi TUGAGACH eski refresh'ni ishlatish haliyam o'g'irlik deb baholanadi
// va butun sessiya-oilasi bekor qilinadi (xavfsizlik xossasi saqlanadi).
func TestRotate_GraceExpired_RevokesFamily(t *testing.T) {
	m, _ := newMakerWithGrace(t, 120*time.Millisecond)
	ctx := context.Background()

	otherAccess, _, err := m.Generate(ctx, "mobile2", "sid-live2", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-live2", "mobile2", 720*time.Hour))

	_, refresh, err := m.Generate(ctx, "mobile2", "sid-stolen2", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-stolen2", "mobile2", 720*time.Hour))

	_, _, err = m.Rotate(ctx, refresh)
	require.NoError(t, err)

	// Grace oynasi tugashini kutamiz.
	time.Sleep(250 * time.Millisecond)

	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "grace tashqarisidagi reuse rad etilishi kerak")

	_, err = m.ValidateAccess(ctx, otherAccess)
	require.Error(t, err, "grace tashqarisidagi reuse butun sessiya-oilasini bekor qilishi kerak")
}

// Logout/parol reset sessiyani o'chirgan bo'lsa, grace yozuvi orqali sessiya
// TIRILMASLIGI shart.
func TestRotate_GraceBlockedAfterSessionRevoked(t *testing.T) {
	m, _ := newMakerWithGrace(t, 60*time.Second)
	ctx := context.Background()

	_, refresh, err := m.Generate(ctx, "mobile3", "sid-logout", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-logout", "mobile3", 720*time.Hour))

	_, _, err = m.Rotate(ctx, refresh) // grace yozuvi yaratildi
	require.NoError(t, err)

	// Logout: sessiya kaliti o'chadi.
	require.NoError(t, m.RevokeSession(ctx, "sid-logout"))

	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "sessiya bekor qilingach grace ishlamasligi kerak")
}

// RevokeAllUserSessions grace yozuvlarini ham tozalashi kerak (aks holda bekor
// qilingan sessiya grace orqali tirilardi).
func TestRevokeAllUserSessions_ClearsGrace(t *testing.T) {
	m, cli := newMakerWithGrace(t, 60*time.Second)
	ctx := context.Background()

	_, refresh, err := m.Generate(ctx, "mobile4", "sid-grace", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-grace", "mobile4", 720*time.Hour))
	_, _, err = m.Rotate(ctx, refresh)
	require.NoError(t, err)

	// Grace yozuvi haqiqatan mavjudligini tasdiqlaymiz.
	n, err := cli.Keys(ctx, testPrefix+":refresh:used:*").Result()
	require.NoError(t, err)
	require.NotEmpty(t, n, "rotatsiyadan keyin grace yozuvi bo'lishi kerak")

	require.NoError(t, m.RevokeAllUserSessions(ctx, "mobile4"))

	n, err = cli.Keys(ctx, testPrefix+":refresh:used:*").Result()
	require.NoError(t, err)
	require.Empty(t, n, "parol reset/deaktivatsiyadan keyin grace yozuvlari o'chishi kerak")

	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "family revoke'dan keyin grace ishlamasligi kerak")
}

// ─── BE-10: sessiya kaliti Rotate'da tekshiriladi va uzaytiriladi ────────────

// Sessiya kaliti yo'q (30 kunlik TTL tugagan / bekor qilingan) bo'lsa Rotate xato
// qaytarishi kerak. Avval Rotate faqat refresh JTI'ni ko'rardi va "yaroqli" juftlik
// qaytarardi — klient CHEKSIZ refresh siklida qolib, toza logout'ga tushmasdi.
func TestRotate_SessionMissing_Refuses(t *testing.T) {
	m, _ := newMaker(t)
	ctx := context.Background()

	// StoreSession ATAYIN chaqirilmaydi — sessiya kaliti TTL bilan o'chgan holat.
	_, refresh, err := m.Generate(ctx, "ghost", "sid-expired", "mentor")
	require.NoError(t, err)

	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err, "sessiya kaliti yo'q bo'lsa rotatsiya rad etilishi kerak")

	// Yetim JTI tozalangani uchun takroriy urinish ham xato beradi (cheksiz sikl yo'q).
	_, _, err = m.Rotate(ctx, refresh)
	require.Error(t, err)
}

// Faol foydalanuvchi refresh qilib turganda sessiya kaliti TTL'i uzayishi kerak
// (sliding sessiya) — aks holda 30 kundan keyin access 401, refresh esa "ishlaydi".
func TestRotate_ExtendsSessionTTL(t *testing.T) {
	m, cli := newMaker(t)
	ctx := context.Background()

	_, refresh, err := m.Generate(ctx, "active", "sid-sliding", "mentor")
	require.NoError(t, err)
	// Sessiya tugashiga oz qoldi (2 soniya).
	require.NoError(t, m.StoreSession(ctx, "sid-sliding", "active", 2*time.Second))

	before, err := cli.TTL(ctx, testPrefix+":sess:sid-sliding").Result()
	require.NoError(t, err)
	require.LessOrEqual(t, before, 2*time.Second)

	_, _, err = m.Rotate(ctx, refresh)
	require.NoError(t, err)

	after, err := cli.TTL(ctx, testPrefix+":sess:sid-sliding").Result()
	require.NoError(t, err)
	require.Greater(t, after, time.Hour, "Rotate sessiya TTL'ini refreshTTL'ga uzaytirishi kerak")

	// Sessiya payload'i userID bo'lib qolishi shart (RevokeAllUserSessions skani
	// shunga tayanadi) — aks holda parol reset sessiyani topa olmaydi.
	val, err := cli.Get(ctx, testPrefix+":sess:sid-sliding").Result()
	require.NoError(t, err)
	require.Equal(t, "active", val)
}

// ─── Parallel rotatsiya poygasi (CAS) ────────────────────────────────────────

// countLiveRefreshJTI — sessiyada nechta TIRIK refresh JTI borligini sanaydi
// (grace yozuvlari `refresh:used:` hisobga olinmaydi).
func countLiveRefreshJTI(t *testing.T, cli *goredis.Client) int {
	t.Helper()
	ctx := context.Background()
	keys, err := cli.Keys(ctx, testPrefix+":refresh:*").Result()
	require.NoError(t, err)
	n := 0
	for _, k := range keys {
		if !strings.HasPrefix(k, testPrefix+":refresh:used:") {
			n++
		}
	}
	return n
}

// Bitta refresh token bilan N ta PARALLEL rotatsiya → Redis'da AYNAN 1 ta tirik JTI
// qolishi va barcha javoblar BIR XIL juftlik bo'lishi shart.
//
// Tuzatishdan oldin `Get` va yozuv orasi check-then-act edi (TxPipelined = MULTI/EXEC,
// WATCH'siz, ya'ni CAS emas): har bir parallel so'rov o'z JTI'sini yozib, bitta sessiyada
// N ta mustaqil tirik refresh zanjiri paydo bo'lardi ("sessiya forking") — bu holda
// refresh-reuse o'g'irlik detektorini butunlay chetlab o'tish mumkin edi.
func TestRotate_ParallelRace_SingleWinner(t *testing.T) {
	m, cli := newMakerWithGrace(t, 60*time.Second)
	ctx := context.Background()

	access, refresh, err := m.Generate(ctx, "racer", "sid-race", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-race", "racer", 720*time.Hour))

	require.Equal(t, 1, countLiveRefreshJTI(t, cli), "boshlanishida 1 ta tirik JTI bo'lishi kerak")

	const n = 8
	type res struct {
		access, refresh string
		err             error
	}
	out := make([]res, n)
	var wg sync.WaitGroup
	start := make(chan struct{}) // barcha goroutine bir vaqtda boshlasin
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			a, r, err := m.Rotate(ctx, refresh)
			out[i] = res{a, r, err}
		}(i)
	}
	close(start)
	wg.Wait()

	// Eng muhim invariant: sessiyada faqat BITTA tirik refresh zanjiri qoladi.
	require.Equal(t, 1, countLiveRefreshJTI(t, cli),
		"parallel rotatsiyadan keyin AYNAN 1 ta tirik refresh JTI qolishi shart (sessiya forking yo'q)")

	// Grace yoqilgani uchun hech kim zarar ko'rmasligi kerak: hammasi bir xil juftlik.
	for i, r := range out {
		require.NoError(t, r.err, "grace yoqilganda parallel so'rov xato bermasligi kerak (i=%d)", i)
		require.Equal(t, out[0].access, r.access, "barcha javoblar bir xil access bo'lishi kerak (i=%d)", i)
		require.Equal(t, out[0].refresh, r.refresh, "barcha javoblar bir xil refresh bo'lishi kerak (i=%d)", i)
	}
	require.NotEqual(t, refresh, out[0].refresh)

	// Poyga sessiya-oilasini O'LDIRMASLIGI shart (bu reuse emas) — mavjud access tirik.
	_, err = m.ValidateAccess(ctx, access)
	require.NoError(t, err, "parallel poyga sessiyani bekor qilmasligi kerak")

	// G'olibning refresh'i keyingi normal rotatsiyada ishlashi kerak.
	_, next, err := m.Rotate(ctx, out[0].refresh)
	require.NoError(t, err)
	require.NotEqual(t, out[0].refresh, next)
	require.Equal(t, 1, countLiveRefreshJTI(t, cli))
}

// Grace O'CHIRILGAN (JWT_REFRESH_GRACE=0) holatda ham "sessiya forking" bo'lmasligi
// shart: rotatsiyani aynan BITTA so'rov bajaradi, qolganlari yangi JTI YOZMAYDI.
//
// Diqqat — bu rejimda mag'lublardan biri "reuse" yo'liga tushib butun sessiya-oilasini
// bekor qilishi MUMKIN (grace yozuvi bo'lmagani uchun parallel poygani ketma-ket
// o'g'irlikdan ajratib bo'lmaydi). Bu grace=0 ning ataylab tanlangan qat'iy xatti-harakati;
// default 60s grace bilan bunday bo'lmaydi (yuqoridagi testga qarang). Shuning uchun
// bu yerda tirik JTI soni 0 (oila bekor qilingan) yoki 1 (g'olib) — lekin HECH QACHON N.
func TestRotate_ParallelRace_NoGrace_NoForking(t *testing.T) {
	m, cli := newMakerWithGrace(t, 0)
	ctx := context.Background()

	_, refresh, err := m.Generate(ctx, "racer2", "sid-race2", "mentor")
	require.NoError(t, err)
	require.NoError(t, m.StoreSession(ctx, "sid-race2", "racer2", 720*time.Hour))

	const n = 8
	errs := make([]error, n)
	toks := make([]string, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, r, err := m.Rotate(ctx, refresh)
			toks[i], errs[i] = r, err
		}(i)
	}
	close(start)
	wg.Wait()

	winners := 0
	for i := range errs {
		if errs[i] == nil {
			winners++
			require.NotEmpty(t, toks[i])
		}
	}
	require.Equal(t, 1, winners, "grace'siz holatda ham rotatsiyani aynan bitta so'rov bajarishi kerak")

	// Asosiy invariant: N ta parallel zanjir YARATILMAYDI. Tuzatishdan oldin bu yerda
	// 8 ta tirik JTI qolardi (har biri 30 kun yashovchi mustaqil zanjir).
	live := countLiveRefreshJTI(t, cli)
	require.LessOrEqual(t, live, 1, "sessiya forking bo'lmasligi kerak (tirik JTI: %d)", live)
}
