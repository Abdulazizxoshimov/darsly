package postgres_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
)

func ctx() context.Context { return context.Background() }

func makeUser(t *testing.T, users interface {
	Create(context.Context, *entity.User) error
}, email string) *entity.User {
	t.Helper()
	u := &entity.User{
		ID: uuid.NewString(), Email: email, PasswordHash: "x", FullName: "T",
		Color: "#fff", Role: "mentor", Timezone: "UTC", Language: "uz", IsActive: true,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, users.Create(ctx(), u))
	return u
}

func TestUserRepo_UniqueEmailAndDeleteHard(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)

	u := makeUser(t, users, "dup@darsly.uz")

	// Takroriy email → conflict.
	dup := *u
	dup.ID = uuid.NewString()
	err := users.Create(ctx(), &dup)
	require.True(t, apperr.IsConflict(err), "takroriy email conflict qaytarishi kerak")

	// DeleteHard email'ni bo'shatadi.
	require.NoError(t, users.DeleteHard(ctx(), u.ID))
	u2 := *u
	u2.ID = uuid.NewString()
	require.NoError(t, users.Create(ctx(), &u2), "hard-delete'dan keyin email qayta ishlatilishi kerak")
}

func TestLessonRepo_ClaimReminder_Atomic(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	mentor := makeUser(t, users, "m@darsly.uz")

	l := &entity.Lesson{
		ID: uuid.NewString(), MentorID: mentor.ID, Title: "X", DurationMin: 60,
		JoinSlug: "clm-slug-001", Status: entity.LessonStatusScheduled,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, lessons.Create(ctx(), l))

	first, err := lessons.ClaimReminder(ctx(), l.ID)
	require.NoError(t, err)
	require.True(t, first, "birinchi claim muvaffaqiyatli")

	second, err := lessons.ClaimReminder(ctx(), l.ID)
	require.NoError(t, err)
	require.False(t, second, "ikkinchi claim rad etilishi kerak (dublikat eslatma yo'q)")
}

// TransitionFromPending haqiqiy konkurentlik ostida atomik ekanini isbotlaydi:
// 20 goroutine bir vaqtda admit qilsa — FAQAT bittasi yutishi kerak (TOCTOU yo'q).
func TestWaitingRoomRepo_TransitionAtomicUnderConcurrency(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	waiting := pgRepo.NewWaitingRoomRepo(pg)
	mentor := makeUser(t, users, "w@darsly.uz")

	l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: "X", DurationMin: 60, JoinSlug: "wr-slug-001", Status: entity.LessonStatusLive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, lessons.Create(ctx(), l))

	req := &entity.WaitingRoomRequest{ID: uuid.NewString(), LessonID: l.ID, RequesterName: "G", GuestIdentity: "guest_x", Status: entity.WaitingStatusPending, CreatedAt: time.Now().UTC()}
	require.NoError(t, waiting.Create(ctx(), req))

	const n = 20
	var wins int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := waiting.TransitionFromPending(ctx(), req.ID, entity.WaitingStatusAdmitted, time.Now().UTC())
			require.NoError(t, err)
			if ok {
				atomic.AddInt64(&wins, 1)
			}
		}()
	}
	close(start)
	wg.Wait()

	require.Equal(t, int64(1), wins, "atomik CAS: faqat bitta admit yutishi kerak (TOCTOU himoyasi)")
}

func TestRecordingRepo_Lifecycle(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	recs := pgRepo.NewRecordingRepo(pg)
	mentor := makeUser(t, users, "r@darsly.uz")
	l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: "X", DurationMin: 60, JoinSlug: "rec-slug-001", Status: entity.LessonStatusLive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, lessons.Create(ctx(), l))

	rec := &entity.Recording{ID: uuid.NewString(), LessonID: l.ID, EgressID: "EG-int-1", ObjectKey: "k.mp4", Status: entity.RecordingStatusProcessing, StartedAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}
	require.NoError(t, recs.Create(ctx(), rec))

	require.NoError(t, recs.MarkReady(ctx(), "EG-int-1", "final.mp4", 42, 1027, time.Now().UTC()))
	got, err := recs.GetByEgressID(ctx(), "EG-int-1")
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, got.Status)
	require.Equal(t, "final.mp4", got.ObjectKey)
	require.Equal(t, 42, got.DurationSec)
	require.Equal(t, int64(1027), got.SizeBytes)
}

func TestNotificationRepo_UnreadFlow(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	notifs := pgRepo.NewNotificationRepo(pg)
	u := makeUser(t, users, "n@darsly.uz")

	for range 3 {
		require.NoError(t, notifs.Create(ctx(), &entity.Notification{
			ID: uuid.NewString(), UserID: u.ID, Type: "system", Title: "T", Body: "B", CreatedAt: time.Now().UTC(),
		}))
	}
	cnt, err := notifs.UnreadCount(ctx(), u.ID)
	require.NoError(t, err)
	require.Equal(t, 3, cnt)

	items, _, err := notifs.ListByUser(ctx(), u.ID, &entity.NotificationFilter{})
	require.NoError(t, err)
	require.NoError(t, notifs.MarkRead(ctx(), items[0].ID, u.ID))
	cnt, _ = notifs.UnreadCount(ctx(), u.ID)
	require.Equal(t, 2, cnt)

	require.NoError(t, notifs.MarkAllRead(ctx(), u.ID))
	cnt, _ = notifs.UnreadCount(ctx(), u.ID)
	require.Equal(t, 0, cnt)
}

// TestChatRepo_DirectMessageVisibility — shaxsiy xabar ko'rinuvchanligi HAQIQIY SQL'da.
//
// Bu testni fake repo bilan almashtirib bo'lmaydi: xavf aynan SQL'da —
// `to_identity IS NULL` shartining to'g'ri generatsiya bo'lishi va OR
// tarmoqlarining to'g'ri qavslanishi. Xato bo'lsa ikki oqibatdan biri chiqadi:
// ommaviy xabarlar YO'QOLADI yoki begona shaxsiy xabar SIZIB CHIQADI.
func TestChatRepo_DirectMessageVisibility(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	chats := pgRepo.NewChatRepo(pg)

	mentor := makeUser(t, users, "chat_dm_mentor@darsly.uz")
	sched := time.Now().UTC().Add(time.Hour)
	l := &entity.Lesson{
		ID: uuid.NewString(), MentorID: mentor.ID, Title: "Chat DM", DurationMin: 60,
		ScheduledAt: &sched, Status: entity.LessonStatusLive,
		JoinSlug:  uuid.NewString()[:12],
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, lessons.Create(ctx(), l))

	add := func(sender, name, body string, to *string, offset time.Duration) {
		t.Helper()
		require.NoError(t, chats.Create(ctx(), &entity.ChatMessage{
			ID: uuid.NewString(), LessonID: l.ID, SenderIdentity: sender,
			SenderName: name, Body: body, ToIdentity: to,
			CreatedAt: time.Now().UTC().Add(offset),
		}))
	}
	ptr := func(s string) *string { return &s }

	add("mentor", "Ustoz", "hammaga", nil, -5*time.Minute)
	add("ali", "Ali", "ali→ustoz", ptr("mentor"), -4*time.Minute)
	add("mentor", "Ustoz", "ustoz→ali", ptr("ali"), -3*time.Minute)
	add("vali", "Vali", "vali→ustoz", ptr("mentor"), -2*time.Minute)

	bodies := func(identity string) []string {
		t.Helper()
		items, err := chats.ListByLesson(ctx(), l.ID, identity, nil, 50)
		require.NoError(t, err)
		out := make([]string, 0, len(items))
		for _, m := range items {
			out = append(out, m.Body)
		}
		return out
	}

	// Ali: ommaviy + o'zi ishtirok etgan ikki DM. Vali'ning DM'i KO'RINMAYDI.
	require.ElementsMatch(t, []string{"hammaga", "ali→ustoz", "ustoz→ali"}, bodies("ali"))

	// Vali: ommaviy + faqat o'ziniki.
	require.ElementsMatch(t, []string{"hammaga", "vali→ustoz"}, bodies("vali"))

	// Ustoz: ommaviy + unga/undan ketgan hamma DM.
	require.ElementsMatch(t, []string{"hammaga", "ali→ustoz", "ustoz→ali", "vali→ustoz"}, bodies("mentor"))

	// Xonaga kirmagan (identity'siz) — FAQAT ommaviy.
	require.ElementsMatch(t, []string{"hammaga"}, bodies(""))

	// Kursor-paginatsiya DM filtri bilan birga ishlashi kerak.
	items, err := chats.ListByLesson(ctx(), l.ID, "ali", nil, 2)
	require.NoError(t, err)
	require.Len(t, items, 2, "limit qo'llanishi kerak")
	require.Equal(t, "ustoz→ali", items[0].Body, "eng yangi birinchi")
}
