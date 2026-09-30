package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/testutil"
)

// flakyNotif — birinchi failN chaqiruv yiqiladi, keyin muvaffaqiyat.
type flakyNotif struct {
	failN int
	calls int
}

func (f *flakyNotif) Notify(_ context.Context, userID, _, _, _ string, _ *string) (*entity.Notification, error) {
	f.calls++
	if f.calls <= f.failN {
		return nil, errors.New("db down")
	}
	return &entity.Notification{UserID: userID}, nil
}
func (f *flakyNotif) List(context.Context, string, *entity.NotificationFilter) ([]*entity.Notification, int, error) {
	return nil, 0, nil
}
func (f *flakyNotif) UnreadCount(context.Context, string) (int, error) { return 0, nil }
func (f *flakyNotif) MarkRead(context.Context, string, string) error   { return nil }
func (f *flakyNotif) MarkAllRead(context.Context, string) error        { return nil }

func reminderFixture(t *testing.T, failN int) (*ReminderWorker, *flakyNotif) {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	at := time.Now().UTC().Add(5 * time.Minute)
	if err := lrepo.Create(context.Background(), &entity.Lesson{
		ID: "l1", MentorID: "m1", Title: "Dars", Status: entity.LessonStatusScheduled, ScheduledAt: &at,
	}); err != nil {
		t.Fatal(err)
	}
	n := &flakyNotif{failN: failN}
	w := NewReminderWorker(lrepo, n, testutil.NewLogger())
	w.retryBackoff = time.Millisecond
	return w, n
}

// Vaqtinchalik Notify xatosi eslatmani yo'qotmasligi kerak (claim allaqachon o'rnatilgan).
func TestReminder_TransientNotifyFailureRetried(t *testing.T) {
	w, n := reminderFixture(t, 2)
	w.tick(context.Background(), 10*time.Minute)
	if n.calls != 3 {
		t.Fatalf("3-urinishda yetkazilishi kerak, calls=%d", n.calls)
	}
	// Keyingi tick dublikat yubormaydi (claim o'rnatilgan).
	w.tick(context.Background(), 10*time.Minute)
	if n.calls != 3 {
		t.Fatalf("dublikat eslatma: calls=%d", n.calls)
	}
}

// Doimiy xatoda urinishlar cheklangan.
func TestReminder_PermanentFailureBounded(t *testing.T) {
	w, n := reminderFixture(t, 100)
	w.tick(context.Background(), 10*time.Minute)
	if n.calls != notifyAttempts {
		t.Fatalf("calls=%d, kutilgan %d", n.calls, notifyAttempts)
	}
}
