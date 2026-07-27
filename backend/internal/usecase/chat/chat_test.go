package chat_test

import (
	"context"
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
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", Email: "m@x.uz", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Title: "X", Status: entity.LessonStatusLive}))
	lk := testutil.NewFakeLiveKit() // enabled mock — SendData chaqiruvini assert qilamiz
	uc := chat.New(testutil.NewFakeChatRepo(), lrepo, urepo, lk, testutil.NewLogger())
	return uc, lk
}

func TestSendAndHistory(t *testing.T) {
	uc, lk := setup(t)
	ctx := context.Background()

	m, err := uc.Send(ctx, "mentor1", testLessonID, "Salom, darsni boshlaymiz")
	require.NoError(t, err)
	require.Equal(t, "Dilnoza", m.SenderName)
	require.Equal(t, "mentor1", m.SenderIdentity)
	require.GreaterOrEqual(t, lk.Calls["SendData"], 1, "chat LiveKit data-channel orqali broadcast qilinishi kerak")

	hist, err := uc.History(ctx, "mentor1", testLessonID, nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, "Salom, darsni boshlaymiz", hist[0].Body)
}

func TestSend_Ownership(t *testing.T) {
	uc, _ := setup(t)
	_, err := uc.Send(context.Background(), "intruder", testLessonID, "hack")
	require.True(t, apperr.IsForbidden(err), "boshqa mentor chat yubora olmaydi")

	_, err = uc.History(context.Background(), "intruder", testLessonID, nil, 0)
	require.True(t, apperr.IsForbidden(err))
}
