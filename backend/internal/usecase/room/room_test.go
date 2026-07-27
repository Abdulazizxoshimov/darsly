package room_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/room"
)

// testLessonID — haqiqiy UUID: usecase ID formatini tekshiradi (shared.ValidateID).
const testLessonID = "11111111-1111-4111-8111-111111111111"

func setup(t *testing.T) (room.UseCase, *testutil.FakeLessonRepo, *testutil.FakeLiveKit) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusScheduled}))
	lk := testutil.NewFakeLiveKit()
	uc := room.New(lrepo, urepo, lk, testutil.NewFakeCache(), testutil.NewLogger())
	return uc, lrepo, lk
}

func TestHostToken_MakesLessonLive(t *testing.T) {
	uc, lrepo, lk := setup(t)
	rt, err := uc.HostToken(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RoomRoleHost, rt.Role)
	require.NotEmpty(t, rt.Token)
	require.GreaterOrEqual(t, lk.Calls["EnsureRoom"], 1, "xona yaratilishi kerak")

	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	require.Equal(t, entity.LessonStatusLive, l.Status, "host token dars'ni live qiladi")
	require.NotNil(t, l.StartedAt)
}

func TestHostToken_Ownership(t *testing.T) {
	uc, _, _ := setup(t)
	_, err := uc.HostToken(context.Background(), "intruder", testLessonID)
	require.True(t, apperr.IsForbidden(err), "faqat egasi host token oladi")
}

func TestHostToken_LiveKitDisabled(t *testing.T) {
	uc, _, lk := setup(t)
	lk.IsEnabled = false
	_, err := uc.HostToken(context.Background(), "mentor1", testLessonID)
	require.Error(t, err, "LiveKit o'chiq bo'lsa xato")
}

func TestParticipantToken(t *testing.T) {
	uc, lrepo, _ := setup(t)
	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	rt, err := uc.ParticipantToken(context.Background(), l, "guest_x", "Aziz")
	require.NoError(t, err)
	require.Equal(t, entity.RoomRoleParticipant, rt.Role)
	require.Equal(t, "guest_x", rt.Identity)
}

func TestEndLesson(t *testing.T) {
	uc, lrepo, lk := setup(t)
	require.NoError(t, uc.EndLesson(context.Background(), "mentor1", testLessonID))
	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	require.Equal(t, entity.LessonStatusEnded, l.Status)
	require.NotNil(t, l.EndedAt)
	require.GreaterOrEqual(t, lk.Calls["DeleteRoom"], 1, "xona yopilishi kerak")

	require.True(t, apperr.IsForbidden(uc.EndLesson(context.Background(), "intruder", testLessonID)))
}

func TestMuteAll_SkipsHost(t *testing.T) {
	uc, _, lk := setup(t)
	lk.Participants = []entity.RoomParticipant{
		{Identity: "mentor1"}, {Identity: "guest_a"}, {Identity: "guest_b"},
	}
	require.NoError(t, uc.MuteAll(context.Background(), "mentor1", testLessonID))
	// host (mentor1) mute qilinmaydi → 2 ta guest mute qilinadi.
	require.Equal(t, 2, lk.Calls["MuteParticipant"])
}

func TestHostControls_Ownership(t *testing.T) {
	uc, _, _ := setup(t)
	require.True(t, apperr.IsForbidden(uc.MuteAll(context.Background(), "intruder", testLessonID)))
	require.True(t, apperr.IsForbidden(uc.RemoveParticipant(context.Background(), "intruder", testLessonID, "g")))
	require.True(t, apperr.IsForbidden(uc.SetSpeakPermission(context.Background(), "intruder", testLessonID, "g", true)))
	_, err := uc.ListParticipants(context.Background(), "intruder", testLessonID)
	require.True(t, apperr.IsForbidden(err))
}

// Yaroqsiz UUID → 404 (avval 22P02 → 500: `POST /lessons/abc/token`).
func TestRoom_InvalidUUID_NotFound(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()

	for _, id := range []string{"abc", "", "lesson-1"} {
		_, err := uc.HostToken(ctx, "mentor1", id)
		require.True(t, apperr.IsNotFound(err), "HostToken(%q) → 404 kutilgan, oldi: %v", id, err)
		require.True(t, apperr.IsNotFound(uc.EndLesson(ctx, "mentor1", id)), "EndLesson(%q) → 404 kutilgan", id)
		require.True(t, apperr.IsNotFound(uc.MuteAll(ctx, "mentor1", id)), "MuteAll(%q) → 404 kutilgan", id)
		require.True(t, apperr.IsNotFound(uc.SetSpeakPermission(ctx, "mentor1", id, "g", true)), "SetSpeakPermission(%q) → 404 kutilgan", id)
		_, err = uc.ListParticipants(ctx, "mentor1", id)
		require.True(t, apperr.IsNotFound(err), "ListParticipants(%q) → 404 kutilgan, oldi: %v", id, err)
	}
}

// BE-13 yon ta'siri: allow-speak host identity'siga qo'llansa ustozning ekran
// ulashishi o'chib qolardi (CanPublishSources → [CAMERA, MICROPHONE]). Host identity
// = mentorID (HostToken'da AccessToken(..., mentorID, ...)). Moderatsiya amallari
// host'ning o'ziga qo'llanmasligi kerak.
func TestHostControls_CannotTargetHost(t *testing.T) {
	uc, _, lk := setup(t)
	ctx := context.Background()

	// allow-speak (yoqish va o'chirish — ikkalasi ham rad etilishi kerak)
	for _, canPublish := range []bool{true, false} {
		err := uc.SetSpeakPermission(ctx, "mentor1", testLessonID, "mentor1", canPublish)
		requireBadRequest(t, err, "host'ning publish huquqi o'zgartirilmasligi kerak")
	}
	require.Equal(t, 0, lk.Calls["SetParticipantPublish"], "LiveKit'ga chaqiruv umuman ketmasligi kerak")

	// mute (MuteAll host'ni o'tkazib yuboradi — izchillik)
	requireBadRequest(t, uc.MuteParticipant(ctx, "mentor1", testLessonID, "mentor1", true),
		"host mute qilinmasligi kerak")

	// remove (aks holda dars host'siz "live" qolardi)
	requireBadRequest(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "mentor1"),
		"host xonadan chiqarilmasligi kerak")
	require.Equal(t, 0, lk.Calls["RemoveParticipant"])

	// Regressiya: oddiy student ustida uchala amal ham AVVALGIDEK ishlaydi.
	require.NoError(t, uc.SetSpeakPermission(ctx, "mentor1", testLessonID, "guest_a", true))
	require.NoError(t, uc.MuteParticipant(ctx, "mentor1", testLessonID, "guest_a", true))
	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "guest_a"))
	require.Equal(t, 1, lk.Calls["SetParticipantPublish"])
	require.Equal(t, 1, lk.Calls["RemoveParticipant"])
}

// requireBadRequest — apperr'da IsBadRequest yordamchisi yo'q (faqat IsValidation),
// shuning uchun kodni As() bilan tekshiramiz (shared paketga yangi API qo'shmaymiz).
func requireBadRequest(t *testing.T, err error, msg string) {
	t.Helper()
	require.Error(t, err, msg)
	ae := apperr.As(err)
	require.NotNil(t, ae, "%s: AppError kutilgan, oldi: %v", msg, err)
	require.Equal(t, apperr.CodeBadRequest, ae.Code, "%s: kutilgan 400 BAD_REQUEST, oldi: %v", msg, ae.Code)
}
