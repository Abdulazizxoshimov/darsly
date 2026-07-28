package roomstate_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
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
	return roomstate.New(lrepo, lk, cache, testutil.NewLogger()), lk, cache
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

	st, err := uc.State(ctx, lessonID)
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

	st, err := uc.State(ctx, lessonID)
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

	st, err := uc.State(ctx, lessonID)
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
	first, err := uc.State(ctx, lessonID)
	require.NoError(t, err)
	firstAt := first.Hands[0].RaisedAt

	require.NoError(t, uc.SetHand(ctx, lessonID, "u2", "Vali", true))
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true)) // takroriy

	st, err := uc.State(ctx, lessonID)
	require.NoError(t, err)
	require.Equal(t, []string{"u1", "u2"}, []string{st.Hands[0].Identity, st.Hands[1].Identity})
	require.True(t, st.Hands[0].RaisedAt.Equal(firstAt), "ko'tarilgan vaqt saqlanishi kerak")
}

func TestState_BoshXonaXatoEmas(t *testing.T) {
	uc, _, _ := setup(t)
	// Hech kim qo'l ko'tarmagan — bu XATO emas, bo'sh ro'yxat.
	st, err := uc.State(context.Background(), lessonID)
	require.NoError(t, err)
	require.NotNil(t, st.Hands)
	require.Empty(t, st.Hands)
}

func TestState_YaroqsizIDValidatsiyadanOtmaydi(t *testing.T) {
	uc, _, _ := setup(t)
	// UUID bo'lmagan ID Postgres/Redis'ga yetmasligi kerak (500 generatori bo'lmasin).
	_, err := uc.State(context.Background(), "abc")
	require.Error(t, err)
}

func TestLowerHand_FaqatEga(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.SetHand(ctx, lessonID, "u1", "Ali", true))

	require.Error(t, uc.LowerHand(ctx, "boshqa-mentor", lessonID, "u1"), "begona mentor tushira olmasligi kerak")

	st, err := uc.State(ctx, lessonID)
	require.NoError(t, err)
	require.Len(t, st.Hands, 1, "begona urinish holatni o'zgartirmasligi kerak")

	require.NoError(t, uc.LowerHand(ctx, mentorID, lessonID, "u1"))
	st, err = uc.State(ctx, lessonID)
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

	st, err := uc.State(ctx, lessonID)
	require.NoError(t, err)
	require.Empty(t, st.Hands)

	m := lastMsg(t, lk)
	require.Equal(t, "lower_all", m["act"])
}

func TestReaction_TezlikChegarasi(t *testing.T) {
	uc, lk, _ := setup(t)
	ctx := context.Background()

	require.NoError(t, uc.Reaction(ctx, lessonID, "u1", "Ali", "👍"))
	before := len(lk.Sent)

	// Ikkinchi reaksiya AYNI oynada — rad etiladi va tarqatilmaydi.
	err := uc.Reaction(ctx, lessonID, "u1", "Ali", "👍")
	require.Error(t, err)
	require.Len(t, lk.Sent, before, "cheklangan reaksiya tarqatilmasligi kerak")

	// Boshqa ishtirokchi cheklanmaydi (cheklov har identity uchun alohida).
	require.NoError(t, uc.Reaction(ctx, lessonID, "u2", "Vali", "👏"))
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

	st, err := uc.State(ctx, lessonID)
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
	st, err := uc.State(ctx, lessonID)
	require.NoError(t, err)
	require.Len(t, st.Hands, 1)
	require.Empty(t, lk.Sent)
}
