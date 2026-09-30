package joinlink_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/joinlink"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

// testIP — testlarning "odatiy bitta klienti". Lockout endi slug+IP bo'yicha
// ajratilgani uchun (M7) har chaqiruvda IP kerak.
const testIP = "10.0.0.1"

func ptr[T any](v T) *T { return &v }

func setup(t *testing.T) (joinlink.UseCase, *testutil.FakeLessonRepo, *testutil.FakeUserRepo) {
	uc, lrepo, urepo, _ := setupWithBlocklist(t)
	return uc, lrepo, urepo
}

// setupWithBlocklist — setup + mentor qora ro'yxati fake'iga kirish (№4 testlari).
func setupWithBlocklist(t *testing.T) (joinlink.UseCase, *testutil.FakeLessonRepo, *testutil.FakeUserRepo, *testutil.FakeBlocklistRepo) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", Email: "m@x.uz", FullName: "Dilnoza Mentor", Role: "mentor"}))
	h := hasher.New(4)
	roomUC := &testutil.FakeRoomUC{}
	cache := testutil.NewFakeCache()
	blk := testutil.NewFakeBlocklistRepo()
	waitUC := waitingroom.New(testutil.NewFakeWaitingRepo(), lrepo, roomUC, ws.NewHub(testutil.NewLogger()), cache, testutil.NewLogger())
	uc := joinlink.New(lrepo, urepo, h, cache, roomUC, waitUC, blk, testutil.NewLogger())
	return uc, lrepo, urepo, blk
}

func seedLesson(t *testing.T, lrepo *testutil.FakeLessonRepo, l *entity.Lesson) {
	t.Helper()
	if l.Status == "" {
		l.Status = entity.LessonStatusLive
	}
	require.NoError(t, lrepo.Create(context.Background(), l))
}

func TestPreview(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", Title: "Matem", JoinSlug: "abc-defg-hjk"})

	pub, err := uc.Preview(context.Background(), "abc-defg-hjk")
	require.NoError(t, err)
	require.Equal(t, "Matem", pub.Title)
	require.Equal(t, "Dilnoza Mentor", pub.MentorName)
	require.False(t, pub.HasPasscode)
}

func TestPreview_NotFound(t *testing.T) {
	uc, _, _ := setup(t)
	_, err := uc.Preview(context.Background(), "yoq-yoq-yoq")
	require.True(t, apperr.IsNotFound(err))
}

func TestJoin_NoPasscode_DirectToken(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s1", IsWaitingRoomEnabled: false})

	resp, err := uc.Join(context.Background(), "s1", testIP, &entity.JoinLessonReq{GuestName: ptr("Aziz")})
	require.NoError(t, err)
	require.Equal(t, "join", resp.NextStep)
	require.NotNil(t, resp.Room, "waiting-room OFF → to'g'ridan-to'g'ri token")
	require.Equal(t, "part-tok", resp.Room.Token)
}

func TestJoin_Passcode(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s2", PasscodeHash: &hash})

	// Parolsiz → 401
	_, err := uc.Join(context.Background(), "s2", testIP, &entity.JoinLessonReq{})
	require.True(t, errorsIsUnauthorized(err), "parol talab qilinadi")

	// Noto'g'ri parol → 401
	_, err = uc.Join(context.Background(), "s2", testIP, &entity.JoinLessonReq{Passcode: ptr("0000")})
	require.True(t, errorsIsUnauthorized(err))

	// To'g'ri parol → token
	resp, err := uc.Join(context.Background(), "s2", testIP, &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.NoError(t, err)
	require.NotNil(t, resp.Room)
}

// Ko'p marta noto'g'ri parol → SHU KLIENT qulflanadi, to'g'ri parol ham rad etiladi.
func TestJoin_PasscodeBruteForceLockout(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "bf", PasscodeHash: &hash})

	for i := range 5 {
		_, err := uc.Join(context.Background(), "bf", testIP, &entity.JoinLessonReq{Passcode: ptr("0000")})
		require.Error(t, err, "urinish %d noto'g'ri", i+1)
	}

	// Endi to'g'ri parol ham qulf tufayli rad etiladi.
	_, err := uc.Join(context.Background(), "bf", testIP, &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.True(t, apperr.IsForbidden(err), "5 urinishdan keyin klient qulflanishi kerak")
}

func TestJoin_WaitingRoom(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s3", IsWaitingRoomEnabled: true})

	resp, err := uc.Join(context.Background(), "s3", testIP, &entity.JoinLessonReq{GuestName: ptr("Laylo")})
	require.NoError(t, err)
	require.Equal(t, "waiting_room", resp.NextStep)
	require.NotEmpty(t, resp.RequestID, "kutish xonasi → request_id qaytadi")
	require.Nil(t, resp.Room, "token admit'dan keyin beriladi")
}

func TestJoin_Locked(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s4", IsLocked: true})
	_, err := uc.Join(context.Background(), "s4", testIP, &entity.JoinLessonReq{})
	require.True(t, apperr.IsForbidden(err), "qulflangan darsga kirib bo'lmaydi")
}

// №3 — yakunlangan dars havolasi: JOIN token bermaydi, lekin xato ham emas —
// klient preview ma'lumotini ko'rsatib "dars tugagan" deydi (next_step bilan).
func TestJoin_Ended_ReturnsLessonEndedState(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", Title: "Matem", JoinSlug: "s5", Status: entity.LessonStatusEnded})

	resp, err := uc.Join(context.Background(), "s5", testIP, &entity.JoinLessonReq{GuestName: ptr("Aziz")})
	require.NoError(t, err)
	require.Equal(t, entity.JoinNextStepLessonEnded, resp.NextStep)
	require.Nil(t, resp.Room, "tugagan darsga token BERILMAYDI")
	require.Empty(t, resp.RequestID, "kutish so'rovi ham yaratilmaydi")
	require.NotNil(t, resp.Lesson)
	require.Equal(t, entity.LessonStatusEnded, resp.Lesson.Status, "klient aniq holatni status'dan oladi")
}

// Bekor qilingan dars ham xuddi shu holatni qaytaradi (status farqlaydi).
func TestJoin_Cancelled_ReturnsLessonEndedState(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s6", Status: entity.LessonStatusCancelled})

	resp, err := uc.Join(context.Background(), "s6", testIP, &entity.JoinLessonReq{})
	require.NoError(t, err)
	require.Equal(t, entity.JoinNextStepLessonEnded, resp.NextStep)
	require.Nil(t, resp.Room)
	require.Equal(t, entity.LessonStatusCancelled, resp.Lesson.Status)
}

// Tugagan darsda hatto parol/qulf ham tekshirilmaydi — havola faqat ma'lumot:
// parol so'rab o'quvchini ovora qilishning ma'nosi yo'q.
func TestJoin_Ended_SkipsPasscodeAndLock(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{
		ID: "l1", MentorID: "mentor1", JoinSlug: "s7",
		Status: entity.LessonStatusEnded, PasscodeHash: &hash, IsLocked: true,
	})

	resp, err := uc.Join(context.Background(), "s7", testIP, &entity.JoinLessonReq{})
	require.NoError(t, err, "parolsiz ham lesson_ended qaytishi kerak")
	require.Equal(t, entity.JoinNextStepLessonEnded, resp.NextStep)
}

// №3 — Preview tugagan dars uchun ham 200 qaytaradi (ma'lumot sahifasi).
func TestPreview_EndedLessonStillVisible(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", Title: "Tarix", JoinSlug: "s8", Status: entity.LessonStatusEnded})

	pub, err := uc.Preview(context.Background(), "s8")
	require.NoError(t, err, "tugagan dars preview'i 200 bo'lishi kerak (avval 400 edi)")
	require.Equal(t, "Tarix", pub.Title)
	require.Equal(t, entity.LessonStatusEnded, pub.Status)
}

// ─── №4 — mentor qora ro'yxati join'da ──────────────────────────────────────

// Doimiy bloklangan ism token ham, kutish-xonasi so'rovi ham ololmaydi.
func TestJoin_MentorBlockedName(t *testing.T) {
	uc, lrepo, _, blk := setupWithBlocklist(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "b1"})
	seedLesson(t, lrepo, &entity.Lesson{ID: "l2", MentorID: "mentor1", JoinSlug: "b2", IsWaitingRoomEnabled: true})
	require.NoError(t, blk.Add(context.Background(), &entity.BlocklistEntry{
		ID: "e1", MentorID: "mentor1", Identity: "guest_old", DisplayName: "Bezori Aka",
	}))

	// To'g'ridan-to'g'ri kirish — katta-kichik harf farqsiz bloklanadi.
	_, err := uc.Join(context.Background(), "b1", testIP, &entity.JoinLessonReq{GuestName: ptr("bezori aka")})
	require.True(t, apperr.IsForbidden(err), "bloklangan ism token olmasligi kerak")

	// Kutish xonasi yo'li ham yopiq (so'rov yaratilmaydi).
	_, err = uc.Join(context.Background(), "b2", testIP, &entity.JoinLessonReq{GuestName: ptr("Bezori Aka")})
	require.True(t, apperr.IsForbidden(err), "kutish xonasi so'rovi ham yaratilmasligi kerak")

	// Boshqa ism bilan halol o'quvchi kiradi (ban ismga bog'langan — ongli cheklov).
	resp, err := uc.Join(context.Background(), "b1", testIP, &entity.JoinLessonReq{GuestName: ptr("Halol")})
	require.NoError(t, err)
	require.Equal(t, entity.JoinNextStepJoin, resp.NextStep)
}

// Blocklist o'qishda xato — fail-open: butun sinf darsdan to'silmaydi.
func TestJoin_BlocklistFailOpen(t *testing.T) {
	uc, lrepo, _, blk := setupWithBlocklist(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "fo"})
	blk.FailNext = context.DeadlineExceeded

	resp, err := uc.Join(context.Background(), "fo", testIP, &entity.JoinLessonReq{GuestName: ptr("Aziz")})
	require.NoError(t, err, "blocklist xatosi kirishni to'smasligi kerak (fail-open)")
	require.NotNil(t, resp.Room)
}

func errorsIsUnauthorized(err error) bool {
	ae := apperr.As(err)
	return ae != nil && ae.HTTPStatus == 401
}

// ⭐ M7 — sinf-lockout DoS yopilgani.
//
// Auditda topilgan holat: lockout FAQAT slug bo'yicha edi, ya'ni bitta odam
// (yoki bitta buzg'unchi) 5 marta noto'g'ri parol kiritsa BUTUN SINF 5 daqiqa
// darsga kira olmasdi. Parolni noto'g'ri eslagan bitta o'quvchi ham 30 kishilik
// darsni buzib qo'yardi.
func TestJoin_LockoutIsPerClient_NotWholeClass(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "cls", PasscodeHash: &hash})

	// Buzg'unchi (yoki parolni unutgan o'quvchi) o'zini qulflaydi.
	const attacker = "203.0.113.9"
	for i := range 5 {
		_, err := uc.Join(context.Background(), "cls", attacker, &entity.JoinLessonReq{Passcode: ptr("0000")})
		require.Error(t, err, "urinish %d", i+1)
	}
	_, err := uc.Join(context.Background(), "cls", attacker, &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.True(t, apperr.IsForbidden(err), "aybdor qulflanishi kerak")

	// ⭐ Sinfdoshlar esa TO'G'RI parol bilan bemalol kirishi kerak.
	for _, ip := range []string{"10.0.0.2", "10.0.0.3", "192.168.1.7"} {
		resp, err := uc.Join(context.Background(), "cls", ip, &entity.JoinLessonReq{Passcode: ptr("1234")})
		require.NoError(t, err, "sinfdosh (%s) qulflanmasligi kerak", ip)
		require.NotNil(t, resp.Room)
	}
}

// Taqsimlangan hujum (har urinish yangi IP'dan) baribir to'xtatilishi kerak —
// aks holda "IP bo'yicha ajratish" himoyani butunlay o'chirib qo'yardi.
func TestJoin_DistributedBruteForceStillLocked(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "dist", PasscodeHash: &hash})

	// Har urinish BOSHQA IP'dan — klient darajasidagi qulf hech qachon ishlamaydi.
	for i := range 50 {
		ip := fmt.Sprintf("198.51.100.%d", i)
		_, err := uc.Join(context.Background(), "dist", ip, &entity.JoinLessonReq{Passcode: ptr("0000")})
		require.Error(t, err)
	}

	// Slug darajasidagi zaxira qulf ishga tushishi kerak.
	_, err := uc.Join(context.Background(), "dist", "198.51.100.200", &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.True(t, apperr.IsForbidden(err), "taqsimlangan hujum slug darajasida to'xtatilishi kerak")
}

// ⭐ Ovoz siyosati JAVOBDA bo'lishi shart.
//
// Avval o'quvchi `allow_self_unmute` ni faqat USTOZ KLIENTI yuboradigan
// data-message'dan bilardi. Mobil ustoz uni yubormaydi — telefondan o'tilgan
// darsda o'quvchining mikrofon tugmasi yolg'on ko'rsatardi (yoqadi, server esa
// jimgina qayta mute qiladi). Siyosat serverda saqlanadi, demak javobda ham
// serverdan kelishi kerak.
func TestPreviewAndJoin_ReturnAudioPolicy(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{
		ID: "l1", MentorID: "mentor1", Title: "Ovoz", JoinSlug: "pol-icy-1",
		MuteOnEntry: true, AllowSelfUnmute: false,
	})

	prev, err := uc.Preview(context.Background(), "pol-icy-1")
	require.NoError(t, err)
	require.True(t, prev.MuteOnEntry, "preview siyosatni qaytarishi kerak")
	require.False(t, prev.AllowSelfUnmute, "unmute taqiqi klientga yetishi kerak")

	resp, err := uc.Join(context.Background(), "pol-icy-1", testIP, &entity.JoinLessonReq{})
	require.NoError(t, err)
	require.True(t, resp.Lesson.MuteOnEntry)
	require.False(t, resp.Lesson.AllowSelfUnmute, "join javobida ham siyosat bo'lishi kerak")
}

// Ruxsat beruvchi dars uchun ikkala bayroq ham to'g'ri o'tishi kerak
// (maydonlar "har doim false" bo'lib qolib ketmasin).
func TestPreview_AudioPolicyPermissive(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{
		ID: "l2", MentorID: "mentor1", Title: "Erkin", JoinSlug: "pol-icy-2",
		MuteOnEntry: false, AllowSelfUnmute: true,
	})

	prev, err := uc.Preview(context.Background(), "pol-icy-2")
	require.NoError(t, err)
	require.False(t, prev.MuteOnEntry)
	require.True(t, prev.AllowSelfUnmute)
}

func TestJoin_ScheduledLesson_WaitsForHost_NoToken(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l9", MentorID: "mentor1", JoinSlug: "s9", Status: entity.LessonStatusScheduled})

	resp, err := uc.Join(context.Background(), "s9", testIP, &entity.JoinLessonReq{GuestName: ptr("Aziz")})
	require.NoError(t, err)
	require.Equal(t, entity.JoinNextStepWaitingForHost, resp.NextStep)
	require.Nil(t, resp.Room, "host kirmaguncha participant token berilmaydi")
}
