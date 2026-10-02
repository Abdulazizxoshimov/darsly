package roomstate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/roomstate"
	"github.com/zoom/darsly/internal/usecase/shared"
)

const (
	lessonID = "11111111-1111-1111-1111-111111111111"
	mentorID = "mentor1"
)

func setup(t *testing.T) (roomstate.UseCase, *testutil.FakeLiveKit, *testutil.FakeCache) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{
		ID: lessonID, MentorID: mentorID, Status: entity.LessonStatusLive,
	}))
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	return roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger()), lk, cache
}

// lastMsg — oxirgi tarqatilgan xabarni JSON map sifatida qaytaradi.
// Xabar shakli KLIENT bilan shartnoma (`frontend/src/livekit/messaging.js`),
// shuning uchun u aynan tekshiriladi.
func lastMsg(t *testing.T, lk *testutil.FakeLiveKit) map[string]any {
	t.Helper()
	require.NotEmpty(t, lk.Sent, "hech narsa tarqatilmadi")
	var m map[string]any
	require.NoError(t, json.Unmarshal(lk.Sent[len(lk.Sent)-1], &m))
	return m
}

func TestSetHand_SaqlaydiVaTarqatadi(t *testing.T) {
	uc, lk, _ := setup(t)
	ctx := context.Background()

	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Len(t, st.Hands, 1)
	require.Equal(t, "u1", st.Hands[0].Identity)
	require.Equal(t, "Ali", st.Hands[0].Name)

	m := lastMsg(t, lk)
	require.Equal(t, "hand", m["kind"])
	require.Equal(t, "u1", m["identity"])
	require.Equal(t, true, m["raised"])
	require.NotZero(t, m["at"], "navbat tartibi uchun `at` bo'lishi shart")
}

func TestSetHand_TushirishOchiradi(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", false))

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Empty(t, st.Hands)
}

// Navbat tartibi — mahsulot qoidasi: kim birinchi so'ragan bo'lsa birinchi turadi.
// Redis hash tartibsiz qaytaradi, shuning uchun tartib SERVERDA hisoblanadi.
func TestState_NavbatTartibiKotarilganVaqtBoyicha(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u2", "Vali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u3", "Guli", true))

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Len(t, st.Hands, 3)
	require.Equal(t, []string{"u1", "u2", "u3"},
		[]string{st.Hands[0].Identity, st.Hands[1].Identity, st.Hands[2].Identity})
}

// Takroriy "ko'tarish" navbatdagi o'rinni buzmasligi kerak, aks holda o'quvchi
// tugmani ikki marta bosib navbatda oxiriga tushib qolardi.
func TestSetHand_TakroriyKotarishOrinniBuzmaydi(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	first, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	firstAt := first.Hands[0].RaisedAt

	require.NoError(t, uc.SetHand(ctx, lessonID, "u2", "Vali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true)) // takroriy

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Equal(t, []string{"u1", "u2"}, []string{st.Hands[0].Identity, st.Hands[1].Identity})
	require.True(t, st.Hands[0].RaisedAt.Equal(firstAt), "ko'tarilgan vaqt saqlanishi kerak")
}

func TestState_BoshXonaXatoEmas(t *testing.T) {
	uc, _, _ := setup(t)
	// Hech kim qo'l ko'tarmagan — bu XATO emas, bo'sh ro'yxat.
	st, err := uc.State(context.Background(), lessonID, "viewer")
	require.NoError(t, err)
	require.NotNil(t, st.Hands)
	require.Empty(t, st.Hands)
}

func TestState_YaroqsizIDValidatsiyadanOtmaydi(t *testing.T) {
	uc, _, _ := setup(t)
	// UUID bo'lmagan ID Postgres/Redis'ga yetmasligi kerak (500 generatori bo'lmasin).
	_, err := uc.State(context.Background(), "abc", "viewer")
	require.Error(t, err)
}

func TestLowerHand_FaqatEga(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))

	require.Error(t, uc.LowerHand(ctx, "boshqa-mentor", lessonID, "u1"), "begona mentor tushira olmasligi kerak")

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Len(t, st.Hands, 1, "begona urinish holatni o'zgartirmasligi kerak")

	require.NoError(t, uc.LowerHand(ctx, mentorID, lessonID, "u1"))
	st, err = uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Empty(t, st.Hands)
}

func TestLowerAll_HammasiniTozalaydi(t *testing.T) {
	uc, lk, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u2", "Vali", true))

	require.Error(t, uc.LowerAll(ctx, "boshqa-mentor", lessonID))
	require.NoError(t, uc.LowerAll(ctx, mentorID, lessonID))

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Empty(t, st.Hands)

	m := lastMsg(t, lk)
	require.Equal(t, "lower_all", m["act"])
}

func TestReaction_TezlikChegarasi(t *testing.T) {
	uc, lk, _ := setup(t)
	ctx := context.Background()

	// Portlash (qarsak) O'TISHI kerak — 10 s oynada 5 ta.
	for i := 0; i < 5; i++ {
		require.NoError(t, uc.Reaction(ctx, lessonID, "u1", "Ali", "👏"),
			"ketma-ket qarsak bloklanmasligi kerak (%d-chi)", i+1)
	}
	before := len(lk.Sent)

	// 6-chi AYNI oynada — rad etiladi va tarqatilmaydi.
	err := uc.Reaction(ctx, lessonID, "u1", "Ali", "👍")
	require.Error(t, err)
	require.Len(t, lk.Sent, before, "cheklangan reaksiya tarqatilmasligi kerak")

	// Boshqa ishtirokchi cheklanmaydi (cheklov har identity uchun alohida).
	require.NoError(t, uc.Reaction(ctx, lessonID, "u2", "Vali", "👏"))
}

// Reaksiya kanali moderatsiyasiz MATN kanaliga aylanmasligi kerak: endpoint
// ochiq va `curl` bilan istalgan 16 baytlik satr yuborilardi.
func TestReaction_FaqatRuxsatEtilganEmoji(t *testing.T) {
	uc, lk, _ := setup(t)
	ctx := context.Background()

	for _, bad := range []string{"haqorat", "<img>", "💣", "", "👍👍"} {
		err := uc.Reaction(ctx, lessonID, "u1", "Ali", bad)
		require.Error(t, err, "ruxsatsiz reaksiya rad etilishi kerak: %q", bad)
		require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)
	}
	require.Empty(t, lk.Sent, "rad etilgan reaksiya tarqatilmasligi kerak")

	// Nazorat: ruxsat etilgani o'tadi (test "hammasini bloklash" fixini
	// yashil ko'rsatmasin).
	require.NoError(t, uc.Reaction(ctx, lessonID, "u2", "Vali", "❤️"))
}

func TestReaction_XabarShakli(t *testing.T) {
	uc, lk, _ := setup(t)
	require.NoError(t, uc.Reaction(context.Background(), lessonID, "u1", "Ali", "🎉"))
	m := lastMsg(t, lk)
	require.Equal(t, "reaction", m["kind"])
	require.Equal(t, "🎉", m["emoji"])
	require.Equal(t, "Ali", m["name"])
}

func TestBroadcast_XonaNomiDarsdanKelibChiqadi(t *testing.T) {
	uc, lk, _ := setup(t)
	require.NoError(t, uc.SetHand(context.Background(), lessonID, "u1", "Ali", true))
	// Xona nomi `shared.RoomName` bo'yicha — klient token'idagi xona bilan bir xil
	// bo'lishi shart, aks holda xabar boshqa xonaga ketardi.
	require.Equal(t, 1, lk.Calls["SendData"])
	require.NotEmpty(t, shared.RoomName(lessonID))
}

func TestClear_HolatniOchiradi(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	require.NoError(t, uc.Clear(ctx, lessonID))

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Empty(t, st.Hands)
}

// LiveKit o'chirilgan bo'lsa ham holat saqlanishi kerak: tarqatish "eng yaxshi
// harakat", haqiqat manbai esa server.
func TestSetHand_LiveKitOchiqBolsaHamSaqlanadi(t *testing.T) {
	uc, lk, _ := setup(t)
	lk.IsEnabled = false
	ctx := context.Background()

	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))
	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.Len(t, st.Hands, 1)
	require.Empty(t, lk.Sent)
}

// ⭐ Xona holati ovoz siyosatini ham qaytarishi kerak.
//
// Siyosat dars O'RTASIDA o'zgaradi ("Hammani o'chirish" checkbox'i). O'zgarish
// paytida ulanmagan yoki qayta ulangan o'quvchi uni data-message'dan ololmaydi —
// bir martalik xabar o'tib ketgan. Xona holati esa aynan "kech kelgan klient
// shu bilan tiklanadi" uchun bor.
func TestState_ReturnsAudioPolicy(t *testing.T) {
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: lessonID, MentorID: mentorID, Status: entity.LessonStatusLive,
		MuteOnEntry: true, AllowSelfUnmute: false,
	}))
	uc := roomstate.New(lrepo, testutil.NewFakeLiveKit(), testutil.NewFakeCache(), nil, testutil.NewLogger())

	st, err := uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.True(t, st.MuteOnEntry)
	require.False(t, st.AllowSelfUnmute, "unmute taqiqi holatda ko'rinishi kerak")

	// Ustoz siyosatni dars o'rtasida yumshatdi — holat YANGI qiymatni bersin
	// (kech ulangan klient eski siyosatga tushib qolmasin).
	l, err := lrepo.GetByID(ctx, lessonID)
	require.NoError(t, err)
	l.AllowSelfUnmute = true
	require.NoError(t, lrepo.Update(ctx, l))

	st, err = uc.State(ctx, lessonID, "viewer")
	require.NoError(t, err)
	require.True(t, st.AllowSelfUnmute, "o'zgargan siyosat holatda aks etishi kerak")
}

// Dars o'qib bo'lmasa siyosat RUXSAT BERUVCHI bo'lishi kerak: mavjud bo'lmagan
// taqiqni ko'rsatish (tugma o'chiq, lekin server hech narsani mute qilmaydi)
// — bu real cheklovsiz real zarar.
func TestState_UnknownLesson_PermissivePolicy(t *testing.T) {
	uc, _, _ := setup(t)
	st, err := uc.State(context.Background(), "22222222-2222-2222-2222-222222222222", "viewer")
	require.NoError(t, err)
	require.True(t, st.AllowSelfUnmute, "dars topilmasa mikrofon tugmasi o'chib qolmasin")
	require.False(t, st.MuteOnEntry)
}

// Chiqarilgan (ban) ishtirokchi holatni (qo'llar/yozuv) o'qiy olmaydi; boshqasi oladi.
func TestState_BannedIdentity_Forbidden(t *testing.T) {
	uc, _, cache := setup(t)
	ctx := context.Background()
	require.NoError(t, shared.Ban(ctx, cache, lessonID, "kicked"))

	_, err := uc.State(ctx, lessonID, "kicked")
	require.True(t, apperr.IsForbidden(err), "banlangan State o'qiy olmasligi kerak: %v", err)

	_, err = uc.State(ctx, lessonID, "other")
	require.NoError(t, err)
}
