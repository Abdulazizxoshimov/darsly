package apitests_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
)

// insertNotification foydalanuvchiga bildirishnoma qo'shadi va ID sini qaytaradi.
func insertNotification(t *testing.T, pg *pgpkg.Postgres, uid, title string) string {
	t.Helper()
	var id string
	err := pg.DB.QueryRow(context.Background(),
		`INSERT INTO notifications (user_id, type, title, body)
		 VALUES ($1, 'system', $2, 'matn') RETURNING id`, uid, title).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestNotification_HappyPath: list, unread-count, mark-read, read-all, unread filtri.
func TestNotification_HappyPath(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	tok := registerStudent(t, cl, "Notif U", "notif_u@darsly.uz")
	uid := userID(t, pg, "notif_u@darsly.uz")

	n1 := insertNotification(t, pg, uid, "Birinchi")
	insertNotification(t, pg, uid, "Ikkinchi")
	insertNotification(t, pg, uid, "Uchinchi")

	// Ro'yxat → total 3, data 3.
	code, body := cl.get("/api/v1/notifications", tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 3, jsonInt(body, "total"), "total 3: %s", body)
	require.Equal(t, 3, jsonLen(body, "data"))

	// O'qilmagan soni → 3.
	code, body = cl.get("/api/v1/notifications/unread-count", tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 3, jsonInt(body, "data", "count"), "unread 3: %s", body)

	// Bittasini o'qilgan deb belgilash → 204.
	code, _ = cl.post("/api/v1/notifications/"+n1+"/read", tok, nil)
	require.Equal(t, http.StatusNoContent, code)

	// O'qilmagan soni → 2.
	code, body = cl.get("/api/v1/notifications/unread-count", tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 2, jsonInt(body, "data", "count"))

	// unread=true filtri → 2 ta.
	code, body = cl.get("/api/v1/notifications?unread=true", tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 2, jsonInt(body, "total"), "unread filtr total 2: %s", body)
	require.Equal(t, 2, jsonLen(body, "data"))

	// Hammasini o'qilgan → 204.
	code, _ = cl.post("/api/v1/notifications/read-all", tok, nil)
	require.Equal(t, http.StatusNoContent, code)

	// O'qilmagan soni → 0.
	code, body = cl.get("/api/v1/notifications/unread-count", tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 0, jsonInt(body, "data", "count"))
}

// TestNotification_CrossUserIsolation: foydalanuvchi B, A ning bildirishnomasini
// o'qilgan qila OLMASLIGI kerak. Repo MarkRead so'rovni user_id bo'yicha scope qiladi
// (WHERE id=$1 AND user_id=$2), shuning uchun begona ID uchun bu xavfsiz no-op —
// 204 qaytaradi, ammo A ning holatini o'zgartirmaydi (403/404 mexanizmi yo'q, lekin
// hech qanday cross-user o'zgarish ham yo'q). Muhim invariant: A ning bildirishnomasi
// O'QILMAGAN qolishi shart.
func TestNotification_CrossUserIsolation(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	tokA := registerStudent(t, cl, "User A", "notif_a@darsly.uz")
	tokB := registerStudent(t, cl, "User B", "notif_b@darsly.uz")
	uidA := userID(t, pg, "notif_a@darsly.uz")

	nA := insertNotification(t, pg, uidA, "A ning maxfiy xabari")

	// B, A ning bildirishnomasini mark-read qiladi (scoped no-op → 204 kutiladi).
	codeB, _ := cl.post("/api/v1/notifications/"+nA+"/read", tokB, nil)
	require.Equal(t, http.StatusNoContent, codeB,
		"mark-read user_id bo'yicha scope qilingan — begona ID uchun xavfsiz no-op (204)")

	// XAVFSIZLIK INVARIANTI: A ning bildirishnomasi O'QILMAGAN qolishi shart
	// (B hech qanday holatda A ning bildirishnomasini o'qilgan qila olmaydi).
	code, body := cl.get("/api/v1/notifications/unread-count", tokA)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 1, jsonInt(body, "data", "count"),
		"!!! A ning o'qilmagan soni 1 bo'lib qolishi kerak — B ta'sir qila olmaydi (cross-user leak bo'lardi)")
}
