package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	"github.com/zoom/darsly/internal/testutil"
)

// B-4 [MEDIUM] — cross-user IDOR salbiy holati.
//
// MarkRead `WHERE id=$1 AND user_id=$2` ga tayanadi. Mavjud UnreadFlow testi
// faqat ijobiy yo'lni qamraydi. Bu test ushlaydigan xato: agar `AND user_id`
// sharti tushib qolsa, foydalanuvchi A begona (B ning) bildirishnomasini
// «o'qildi» qilib qo'ya olardi (IDOR).
func TestNotificationRepo_MarkRead_CrossUserIDOR(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	notifs := pgRepo.NewNotificationRepo(pg)

	userA := makeUser(t, users, "notif_a@darsly.uz")
	userB := makeUser(t, users, "notif_b@darsly.uz")

	// B ga tegishli bildirishnoma.
	bNotif := &entity.Notification{
		ID: uuid.NewString(), UserID: userB.ID, Type: "system",
		Title: "B uchun", Body: "maxfiy", CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, notifs.Create(ctx(), bNotif))

	// A ning O'ZIDA hech qanday o'qilmagan bildirishnoma yo'q (nazorat).
	aCnt, err := notifs.UnreadCount(ctx(), userA.ID)
	require.NoError(t, err)
	require.Equal(t, 0, aCnt)

	// A B ning bildirishnomasini «o'qildi» qilishga urinadi.
	// Xato qaytmaydi (Exec 0 qatorga ta'sir qiladi), lekin holat O'ZGARMASLIGI shart.
	require.NoError(t, notifs.MarkRead(ctx(), bNotif.ID, userA.ID))

	// B ning bildirishnomasi hamon o'qilmagan bo'lib qolishi kerak.
	bCnt, err := notifs.UnreadCount(ctx(), userB.ID)
	require.NoError(t, err)
	require.Equal(t, 1, bCnt, "A begona (B ning) bildirishnomasini o'qildi qila olmasligi kerak (IDOR)")

	// Nazorat: haqiqiy egasi (B) o'zinikini o'qildi qila oladi.
	require.NoError(t, notifs.MarkRead(ctx(), bNotif.ID, userB.ID))
	bCnt, err = notifs.UnreadCount(ctx(), userB.ID)
	require.NoError(t, err)
	require.Equal(t, 0, bCnt, "egasi o'zinikini o'qildi qila olishi kerak")
}
