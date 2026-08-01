package apitests_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Bu fayl №20 (dars arxivi) va №21 (chat transkripti) yo'llarini HTTP darajasida
// tekshiradi: marshrut daraxti (`/chat/transcript` `/chat/:messageID` bilan
// to'qnashmasligi), RBAC siyosati, fayl sarlavhalari va javob shakli.
// Formatlash mantig'i alohida birlik testlarida (`usecase/chat/transcript_test.go`).

// getRaw — `httpClient.get` sarlavhalarni qaytarmaydi, transkript testi esa
// aynan `Content-Disposition` ni tekshiradi (fayl sifatida yuklanishi shart).
func getRaw(t *testing.T, base, path, token string) (int, http.Header, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, body
}

// №21 — transkript TXT va HTML sifatida yuklab olinadi, noto'g'ri format 400.
func TestAPI_ChatTranscript(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "transcript@darsly.uz")

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "Salom, boshladik"})
	require.Equal(t, http.StatusCreated, code, string(body))

	// TXT (default format).
	code, hdr, raw := getRaw(t, srv.URL, "/api/v1/lessons/"+lessonID+"/chat/transcript", tok)
	require.Equal(t, http.StatusOK, code, string(raw))
	require.Contains(t, hdr.Get("Content-Type"), "text/plain")
	require.Contains(t, hdr.Get("Content-Disposition"), "attachment; filename=")
	require.Contains(t, hdr.Get("Content-Disposition"), ".txt")
	require.Contains(t, string(raw), "Dars: Matematika")
	require.Contains(t, string(raw), "Siz: Salom, boshladik")

	// HTML — bir faylli, offline.
	code, hdr, raw = getRaw(t, srv.URL, "/api/v1/lessons/"+lessonID+"/chat/transcript?format=html", tok)
	require.Equal(t, http.StatusOK, code, string(raw))
	require.Contains(t, hdr.Get("Content-Type"), "text/html")
	require.Contains(t, hdr.Get("Content-Disposition"), ".html")
	require.Contains(t, string(raw), "<!DOCTYPE html>")

	// Noma'lum format — 400, hech qanday fayl qaytmaydi.
	code, body = cl.get("/api/v1/lessons/"+lessonID+"/chat/transcript?format=pdf", tok)
	require.Equal(t, http.StatusBadRequest, code, string(body))
}

// `/chat/transcript` `/chat` marshrutini SINDIRMASLIGI kerak (gin daraxtida
// statik segment va parametr bir pozitsiyada — shuning uchun oshkora test).
func TestAPI_ChatTranscript_ChatniSindirmaydi(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "transcript2@darsly.uz")

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "test"})
	require.Equal(t, http.StatusCreated, code, string(body))

	code, body = cl.get("/api/v1/lessons/"+lessonID+"/chat", tok)
	require.Equal(t, http.StatusOK, code, string(body))
	require.Contains(t, string(body), "test")
}

// Egalik: begona mentor na transkript, na arxiv ola olmaydi.
func TestAPI_Archive_Egalik(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	_, lessonID := mentorSession(t, cl, pg, "owner@darsly.uz")
	other, _ := mentorSession(t, cl, pg, "intruder@darsly.uz")

	code, body := cl.get("/api/v1/lessons/"+lessonID+"/archive", other)
	require.Equal(t, http.StatusForbidden, code, string(body))

	code, body = cl.get("/api/v1/lessons/"+lessonID+"/chat/transcript", other)
	require.Equal(t, http.StatusForbidden, code, string(body))
}

// Auth'siz — 401 (marshrut himoyalangan guruhda).
func TestAPI_Archive_Auth(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	_, lessonID := mentorSession(t, cl, pg, "archauth@darsly.uz")

	code, _ := cl.get("/api/v1/lessons/"+lessonID+"/archive", "")
	require.Equal(t, http.StatusUnauthorized, code)
}

// №20 — arxiv javobining SHAKLI: yozuvsiz va chatsiz darsda ham to'liq va
// klient uchun xavfsiz (`null` emas, `[]`).
func TestAPI_Archive_Shakl(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "archive@darsly.uz")

	// Bo'sh dars.
	code, body := cl.get("/api/v1/lessons/"+lessonID+"/archive", tok)
	require.Equal(t, http.StatusOK, code, string(body))

	var resp struct {
		Data struct {
			Lesson    map[string]any   `json:"lesson"`
			Recording map[string]any   `json:"recording"`
			Chat      []map[string]any `json:"chat"`
			Materials []map[string]any `json:"materials"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &resp))
	require.Equal(t, lessonID, resp.Data.Lesson["id"])
	require.Nil(t, resp.Data.Recording, "yozuvsiz darsda recording null bo'lishi kerak")
	require.NotNil(t, resp.Data.Chat, "chat `null` emas `[]` bo'lishi kerak")
	require.Empty(t, resp.Data.Chat)
	require.NotNil(t, resp.Data.Materials)

	// Chat xabari qo'shilgach arxivda `offset_sec` bilan chiqadi.
	code, cbody := cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "arxivga tushsin"})
	require.Equal(t, http.StatusCreated, code, string(cbody))

	code, body = cl.get("/api/v1/lessons/"+lessonID+"/archive", tok)
	require.Equal(t, http.StatusOK, code, string(body))
	require.NoError(t, json.Unmarshal(body, &resp))
	require.Len(t, resp.Data.Chat, 1)
	require.Equal(t, "arxivga tushsin", resp.Data.Chat[0]["body"])
	// Maydon MAVJUD bo'lishi shart (klient unga tayanadi), qiymati esa
	// boshlanmagan darsda 0.
	_, ok := resp.Data.Chat[0]["offset_sec"]
	require.True(t, ok, "offset_sec maydoni bo'lishi kerak")
	require.Contains(t, resp.Data.Chat[0], "to_identity")
	require.Contains(t, resp.Data.Chat[0], "file")
}

// Moderatsiya qilingan xabar arxivga ham, transkriptga ham tushmaydi —
// aks holda "o'chirdim" degan amal ikki yo'l orqali bekor bo'lardi.
func TestAPI_Archive_OchirilganXabar(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}
	tok, lessonID := mentorSession(t, cl, pg, "archdel@darsly.uz")

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/chat", tok, map[string]any{"body": "ochiriladi"})
	require.Equal(t, http.StatusCreated, code, string(body))
	msgID := gjson(body, "data", "id")

	code, body = cl.do(http.MethodDelete, "/api/v1/lessons/"+lessonID+"/chat/"+msgID, tok, nil)
	require.Equal(t, http.StatusNoContent, code, string(body))

	code, body = cl.get("/api/v1/lessons/"+lessonID+"/archive", tok)
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, string(body), "ochiriladi")

	_, _, raw := getRaw(t, srv.URL, "/api/v1/lessons/"+lessonID+"/chat/transcript", tok)
	require.NotContains(t, string(raw), "ochiriladi")
}
