package room_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/roomstate"
)

// testLessonID — haqiqiy UUID: usecase ID formatini tekshiradi (shared.ValidateID).
const testLessonID = "11111111-1111-4111-8111-111111111111"

// fakeRecorder — dars yakunlanganda yozuvni to'xtatish chaqiruvini qayd etadi.
type fakeRecorder struct {
	stopped chan string
}

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{stopped: make(chan string, 4)}
}

func (f *fakeRecorder) StopActiveForLesson(_ context.Context, lessonID string) error {
	f.stopped <- lessonID
	return nil
}

func setupWithRecorder(t *testing.T) (room.UseCase, *testutil.FakeLessonRepo, *testutil.FakeLiveKit, *fakeRecorder) {
	t.Helper()
	rec := newFakeRecorder()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusScheduled}))
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	// Haqiqiy roomstate ulanadi (nil emas): "ruxsat berilganda qo'l tushadi" va
	// "dars tugaganda qo'llar tozalanadi" qoidalari aynan shu integratsiyada yashaydi.
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	return room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), rec, hands, testutil.NewFakeBlocklistRepo()), lrepo, lk, rec
}

func setup(t *testing.T) (room.UseCase, *testutil.FakeLessonRepo, *testutil.FakeLiveKit) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusScheduled}))
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc := room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), nil, hands, testutil.NewFakeBlocklistRepo())
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
	require.NoError(t, uc.MuteAll(context.Background(), "mentor1", testLessonID, nil))
	// host (mentor1) mute qilinmaydi → 2 ta guest mute qilinadi.
	require.Equal(t, 2, lk.Calls["MuteParticipant"])
}

func TestHostControls_Ownership(t *testing.T) {
	uc, _, _ := setup(t)
	require.True(t, apperr.IsForbidden(uc.MuteAll(context.Background(), "intruder", testLessonID, nil)))
	require.True(t, apperr.IsForbidden(uc.RemoveParticipant(context.Background(), "intruder", testLessonID, "g", "")))
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
		require.True(t, apperr.IsNotFound(uc.MuteAll(ctx, "mentor1", id, nil)), "MuteAll(%q) → 404 kutilgan", id)
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
	requireBadRequest(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "mentor1", ""),
		"host xonadan chiqarilmasligi kerak")
	require.Equal(t, 0, lk.Calls["RemoveParticipant"])

	// Regressiya: oddiy student ustida uchala amal ham AVVALGIDEK ishlaydi.
	require.NoError(t, uc.SetSpeakPermission(ctx, "mentor1", testLessonID, "guest_a", true))
	require.NoError(t, uc.MuteParticipant(ctx, "mentor1", testLessonID, "guest_a", true))
	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "guest_a", ""))
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

// ─── Majburiy yozib olish ────────────────────────────────────────────────────

func TestHostToken_DoesNotStartRecording(t *testing.T) {
	// ⭐ 5 DAQIQALIK TUZOQ. Egress xonaga kirib media kutadi va 5 daqiqada
	// hech kim chiqarmasa `egress_aborted "Start signal not received"` bilan
	// bekor bo'ladi (jonli serverda o'lchangan). Token berilishi bilan media
	// paydo bo'lishi orasida ruxsat so'rash va ulanish bor — ulanish yiqilsa
	// egress bo'sh Chrome aylantirib turardi va yozuv umuman qolmasdi.
	//
	// Shuning uchun yozuv BU YERDA boshlanmaydi: u `track_published`
	// webhook'ida boshlanadi. Bu test o'sha qarorni qotiradi.
	uc, lrepo, _, _ := setupWithRecorder(t)

	_, err := uc.HostToken(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)

	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	require.Equal(t, entity.LessonStatusLive, l.Status, "dars baribir jonli bo'ladi")
}

func TestEndLesson_StopsRecording(t *testing.T) {
	uc, _, _, rec := setupWithRecorder(t)
	_, err := uc.HostToken(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)

	require.NoError(t, uc.EndLesson(context.Background(), "mentor1", testLessonID))

	select {
	case id := <-rec.stopped:
		require.Equal(t, testLessonID, id)
	case <-time.After(2 * time.Second):
		t.Fatal("dars yakunlanganda yozuv to'xtatilmadi")
	}
}

func TestHostToken_NilRecorderIsSafe(t *testing.T) {
	// Yozib olish sozlanmagan muhit (masalan lokal dev) ilovani yiqitmasligi kerak.
	uc, _, _ := setup(t)
	_, err := uc.HostToken(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)
}

// ── F4 xavfsizlik: chiqarish = ban ──────────────────────────────────────────

// Chiqarilgan ishtirokchi QAYTIB KIRA OLMAYDI. Avval `RemoveParticipant` faqat
// joriy ulanishni uzardi: token hali yaroqli va `auto_create` yoqilgan bo'lgani
// uchun buzg'unchi darhol qaytib ulanar, hatto yopilgan xonani qayta yaratardi.
func TestRemoveParticipant_BansFromRejoining(t *testing.T) {
	uc, lrepo, _ := setup(t)
	ctx := context.Background()
	lesson, err := lrepo.GetByID(ctx, testLessonID)
	require.NoError(t, err)

	// Chiqarishdan OLDIN token beriladi.
	_, err = uc.ParticipantToken(ctx, lesson, "buzgunchi", "Buzg'unchi")
	require.NoError(t, err)

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", ""))

	// Chiqarishdan KEYIN token berilmaydi.
	_, err = uc.ParticipantToken(ctx, lesson, "buzgunchi", "Buzg'unchi")
	require.True(t, apperr.IsForbidden(err), "chiqarilgan ishtirokchi qayta token ololmasligi kerak")

	// Boshqalar ta'sirlanmaydi.
	_, err = uc.ParticipantToken(ctx, lesson, "oddiy_oquvchi", "Ali")
	require.NoError(t, err, "ban faqat chiqarilgan kishiga tegishli")
}

// Ban DARSGA bog'langan: bir darsdan chiqarilgan boshqa darsga kira oladi.
func TestRemoveParticipant_BanIsPerLesson(t *testing.T) {
	uc, lrepo, _ := setup(t)
	ctx := context.Background()

	other := &entity.Lesson{ID: "22222222-2222-4222-8222-222222222222", MentorID: "mentor1", Status: entity.LessonStatusLive}
	require.NoError(t, lrepo.Create(ctx, other))

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "ali", ""))

	_, err := uc.ParticipantToken(ctx, other, "ali", "Ali")
	require.NoError(t, err, "boshqa darsga kirish bloklanmasligi kerak")
}

// MuteAll parallel bo'lgach ham HOST'ga tegmasligi va hammani qamrashi kerak.
func TestMuteAll_SkipsHostAndCoversEveryone(t *testing.T) {
	uc, _, lk := setup(t)
	lk.Participants = []entity.RoomParticipant{
		{Identity: "mentor1"}, {Identity: "u1"}, {Identity: "u2"}, {Identity: "u3"},
	}
	require.NoError(t, uc.MuteAll(context.Background(), "mentor1", testLessonID, nil))
	require.Equal(t, 3, lk.Calls["MuteParticipant"], "host'dan tashqari hamma mute qilinishi kerak")
}
