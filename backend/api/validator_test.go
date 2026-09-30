package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
)

// installValidator gin ShouldBindJSON'ni `validate` teglarini yuritadigan qiladi.
// Bu test o'sha ulanish haqiqatan ishlashini tekshiradi (avval umuman ishlamasdi).
func TestInstallValidator_EnforcesTags(t *testing.T) {
	gin.SetMode(gin.TestMode)
	installValidator()

	r := gin.New()
	r.POST("/t", func(c *gin.Context) {
		var req entity.RegisterReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	cases := []struct {
		name string
		body string
		want int
	}{
		{"qisqa parol (min=8)", `{"full_name":"Ali","email":"a@x.uz","password":"123"}`, http.StatusBadRequest},
		{"noto'g'ri email", `{"full_name":"Ali","email":"notanemail","password":"parol12345"}`, http.StatusBadRequest},
		{"bo'sh full_name", `{"full_name":"","email":"a@x.uz","password":"parol12345"}`, http.StatusBadRequest},
		{"to'g'ri", `{"full_name":"Ali","email":"a@x.uz","password":"parol12345"}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/t", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			require.Equal(t, tc.want, w.Code, tc.body)
		})
	}
}

// Binding/validatsiya xatolari Go ichki tuzilmasini OSHKOR QILMASLIGI kerak.
// Avval klientga quyidagilar chiqardi:
//
//	"json: cannot unmarshal number into Go struct field LoginReq.email of type string"
//	"Key: 'RegisterReq.password' Error:Field validation for 'password' failed on the 'min' tag"
func TestBindError_DoesNotLeakInternals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	installValidator()

	r := gin.New()
	r.POST("/register", func(c *gin.Context) {
		var req entity.RegisterReq
		if err := c.ShouldBindJSON(&req); err != nil {
			// Handler'lardagi haqiqiy naqsh (hs.BadRequest(c, err.Error())).
			c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Klient javobida HECH QACHON uchramasligi kerak bo'lgan bo'laklar.
	forbidden := []string{
		"RegisterReq", "LoginReq", "Req.", // Go struct nomlari
		"Error:Field validation", "Key: '", // validator ichki formati
		"tag", "'min'", "'max'", "'required'", // tag nomlari
		"Go struct field", "json: cannot unmarshal", // encoding/json ichki matni
		"reflect", "entity.",
	}

	cases := []struct {
		name string
		body string
		// javobda albatta bo'lishi kerak (ma'no yo'qolmaganini tekshiradi)
		wantContains string
	}{
		{"qisqa parol (validatsiya)", `{"full_name":"Ali","email":"a@x.uz","password":"123"}`, "password"},
		{"email turi noto'g'ri (raqam)", `{"full_name":"Ali","email":123,"password":"parol12345"}`, "email"},
		{"buzuq JSON", `{"email":`, "malformed JSON body"},
		{"bo'sh body", ``, "request body is required"},
		{"JSON sintaksis xatosi", `{"email": }`, "malformed JSON body"},
		{"noto'g'ri email format", `{"full_name":"Ali","email":"notanemail","password":"parol12345"}`, "email"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, http.StatusBadRequest, w.Code)
			body := w.Body.String()

			for _, bad := range forbidden {
				require.NotContains(t, body, bad, "ichki tafsilot oshkor bo'ldi (%q): %s", bad, body)
			}
			require.Contains(t, body, tc.wantContains, "foydalanuvchi uchun ma'no yo'qolmasligi kerak: %s", body)
			// Frontend api.jsx `code` bo'yicha matn tanlaydi — kod o'zgarmasligi shart.
			require.Contains(t, body, `"code":"BAD_REQUEST"`)
		})
	}
}

// avatar_url faqat http/https: `javascript:`/`data:` sxemalari rad etilishi kerak.
func TestHTTPURLValidator_RejectsDangerousSchemes(t *testing.T) {
	sv := &structValidator{}
	ok := []string{"https://cdn.darsly.uz/a.png", "http://localhost:9020/a.png"}
	bad := []string{"javascript:alert(1)", "data:text/html,<script>", "file:///etc/passwd", "ftp://x.uz/a", "not a url", "https://"}

	for _, u := range ok {
		u := u
		require.NoError(t, sv.ValidateStruct(&entity.UpdateUserReq{AvatarURL: &u}), u)
	}
	for _, u := range bad {
		u := u
		require.Error(t, sv.ValidateStruct(&entity.UpdateUserReq{AvatarURL: &u}), u)
	}
}
