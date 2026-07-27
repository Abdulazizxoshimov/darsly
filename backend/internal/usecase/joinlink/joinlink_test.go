package joinlink_test

import (
	"context"
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

func ptr[T any](v T) *T { return &v }

func setup(t *testing.T) (joinlink.UseCase, *testutil.FakeLessonRepo, *testutil.FakeUserRepo) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", Email: "m@x.uz", FullName: "Dilnoza Mentor", Role: "mentor"}))
	h := hasher.New(4)
	roomUC := &testutil.FakeRoomUC{}
	cache := testutil.NewFakeCache()
	waitUC := waitingroom.New(testutil.NewFakeWaitingRepo(), lrepo, roomUC, ws.NewHub(testutil.NewLogger()), cache, testutil.NewLogger())
	uc := joinlink.New(lrepo, urepo, h, cache, roomUC, waitUC, testutil.NewLogger())
	return uc, lrepo, urepo
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

	resp, err := uc.Join(context.Background(), "s1", &entity.JoinLessonReq{GuestName: ptr("Aziz")})
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
	_, err := uc.Join(context.Background(), "s2", &entity.JoinLessonReq{})
	require.True(t, errorsIsUnauthorized(err), "parol talab qilinadi")

	// Noto'g'ri parol → 401
	_, err = uc.Join(context.Background(), "s2", &entity.JoinLessonReq{Passcode: ptr("0000")})
	require.True(t, errorsIsUnauthorized(err))

	// To'g'ri parol → token
	resp, err := uc.Join(context.Background(), "s2", &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.NoError(t, err)
	require.NotNil(t, resp.Room)
}

// Ko'p marta noto'g'ri parol → slug qulflanadi, to'g'ri parol ham rad etiladi.
func TestJoin_PasscodeBruteForceLockout(t *testing.T) {
	uc, lrepo, _ := setup(t)
	h := hasher.New(4)
	hash, _ := h.Hash("1234")
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "bf", PasscodeHash: &hash})

	for i := range 5 {
		_, err := uc.Join(context.Background(), "bf", &entity.JoinLessonReq{Passcode: ptr("0000")})
		require.Error(t, err, "urinish %d noto'g'ri", i+1)
	}

	// Endi to'g'ri parol ham qulf tufayli rad etiladi.
	_, err := uc.Join(context.Background(), "bf", &entity.JoinLessonReq{Passcode: ptr("1234")})
	require.True(t, apperr.IsForbidden(err), "5 urinishdan keyin slug qulflanishi kerak")
}

func TestJoin_WaitingRoom(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s3", IsWaitingRoomEnabled: true})

	resp, err := uc.Join(context.Background(), "s3", &entity.JoinLessonReq{GuestName: ptr("Laylo")})
	require.NoError(t, err)
	require.Equal(t, "waiting_room", resp.NextStep)
	require.NotEmpty(t, resp.RequestID, "kutish xonasi → request_id qaytadi")
	require.Nil(t, resp.Room, "token admit'dan keyin beriladi")
}

func TestJoin_Locked(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s4", IsLocked: true})
	_, err := uc.Join(context.Background(), "s4", &entity.JoinLessonReq{})
	require.True(t, apperr.IsForbidden(err), "qulflangan darsga kirib bo'lmaydi")
}

func TestJoin_Ended(t *testing.T) {
	uc, lrepo, _ := setup(t)
	seedLesson(t, lrepo, &entity.Lesson{ID: "l1", MentorID: "mentor1", JoinSlug: "s5", Status: entity.LessonStatusEnded})
	_, err := uc.Join(context.Background(), "s5", &entity.JoinLessonReq{})
	require.Error(t, err, "tugagan darsga kirib bo'lmaydi")
}

func errorsIsUnauthorized(err error) bool {
	ae := apperr.As(err)
	return ae != nil && ae.HTTPStatus == 401
}
