package notification_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/notification"
)

func newNotifUC() notification.UseCase {
	return notification.New(testutil.NewFakeNotifRepo(), ws.NewHub(testutil.NewLogger()), testutil.NewLogger())
}

func TestNotifyAndRead(t *testing.T) {
	uc := newNotifUC()
	ctx := context.Background()

	n, err := uc.Notify(ctx, "user1", entity.NotificationTypeLessonReminder, "Eslatma", "Dars boshlanadi", nil)
	require.NoError(t, err)
	require.Equal(t, "user1", n.UserID)

	count, err := uc.UnreadCount(ctx, "user1")
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, uc.MarkRead(ctx, "user1", n.ID))
	count, _ = uc.UnreadCount(ctx, "user1")
	require.Equal(t, 0, count, "o'qilgandan keyin unread 0")
}

func TestMarkAllRead(t *testing.T) {
	uc := newNotifUC()
	ctx := context.Background()
	for range 3 {
		_, _ = uc.Notify(ctx, "u", entity.NotificationTypeSystem, "T", "B", nil)
	}
	require.NoError(t, uc.MarkAllRead(ctx, "u"))
	count, _ := uc.UnreadCount(ctx, "u")
	require.Equal(t, 0, count)
}

func TestNotify_ScopedPerUser(t *testing.T) {
	uc := newNotifUC()
	ctx := context.Background()
	_, _ = uc.Notify(ctx, "a", entity.NotificationTypeSystem, "T", "B", nil)
	_, _ = uc.Notify(ctx, "b", entity.NotificationTypeSystem, "T", "B", nil)

	ca, _ := uc.UnreadCount(ctx, "a")
	require.Equal(t, 1, ca, "foydalanuvchi faqat o'z bildirishnomasini ko'radi")
}
