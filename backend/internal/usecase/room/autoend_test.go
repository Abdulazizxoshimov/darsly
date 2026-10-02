package room_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/roomstate"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// autoEndSetup — sweep testlari uchun: jonli dars + boshqariladigan LiveKit/kesh.
func autoEndSetup(t *testing.T, startedAgo time.Duration) (room.UseCase, *testutil.FakeLessonRepo, *testutil.FakeLiveKit, *testutil.FakeCache) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	started := time.Now().UTC().Add(-startedAgo)
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{
		ID:        testLessonID,
		MentorID:  "mentor1",
		Status:    entity.LessonStatusLive,
		StartedAt: &started,
		UpdatedAt: started,
	}))
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc := room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), nil, hands, testutil.NewFakeBlocklistRepo(), testutil.NewFakeBanRepo())
	return uc, lrepo, lk, cache
}

func lessonStatus(t *testing.T, lrepo *testutil.FakeLessonRepo) string {
	t.Helper()
	l, err := lrepo.GetByID(context.Background(), testLessonID)
	require.NoError(t, err)
	return l.Status
}

// 4 soatlik texnik limit: limitdan oshgan dars avtomatik yakunlanadi — mentor
// "Yakunlash" tugmasini bosmagan bo'lsa ham (telefon o'chgan, ilova yopilgan).
func TestSweepAutoEnd_MaxDurationEndsLesson(t *testing.T) {
	uc, lrepo, lk, _ := autoEndSetup(t, 5*time.Hour)
	// Xonada odam BOR — ya'ni yakunlash faqat limit tufayli bo'lishi kerak.
	lk.Participants = []entity.RoomParticipant{{Identity: "student1"}}

	require.Equal(t, 1, uc.SweepAutoEnd(context.Background(), 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusEnded, lessonStatus(t, lrepo))
}

// Limit ichidagi va odam bor dars TEGILMAYDI.
func TestSweepAutoEnd_ActiveLessonUntouched(t *testing.T) {
	uc, lrepo, lk, _ := autoEndSetup(t, 30*time.Minute)
	lk.Participants = []entity.RoomParticipant{{Identity: "student1"}}

	require.Equal(t, 0, uc.SweepAutoEnd(context.Background(), 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))
}

// Bo'sh xona: grace ICHIDA yakunlanmaydi (mentor 4G uzilishida qaytib kelishi
// mumkin), grace O'TGACH yakunlanadi.
func TestSweepAutoEnd_EmptyRoomWaitsForGrace(t *testing.T) {
	ctx := context.Background()
	uc, lrepo, lk, cache := autoEndSetup(t, time.Hour)
	lk.Participants = nil // xona bo'sh

	// Birinchi sweep faqat "bo'shlik boshlandi" belgisini qo'yadi.
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))

	// Grace hali o'tmagan.
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))

	// Belgini 25 daqiqa orqaga suramiz — grace o'tgan holat.
	require.NoError(t, cache.Set(ctx, "room:empty:"+testLessonID,
		time.Now().UTC().Add(-25*time.Minute).Unix(), time.Hour))

	require.Equal(t, 1, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusEnded, lessonStatus(t, lrepo))
}

// Xonaga kimdir qaytsa bo'shlik hisobi BEKOR bo'ladi (mentor uzilib, qayta
// ulangan holat — dars yakunlanmasligi shart).
func TestSweepAutoEnd_ParticipantReturnCancelsGrace(t *testing.T) {
	ctx := context.Background()
	uc, lrepo, lk, cache := autoEndSetup(t, time.Hour)
	lk.Participants = nil
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))

	// Mentor qaytdi.
	lk.Participants = []entity.RoomParticipant{{Identity: "mentor1"}}
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	v, _ := cache.Get(ctx, "room:empty:"+testLessonID)
	require.Empty(t, v, "xonada odam bo'lsa bo'shlik belgisi tozalanishi kerak")

	// Yana chiqdi — hisob NOLDAN boshlanadi, ya'ni darhol yakunlanmaydi.
	lk.Participants = nil
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))
}

// Idempotentlik: yakunlangan dars ikkinchi sweep'da qayta yakunlanmaydi
// (qayta ishga tushirilganda yoki ikki instansda dublikat bo'lmasin).
func TestSweepAutoEnd_Idempotent(t *testing.T) {
	ctx := context.Background()
	uc, lrepo, lk, _ := autoEndSetup(t, 5*time.Hour)
	lk.Participants = []entity.RoomParticipant{{Identity: "student1"}}

	require.Equal(t, 1, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute), "dublikat yakun bo'lmasligi kerak")
	require.Equal(t, entity.LessonStatusEnded, lessonStatus(t, lrepo))
}

// Avto-yakunda YOZUV ham yopilishi shart (aks holda egress serverda ishlab
// turaveradi va yozuv abadiy "recording" holatida qoladi).
func TestSweepAutoEnd_StopsRecording(t *testing.T) {
	ctx := context.Background()
	rec := newFakeRecorder()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	started := time.Now().UTC().Add(-5 * time.Hour)
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive,
		StartedAt: &started, UpdatedAt: started,
	}))
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc := room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), rec, hands, testutil.NewFakeBlocklistRepo(), testutil.NewFakeBanRepo())

	require.Equal(t, 1, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	select {
	case id := <-rec.stopped:
		require.Equal(t, testLessonID, id)
	case <-time.After(time.Second):
		t.Fatal("avto-yakunda yozuv to'xtatilmadi")
	}
}

// RECONCILIATION (audit R3): yozuv yoqilgan jonli darsda media bor, lekin
// yozuv ketmayapti bo'lsa (yo'qolgan `track_published` webhook) sweep uni
// o'zi boshlaydi.
func TestSweepAutoEnd_ReconcilesRecording(t *testing.T) {
	ctx := context.Background()
	rec := newFakeRecorder() // IsRecording=false (default)
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	started := time.Now().UTC().Add(-30 * time.Minute) // limit ichida
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive,
		IsRecordingEnabled: true, StartedAt: &started, UpdatedAt: started,
	}))
	// Xonada UNMUTED mikrofonli ishtirokchi — media haqiqatan chiqarilyapti.
	lk.Participants = []entity.RoomParticipant{{Identity: "student1", Active: true, AudioMuted: false, VideoMuted: true}}
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc := room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), rec, hands, testutil.NewFakeBlocklistRepo(), testutil.NewFakeBanRepo())

	// Dars yakunlanmaydi (limit ichida, xonada odam bor), lekin yozuv boshlanadi.
	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))
	select {
	case id := <-rec.ensured:
		require.Equal(t, testLessonID, id)
	case <-time.After(time.Second):
		t.Fatal("yo'qolgan webhook uchun yozuv reconciliation ishlamadi")
	}
}

// Reconciliation FAQAT haqiqiy media bo'lsa: hammasi muted bo'lsa egressni
// bo'sh ishga tushirmaymiz ("5 daqiqalik tuzoq").
func TestSweepAutoEnd_NoReconcileWhenAllMuted(t *testing.T) {
	ctx := context.Background()
	rec := newFakeRecorder()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	started := time.Now().UTC().Add(-30 * time.Minute)
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive,
		IsRecordingEnabled: true, StartedAt: &started, UpdatedAt: started,
	}))
	lk.Participants = []entity.RoomParticipant{{Identity: "student1", Active: true, AudioMuted: true, VideoMuted: true}}
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc := room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), rec, hands, testutil.NewFakeBlocklistRepo(), testutil.NewFakeBanRepo())

	require.Equal(t, 0, uc.SweepAutoEnd(ctx, 4*time.Hour, 20*time.Minute))
	select {
	case <-rec.ensured:
		t.Fatal("hamma muted bo'lsa yozuv boshlanmasligi kerak")
	case <-time.After(100 * time.Millisecond):
		// kutilgan: reconciliation ishlamadi
	}
}

// LiveKit o'chirilgan bo'lsa (dev muhiti) bo'sh xona qoidasi QO'LLANMAYDI —
// aks holda har jonli dars "bo'sh" deb yakunlanardi.
func TestSweepAutoEnd_NoLiveKitSkipsEmptyRule(t *testing.T) {
	uc, lrepo, lk, _ := autoEndSetup(t, time.Hour)
	lk.IsEnabled = false

	require.Equal(t, 0, uc.SweepAutoEnd(context.Background(), 4*time.Hour, time.Nanosecond))
	require.Equal(t, entity.LessonStatusLive, lessonStatus(t, lrepo))
}

// Webhook signali: `NoteRoomEmpty` grace hisobini xona bo'shagan LAHZADAN
// boshlaydi, `NoteRoomOccupied` esa bekor qiladi.
func TestNoteRoomOccupancy(t *testing.T) {
	ctx := context.Background()
	uc, _, _, cache := autoEndSetup(t, time.Hour)
	rn := shared.RoomName(testLessonID)

	uc.NoteRoomEmpty(ctx, rn)
	v, _ := cache.Get(ctx, "room:empty:"+testLessonID)
	require.NotEmpty(t, v)

	uc.NoteRoomOccupied(ctx, rn)
	v, _ = cache.Get(ctx, "room:empty:"+testLessonID)
	require.Empty(t, v)

	// Begona xona nomi jimgina e'tiborsiz qoldiriladi (panic/yozuv yo'q).
	uc.NoteRoomEmpty(ctx, "some-other-room")
	v, _ = cache.Get(ctx, "room:empty:"+testLessonID)
	require.Empty(t, v)
}
