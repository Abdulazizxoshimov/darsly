package waitingroom_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

func setup(t *testing.T) (waitingroom.UseCase, *testutil.FakeWaitingRepo, *testutil.FakeLessonRepo, *entity.Lesson) {
	uc, wrepo, lrepo, lesson, _ := setupWithCache(t)
	return uc, wrepo, lrepo, lesson
}

// setupWithCache — cache'ga ham kirish beradi (token TTL tugashini modellash uchun).
func setupWithCache(t *testing.T) (waitingroom.UseCase, *testutil.FakeWaitingRepo, *testutil.FakeLessonRepo, *entity.Lesson, *testutil.FakeCache) {
	t.Helper()
	wrepo := testutil.NewFakeWaitingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	// Fake lesson repo UUID tekshiruvidan o'tishi uchun ID haqiqiy UUID bo'lishi kerak
	// (shared.ValidateID — yaroqsiz UUID DB'ga yetmasin).
	lesson := &entity.Lesson{ID: uuid.NewString(), MentorID: "mentor1", Status: entity.LessonStatusLive, IsWaitingRoomEnabled: true}
	require.NoError(t, lrepo.Create(context.Background(), lesson))
	hub := ws.NewHub(testutil.NewLogger())
	cache := testutil.NewFakeCache()
	uc := waitingroom.New(wrepo, lrepo, &testutil.FakeRoomUC{}, hub, cache, testutil.NewLogger())
	return uc, wrepo, lrepo, lesson, cache
}

func TestCreateAndAdmit(t *testing.T) {
	uc, _, _, lesson := setup(t)
	req, err := uc.CreateRequest(context.Background(), lesson, "Kamola")
	require.NoError(t, err)
	require.Equal(t, entity.WaitingStatusPending, req.Status)
	require.NotEmpty(t, req.GuestIdentity)

	rt, err := uc.Admit(context.Background(), "mentor1", req.ID)
	require.NoError(t, err)
	require.Equal(t, "part-tok", rt.Token)

	// Status endi admitted va token cache'da.
	st, err := uc.Status(context.Background(), req.ID)
	require.NoError(t, err)
	require.Equal(t, entity.WaitingStatusAdmitted, st.Status)
	require.NotNil(t, st.Room, "admit'dan keyin token cache'da bo'lishi kerak")
}

func TestAdmit_Idempotent_NoDoubleToken(t *testing.T) {
	uc, _, _, lesson := setup(t)
	req, _ := uc.CreateRequest(context.Background(), lesson, "X")

	_, err := uc.Admit(context.Background(), "mentor1", req.ID)
	require.NoError(t, err)

	// Ikkinchi admit — atomik transition tufayli Conflict (ikkinchi token BERILMAYDI).
	_, err = uc.Admit(context.Background(), "mentor1", req.ID)
	require.Error(t, err)
	require.True(t, apperr.IsConflict(err), "ikkinchi admit → 409 (TOCTOU himoyasi)")
}

func TestAdmit_Ownership(t *testing.T) {
	uc, _, _, lesson := setup(t)
	req, _ := uc.CreateRequest(context.Background(), lesson, "X")
	_, err := uc.Admit(context.Background(), "boshqa-mentor", req.ID)
	require.True(t, apperr.IsForbidden(err), "boshqa mentor admit qila olmaydi")
}

func TestReject(t *testing.T) {
	uc, _, _, lesson := setup(t)
	req, _ := uc.CreateRequest(context.Background(), lesson, "X")
	require.NoError(t, uc.Reject(context.Background(), "mentor1", req.ID))

	st, _ := uc.Status(context.Background(), req.ID)
	require.Equal(t, entity.WaitingStatusRejected, st.Status)
	require.Nil(t, st.Room)

	// Rad etilgandan keyin admit → Conflict.
	_, err := uc.Admit(context.Background(), "mentor1", req.ID)
	require.True(t, apperr.IsConflict(err))
}

// Yaroqsiz request_id DB'ga yetmasligi kerak: Postgres 22P02 → 500 INTERNAL_ERROR
// bo'lib qaytardi va ochiq endpoint orqali har kim log/Sentry'ni to'ldira olardi.
// Kutilgan: NotFound (mavjud bo'lmagan ID bilan AYNAN bir xil javob → oracle yo'q).
func TestStatus_InvalidRequestID_NotFound(t *testing.T) {
	uc, _, _, _ := setup(t)

	cases := []struct {
		name string
		id   string
	}{
		{"harflar", "abc"},
		{"bo'sh", ""},
		{"kesilgan uuid", "550e8400-e29b-41d4-a716"},
		{"sql-injection urinishi", "' OR 1=1 --"},
		{"uuid + qo'shimcha", "550e8400-e29b-41d4-a716-446655440000x"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := uc.Status(context.Background(), tc.id)
			require.True(t, apperr.IsNotFound(err), "yaroqsiz id → 404 (500 emas), oldi: %v", err)
		})
	}

	// Mavjud bo'lmagan, LEKIN shakli to'g'ri UUID — javob aynan bir xil bo'lishi shart
	// (aks holda "shakl noto'g'ri" va "mavjud emas" farqlanib, enumeration oracle'i paydo bo'ladi).
	_, missingErr := uc.Status(context.Background(), "550e8400-e29b-41d4-a716-446655440000")
	_, malformedErr := uc.Status(context.Background(), "abc")
	require.Equal(t, missingErr.Error(), malformedErr.Error(), "yaroqsiz va mavjud emas javoblari farq qilmasin")
}

// Authed yo'llar ham 500 bermasin.
func TestAdmitReject_InvalidRequestID_NotFound(t *testing.T) {
	uc, _, _, _ := setup(t)

	_, err := uc.Admit(context.Background(), "mentor1", "abc")
	require.True(t, apperr.IsNotFound(err), "admit(yaroqsiz id) → 404")

	require.True(t, apperr.IsNotFound(uc.Reject(context.Background(), "mentor1", "abc")), "reject(yaroqsiz id) → 404")
}

// WS upgrade'dan keyingi goroutine yaroqsiz id bilan panic/xato bermasin.
func TestDeliverCurrentStatus_InvalidRequestID_NoPanic(t *testing.T) {
	uc, _, _, _ := setup(t)
	require.NotPanics(t, func() {
		uc.DeliverCurrentStatus(context.Background(), "abc")
	})
}

// BE-12: token cache TTL o'tib ketsa (guest fon rejimida uzoq turdi) status
// {admitted, room:null} qaytarib boshi berk ko'chaga olib bormasin — token
// DB'dagi GuestIdentity bilan qayta chiqarilishi kerak.
func TestStatus_TokenReissuedAfterCacheExpiry(t *testing.T) {
	uc, _, _, lesson, cache := setupWithCache(t)
	ctx := context.Background()

	req, err := uc.CreateRequest(ctx, lesson, "Kamola")
	require.NoError(t, err)
	rt, err := uc.Admit(ctx, "mentor1", req.ID)
	require.NoError(t, err)
	require.NotNil(t, rt)

	// TTL tugashini modellashtiramiz.
	require.NoError(t, cache.Del(ctx, "waitroom:token:"+req.ID))

	st, err := uc.Status(ctx, req.ID)
	require.NoError(t, err)
	require.Equal(t, entity.WaitingStatusAdmitted, st.Status)
	require.NotNil(t, st.Room, "cache tugagach token QAYTA chiqarilishi kerak (boshi berk ko'cha yo'q)")
	require.Equal(t, "part-tok", st.Room.Token)

	// Qayta chiqarilgan token cache'ga ham qaytadi (keyingi so'rov SFU'ga bormaydi).
	st2, err := uc.Status(ctx, req.ID)
	require.NoError(t, err)
	require.NotNil(t, st2.Room)
}

// Dars tugagan bo'lsa token QAYTA CHIQARILMAYDI (tugagan darsga kirish yo'li ochilmasin).
func TestStatus_NoReissueWhenLessonNotLive(t *testing.T) {
	uc, _, lrepo, lesson, cache := setupWithCache(t)
	ctx := context.Background()

	req, _ := uc.CreateRequest(ctx, lesson, "Kamola")
	_, err := uc.Admit(ctx, "mentor1", req.ID)
	require.NoError(t, err)
	require.NoError(t, cache.Del(ctx, "waitroom:token:"+req.ID))

	l, err := lrepo.GetByID(ctx, lesson.ID)
	require.NoError(t, err)
	l.Status = entity.LessonStatusEnded
	require.NoError(t, lrepo.Update(ctx, l))

	st, err := uc.Status(ctx, req.ID)
	require.NoError(t, err)
	require.Equal(t, entity.WaitingStatusAdmitted, st.Status)
	require.Nil(t, st.Room, "tugagan darsga token berilmasligi kerak")
}

// Rad etilgan so'rovga hech qanday holatda token chiqarilmaydi.
func TestStatus_NoTokenForRejected(t *testing.T) {
	uc, _, _, lesson, cache := setupWithCache(t)
	ctx := context.Background()

	req, _ := uc.CreateRequest(ctx, lesson, "Kamola")
	require.NoError(t, uc.Reject(ctx, "mentor1", req.ID))
	require.NoError(t, cache.Del(ctx, "waitroom:token:"+req.ID))

	st, err := uc.Status(ctx, req.ID)
	require.NoError(t, err)
	require.Equal(t, entity.WaitingStatusRejected, st.Status)
	require.Nil(t, st.Room)
}

func TestListPending_Ownership(t *testing.T) {
	uc, _, _, lesson := setup(t)
	_, _ = uc.CreateRequest(context.Background(), lesson, "A")
	_, _ = uc.CreateRequest(context.Background(), lesson, "B")

	items, err := uc.ListPending(context.Background(), "mentor1", lesson.ID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	_, err = uc.ListPending(context.Background(), "intruder", lesson.ID)
	require.True(t, apperr.IsForbidden(err))
}
