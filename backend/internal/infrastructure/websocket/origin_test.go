package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Rad etilgan upgrade javobi API bilan bir xil {code,message} JSON bo'lishi kerak
// (gorilla default'i text/plain qaytaradi).
func TestUpgradeError_JSONFormat(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("FRONTEND_BASE_URL", "https://app.darsly.uz")
	t.Setenv("WS_ALLOWED_ORIGINS", "")
	t.Setenv("WS_ALLOW_EMPTY_ORIGIN", "")

	r := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
	r.Header.Set("Origin", "https://evil.example.com")
	r.Header.Set("Connection", "Upgrade")
	r.Header.Set("Upgrade", "websocket")
	r.Header.Set("Sec-WebSocket-Version", "13")
	r.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	w := httptest.NewRecorder()
	up := newUpgrader()
	if _, err := up.Upgrade(w, r, nil); err == nil {
		t.Fatal("noto'g'ri Origin bilan upgrade muvaffaqiyatli bo'lmasligi kerak")
	}

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, kutilgan 403", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, kutilgan application/json", ct)
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("javob JSON emas: %v (%s)", err, w.Body.String())
	}
	if body.Code != "FORBIDDEN" || body.Message == "" {
		t.Fatalf("kutilmagan javob: %+v", body)
	}
	// Allowlist tarkibi javobda sizib chiqmasin.
	if strings.Contains(w.Body.String(), "darsly.uz") {
		t.Fatalf("javobda allowlist sizib chiqdi: %s", w.Body.String())
	}
}

// CheckOrigin jadval testi — brauzer (Origin bor) va native mobil (Origin yo'q) holatlari.
func TestCheckOrigin(t *testing.T) {
	const web = "https://app.194.163.139.242.sslip.io"

	cases := []struct {
		name string
		env  map[string]string
		// origin: "" → header umuman qo'yilmaydi (native mobil klient)
		origin string
		want   bool
	}{
		// --- orqaga moslik: faqat FRONTEND_BASE_URL sozlangan ---
		{
			name:   "prod: to'g'ri web origin ruxsat",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: web,
			want:   true,
		},
		{
			name:   "prod: noto'g'ri origin rad",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: "https://evil.example.com",
			want:   false,
		},
		{
			name:   "prod: oxirgi slash farq qilmaydi",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web + "/"},
			origin: web,
			want:   true,
		},
		{
			name:   "prod: harf registri farq qilmaydi",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: "HTTPS://APP.194.163.139.242.SSLIP.IO",
			want:   true,
		},

		// --- BE-1: native mobil klient Origin yubormaydi ---
		{
			name:   "prod: bo'sh Origin (mobil) ruxsat",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: "",
			want:   true,
		},
		{
			name: "prod: WS_ALLOW_EMPTY_ORIGIN=false → bo'sh Origin rad",
			env: map[string]string{
				"APP_ENV": "production", "FRONTEND_BASE_URL": web,
				"WS_ALLOW_EMPTY_ORIGIN": "false",
			},
			origin: "",
			want:   false,
		},
		{
			name: "prod: WS_ALLOW_EMPTY_ORIGIN=false web origin'ga tegmaydi",
			env: map[string]string{
				"APP_ENV": "production", "FRONTEND_BASE_URL": web,
				"WS_ALLOW_EMPTY_ORIGIN": "false",
			},
			origin: web,
			want:   true,
		},
		{
			name: "prod: WS_ALLOW_EMPTY_ORIGIN noto'g'ri qiymat → default true",
			env: map[string]string{
				"APP_ENV": "production", "FRONTEND_BASE_URL": web,
				"WS_ALLOW_EMPTY_ORIGIN": "hmm",
			},
			origin: "",
			want:   true,
		},

		// --- "null" Origin: sandbox iframe / file:// / cross-origin redirect ---
		// Bu HAQIQIY Origin qiymati (bo'sh emas) → allowlist bilan tekshiriladi.
		{
			name:   "prod: Origin: null → rad (allowlist bor)",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: "null",
			want:   false,
		},
		{
			name: "prod: Origin: null WS_ALLOW_EMPTY_ORIGIN=true bo'lsa ham rad",
			env: map[string]string{
				"APP_ENV": "production", "FRONTEND_BASE_URL": web,
				"WS_ALLOW_EMPTY_ORIGIN": "true",
			},
			origin: "null",
			want:   false,
		},
		{
			name:   "dev: Origin: null allowlist bo'sh → ruxsat (avvalgi xulq)",
			env:    map[string]string{"APP_ENV": "development"},
			origin: "null",
			want:   true,
		},
		// Faqat-bo'shliqli Origin normalizatsiyadan keyin "bo'sh" hisoblanadi.
		{
			name:   "prod: faqat-bo'shliqli Origin → bo'sh kabi (ruxsat)",
			env:    map[string]string{"APP_ENV": "production", "FRONTEND_BASE_URL": web},
			origin: "   ",
			want:   true,
		},
		{
			name: "prod: faqat-bo'shliqli Origin + ALLOW_EMPTY=false → rad",
			env: map[string]string{
				"APP_ENV": "production", "FRONTEND_BASE_URL": web,
				"WS_ALLOW_EMPTY_ORIGIN": "false",
			},
			origin: "   ",
			want:   false,
		},

		// --- ko'p-origin ro'yxati ---
		{
			name: "prod: ko'p-origin ro'yxati — birinchisi",
			env: map[string]string{
				"APP_ENV":            "production",
				"WS_ALLOWED_ORIGINS": "https://app.darsly.uz, https://admin.darsly.uz",
			},
			origin: "https://app.darsly.uz",
			want:   true,
		},
		{
			name: "prod: ko'p-origin ro'yxati — ikkinchisi (bo'shliq bilan)",
			env: map[string]string{
				"APP_ENV":            "production",
				"WS_ALLOWED_ORIGINS": "https://app.darsly.uz, https://admin.darsly.uz",
			},
			origin: "https://admin.darsly.uz",
			want:   true,
		},
		{
			name: "prod: ko'p-origin ro'yxatida yo'q → rad",
			env: map[string]string{
				"APP_ENV":            "production",
				"WS_ALLOWED_ORIGINS": "https://app.darsly.uz, https://admin.darsly.uz",
			},
			origin: "https://evil.example.com",
			want:   false,
		},
		{
			name: "WS_ALLOWED_ORIGINS FRONTEND_BASE_URL'dan ustun",
			env: map[string]string{
				"APP_ENV":            "production",
				"FRONTEND_BASE_URL":  "https://old.darsly.uz",
				"WS_ALLOWED_ORIGINS": "https://new.darsly.uz",
			},
			origin: "https://old.darsly.uz",
			want:   false,
		},

		// --- production / dev farqi (allowlist sozlanmagan) ---
		{
			name:   "prod: allowlist bo'sh → har qanday Origin rad",
			env:    map[string]string{"APP_ENV": "production"},
			origin: "https://app.darsly.uz",
			want:   false,
		},
		{
			name:   "dev: allowlist bo'sh → har qanday Origin ruxsat",
			env:    map[string]string{"APP_ENV": "development"},
			origin: "http://localhost:5173",
			want:   true,
		},
		{
			name:   "dev: allowlist bor → mos kelmagan Origin rad",
			env:    map[string]string{"APP_ENV": "development", "FRONTEND_BASE_URL": "http://localhost:3000"},
			origin: "http://localhost:5173",
			want:   false,
		},
		{
			name:   "dev: bo'sh Origin ruxsat",
			env:    map[string]string{"APP_ENV": "development", "FRONTEND_BASE_URL": "http://localhost:3000"},
			origin: "",
			want:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Barcha aloqador env'lar har testda aniq holatga keltiriladi.
			for _, k := range []string{"APP_ENV", "FRONTEND_BASE_URL", "WS_ALLOWED_ORIGINS", "WS_ALLOW_EMPTY_ORIGIN"} {
				t.Setenv(k, "")
			}
			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			up := newUpgrader()
			r := httptest.NewRequest(http.MethodGet, "/api/v1/ws", nil)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}

			if got := up.CheckOrigin(r); got != tc.want {
				t.Fatalf("CheckOrigin(origin=%q) = %v, kutilgan %v", tc.origin, got, tc.want)
			}
		})
	}
}
