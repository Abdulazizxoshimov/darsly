package chat_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/chat"
)

// testLessonID — haqiqiy UUID: usecase ID formatini tekshiradi (shared.ValidateID).
const testLessonID = "11111111-1111-4111-8111-111111111111"

func setup(t *testing.T) (chat.UseCase, *testutil.FakeLiveKit) {
	t.Helper()
	uc, lk, _ := setupFull(t)
	return uc, lk
}

// setupFull — fayl/moderatsiya testlari uchun repo va saqlagichga ham kirish beradi.
func setupFull(t *testing.T) (chat.UseCase, *testutil.FakeLiveKit, *testutil.FakeChatRepo) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", Email: "m@x.uz", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Title: "X", Status: entity.LessonStatusLive}))
	lk := testutil.NewFakeLiveKit() // enabled mock — SendData chaqiruvini assert qilamiz
	crepo := testutil.NewFakeChatRepo()
	uc := chat.New(crepo, lrepo, urepo, lk, testutil.NewFakeMinio(), testutil.NewFakeCache(), testutil.NewLogger())
	return uc, lk, crepo
}

func TestSendAndHistory(t *testing.T) {
	uc, lk := setup(t)
	ctx := context.Background()

	m, err := uc.Send(ctx, "mentor1", testLessonID, "Salom, darsni boshlaymiz", "")
	require.NoError(t, err)
	require.Equal(t, "Dilnoza", m.SenderName)
	require.Equal(t, "mentor1", m.SenderIdentity)
	require.Nil(t, m.ToIdentity, "manzilsiz xabar ommaviy bo'lishi kerak")
	require.GreaterOrEqual(t, lk.Calls["SendData"], 1, "chat LiveKit data-channel orqali broadcast qilinishi kerak")

	hist, err := uc.History(ctx, "mentor1", testLessonID, nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, "Salom, darsni boshlaymiz", hist[0].Body)
}

func TestSend_Ownership(t *testing.T) {
	uc, _ := setup(t)
	_, err := uc.Send(context.Background(), "intruder", testLessonID, "hack", "")
	require.True(t, apperr.IsForbidden(err), "boshqa mentor chat yubora olmaydi")

	_, err = uc.History(context.Background(), "intruder", testLessonID, nil, 0)
	require.True(t, apperr.IsForbidden(err))
}

// O'QUVCHI xabari ham saqlanadi — avval faqat host xabari saqlanardi va
// o'quvchi yozgani hech qayerda qolmasdi.
func TestSendFromRoom_SaqlanadiVaTarqaladi(t *testing.T) {
	uc, lk := setup(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "Savolim bor", "")
	require.NoError(t, err)
	require.Equal(t, "guest_a", m.SenderIdentity)
	require.Equal(t, "Ali", m.SenderName)
	require.GreaterOrEqual(t, lk.Calls["SendData"], 1)

	hist, err := uc.HistoryForRoom(ctx, testLessonID, "guest_a", nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, "Savolim bor", hist[0].Body)
}

// Kech kirgan ishtirokchi ham OLDINGI ommaviy xabarlarni ko'radi.
func TestHistoryForRoom_KechKirganHamKoradi(t *testing.T) {
	uc, _ := setup(t)
	ctx := context.Background()
	_, err := uc.Send(ctx, "mentor1", testLessonID, "Dars boshlandi", "")
	require.NoError(t, err)
	_, err = uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "Salom", "")
	require.NoError(t, err)

	hist, err := uc.HistoryForRoom(ctx, testLessonID, "kech_kirgan", nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 2, "kech kirgan ikkala ommaviy xabarni ham ko'rishi kerak")
}

// ── Shaxsiy xabar: eng muhim qism ────────────────────────────────────────────

func TestDM_FaqatIkkiTomongaYetkaziladi(t *testing.T) {
	uc, lk := setup(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "Faqat sizga", "mentor1")
	require.NoError(t, err)
	require.NotNil(t, m.ToIdentity)
	require.Equal(t, "mentor1", *m.ToIdentity)

	// Yetkazish MANZILLI bo'lishi shart: ommaviy broadcast bo'lsa xabar boshqa
	// klientlarga ham borardi va "shaxsiy" so'zi yolg'on bo'lardi.
	require.Equal(t, 1, lk.Calls["SendDataTo"], "shaxsiy xabar manzilli yuborilishi kerak")
	require.Equal(t, 0, lk.Calls["SendData"], "shaxsiy xabar xonaga tarqalmasligi kerak")
	require.Len(t, lk.SentTo, 1)
	require.ElementsMatch(t, []string{"mentor1", "guest_a"}, lk.SentTo[0],
		"ikkala tomon ham olishi kerak (yuboruvchi o'z yozganini ko'rsin)")

	// Yuborilgan mazmun ham to'g'ri shaklda bo'lsin (klient bilan shartnoma).
	var sent entity.ChatMessage
	require.NoError(t, json.Unmarshal(lk.Sent[0], &sent))
	require.Equal(t, "Faqat sizga", sent.Body)
	require.NotNil(t, sent.ToIdentity)
}

func TestDM_BegonagaKorinmaydi(t *testing.T) {
	uc, _ := setup(t)
	ctx := context.Background()

	_, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "Maxfiy", "mentor1")
	require.NoError(t, err)

	// Yuboruvchi ko'radi.
	hist, err := uc.HistoryForRoom(ctx, testLessonID, "guest_a", nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)

	// Qabul qiluvchi ko'radi.
	hist, err = uc.History(ctx, "mentor1", testLessonID, nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)

	// UCHINCHI shaxs KO'RMAYDI — bu filtr SQL/repo qatlamida, ya'ni begona
	// yozishma umuman o'qilmaydi.
	hist, err = uc.HistoryForRoom(ctx, testLessonID, "guest_b", nil, 0)
	require.NoError(t, err)
	require.Empty(t, hist, "begona shaxsiy xabar ko'rinmasligi kerak")
}

func TestDM_OzigaYozishOmmaviyDebQaraladi(t *testing.T) {
	uc, _ := setup(t)
	// O'ziga yozish ma'nosiz: uni DM sifatida saqlash "hech kim ko'rmaydigan"
	// xabar yasardi. Ommaviy deb qaraymiz.
	m, err := uc.SendFromRoom(context.Background(), testLessonID, "guest_a", "Ali", "test", "guest_a")
	require.NoError(t, err)
	require.Nil(t, m.ToIdentity)
}

func TestSendFromRoom_TezlikChegarasi(t *testing.T) {
	uc, _ := setup(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := uc.SendFromRoom(ctx, testLessonID, "spammer", "S", "xabar", "")
		require.NoError(t, err, "birinchi 5 ta xabar o'tishi kerak")
	}
	_, err := uc.SendFromRoom(ctx, testLessonID, "spammer", "S", "6-chi", "")
	require.Error(t, err, "6-chi xabar rad etilishi kerak")

	// Boshqa ishtirokchi cheklanmaydi (cheklov har identity uchun alohida).
	_, err = uc.SendFromRoom(ctx, testLessonID, "boshqa", "B", "salom", "")
	require.NoError(t, err)
}

func TestSendFromRoom_YaroqsizID(t *testing.T) {
	uc, _ := setup(t)
	// Ochiq endpoint: yaroqsiz UUID Postgres'ga yetib 500 generatoriga aylanmasin.
	_, err := uc.SendFromRoom(context.Background(), "abc", "guest_a", "Ali", "x", "")
	require.Error(t, err)
	_, err = uc.HistoryForRoom(context.Background(), "abc", "guest_a", nil, 0)
	require.Error(t, err)
}
