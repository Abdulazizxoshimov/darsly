package apitests_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
)

// Bu fayl №6 (chat moderatsiyasi), №7 (poll e'lon qilish) va №15 (fayl ulashish)
// yo'llarini HTTP darajasida tekshiradi: marshrut daraxti, RBAC siyosati va
// javob statuslari. Usecase mantig'i alohida birlik testlarida.

// mentorSession — mentor roli bilan tayyor sessiya va bitta dars qaytaradi.
func mentorSession(t *testing.T, cl *httpClient, pg *pgpkg.Postgres, email string) (token, lessonID string) {
	t.Helper()
	code, body := cl.post("/api/v1/auth/register", "",
		map[string]string{"full_name": "Dilnoza", "email": email, "password": "parol12345"})
	require.Equal(t, http.StatusCreated, code, string(body))

	_, err := pg.DB.Exec(context.Background(), "UPDATE users SET role='mentor' WHERE email=$1", email)
	require.NoError(t, err)

	code, body = cl.post("/api/v1/auth/login", "", map[string]string{"email": email, "password": "parol12345"})
	require.Equal(t, http.StatusOK, code, string(body))
	token = gjson(body, "data", "access_token")

	code, body = cl.post("/api/v1/lessons", token, map[string]any{"title": "Matematika"})
	require.Equal(t, http.StatusCreated, code, string(body))
	return token, gjson(body, "data", "id")
}

// №6 — chat xabarini o'chirish yo'li mavjud, RBAC ochiq va o'chirilgan xabar
// tarixga qaytmaydi.
func TestAPI_ChatDelete(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "chatdel@darsly.uz")

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "birinchi"})
	require.Equal(t, http.StatusCreated, code, string(body))
	msgID := gjson(body, "data", "id")
	require.NotEmpty(t, msgID)

	code, body = cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "ikkinchi"})
	require.Equal(t, http.StatusCreated, code, string(body))

	code, body = cl.do(http.MethodDelete, "/api/v1/lessons/"+lessonID+"/chat/"+msgID, tok, nil)
	require.Equal(t, http.StatusNoContent, code, string(body))

	// Tarixda faqat o'chirilmagani qoladi.
	code, body = cl.get("/api/v1/lessons/"+lessonID+"/chat", tok)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, string(body), "birinchi", "o'chirilgan xabar tarixda qolmasligi kerak")
	require.Contains(t, string(body), "ikkinchi")

	// Takroriy o'chirish — 404 (atomik `deleted_at IS NULL`).
	code, _ = cl.do(http.MethodDelete, "/api/v1/lessons/"+lessonID+"/chat/"+msgID, tok, nil)
	require.Equal(t, http.StatusNotFound, code)

	// Yaroqsiz ID — 404, 500 EMAS (Sentry shovqini bo'lmasin).
	code, _ = cl.do(http.MethodDelete, "/api/v1/lessons/"+lessonID+"/chat/abc", tok, nil)
	require.Equal(t, http.StatusNotFound, code)
}

// Begona mentor — 403. Bu tekshiruv HTTP darajasida ham kerak: RBAC siyosati
// endpointni ochadi, egalikni esa usecase tekshiradi.
func TestAPI_ChatDelete_BegonaMentor403(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	ownerTok, lessonID := mentorSession(t, cl, pg, "egasi@darsly.uz")
	intruderTok, _ := mentorSession(t, cl, pg, "begona@darsly.uz")

	_, body := cl.post("/api/v1/lessons/"+lessonID+"/chat", ownerTok, map[string]any{"body": "x"})
	msgID := gjson(body, "data", "id")

	code, _ := cl.do(http.MethodDelete, "/api/v1/lessons/"+lessonID+"/chat/"+msgID, intruderTok, nil)
	require.Equal(t, http.StatusForbidden, code)
}

// №7 — e'lon qilish yo'li va rejim mantig'i.
func TestAPI_PollPublish(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "poll@darsly.uz")

	// mentor_only — e'lon qilib bo'lmaydi (400).
	code, body := cl.post("/api/v1/lessons/"+lessonID+"/polls", tok,
		map[string]any{"question": "Tushunarlimi?", "options": []string{"Ha", "Yo'q"}})
	require.Equal(t, http.StatusCreated, code, string(body))
	require.Contains(t, string(body), `"results_visibility":"mentor_only"`, "default yopiq bo'lishi kerak")
	privateID := gjson(body, "data", "id")

	code, body = cl.post("/api/v1/lessons/"+lessonID+"/polls/"+privateID+"/publish", tok, nil)
	require.Equal(t, http.StatusBadRequest, code, string(body))

	// public — e'lon qilinadi (200) va natija qaytadi.
	code, body = cl.post("/api/v1/lessons/"+lessonID+"/polls", tok, map[string]any{
		"question": "Yoqdimi?", "options": []string{"Ha", "Yo'q"}, "results_visibility": "public",
	})
	require.Equal(t, http.StatusCreated, code, string(body))
	publicID := gjson(body, "data", "id")

	code, body = cl.post("/api/v1/lessons/"+lessonID+"/polls/"+publicID+"/publish", tok, nil)
	require.Equal(t, http.StatusOK, code, string(body))
	require.Contains(t, string(body), `"results_published_at"`)

	// Noto'g'ri rejim — validatsiya.
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/polls", tok, map[string]any{
		"question": "Q", "options": []string{"A", "B"}, "results_visibility": "hammaga",
	})
	require.Equal(t, http.StatusBadRequest, code)
}

// №15 — fayl yuklash yo'li: cheklovlar HTTP darajasida ham qo'llanadi.
//
// Muhit MinIO'siz (`minio.NewNop`), shuning uchun MUVAFFAQIYATLI yuklash bu
// yerda tekshirilmaydi (u birlik testlarida) — bu yerda RAD ETISH yo'llari
// tekshiriladi, ya'ni yaroqsiz fayl saqlagichgacha umuman bormaydi.
func TestAPI_ChatUploadRejects(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "upload@darsly.uz")
	path := "/api/v1/lessons/" + lessonID + "/chat/upload"

	// Ruxsatsiz tur → 400.
	code, body := cl.upload(path, tok, "virus.exe", []byte("MZ\x90\x00"), nil)
	require.Equal(t, http.StatusBadRequest, code, string(body))

	// Mazmun kengaytmaga mos kelmasa → 400.
	code, body = cl.upload(path, tok, "rasm.png", []byte("<html><script>x</script></html>"), nil)
	require.Equal(t, http.StatusBadRequest, code, string(body))

	// Faylsiz so'rov → 400 (500 emas).
	code, body = cl.upload(path, tok, "", nil, map[string]string{"body": "izoh"})
	require.Equal(t, http.StatusBadRequest, code, string(body))

	// Haddan uzun caption / `to` → 400 (validatsiyasiz 20 MB caption o'tardi).
	png := append([]byte("\x89PNG\r\n\x1a\n"), make([]byte, 32)...)
	code, body = cl.upload(path, tok, "a.png", png, map[string]string{"body": strings.Repeat("x", 2001)})
	require.Equal(t, http.StatusBadRequest, code, string(body))
	code, body = cl.upload(path, tok, "a.png", png, map[string]string{"to": strings.Repeat("y", 129)})
	require.Equal(t, http.StatusBadRequest, code, string(body))
}

// upload — multipart so'rov (fayl + qo'shimcha maydonlar). fileName bo'sh
// bo'lsa fayl umuman qo'shilmaydi.
func (c *httpClient) upload(path, token, fileName string, content []byte, fields map[string]string) (int, []byte) {
	c.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		require.NoError(c.t, w.WriteField(k, v))
	}
	if fileName != "" {
		fw, err := w.CreateFormFile("file", fileName)
		require.NoError(c.t, err)
		_, err = fw.Write(content)
		require.NoError(c.t, err)
	}
	require.NoError(c.t, w.Close())

	req, _ := http.NewRequest(http.MethodPost, c.base+path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}
