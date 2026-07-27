package livekit

import (
	"testing"

	"github.com/zoom/darsly/internal/pkg/config"
)

// BE-8: yozuv layout'i. Default "speaker" bo'lishi va noma'lum qiymat
// default'ga qaytishi kerak (noto'g'ri env tufayli yozuv buzilmasin).
func TestNormalizeLayout(t *testing.T) {
	cases := []struct{ in, want string }{
		// Yaroqli qiymatlar (livekit/egress default shabloni qabul qiladi).
		{"speaker", "speaker"},
		{"speaker-light", "speaker-light"},
		{"single-speaker", "single-speaker"},
		{"single-speaker-light", "single-speaker-light"},
		{"grid", "grid"},
		{"grid-light", "grid-light"},
		// Normalizatsiya.
		{"SPEAKER", "speaker"},
		{"  grid  ", "grid"},
		// Yaroqsiz → default.
		{"", DefaultEgressLayout},
		{"gallery", DefaultEgressLayout},
		{"speaker-dark", DefaultEgressLayout}, // shablon faqat "-light" ni biladi
		{"../../etc/passwd", DefaultEgressLayout},
	}
	for _, tc := range cases {
		if got := normalizeLayout(tc.in); got != tc.want {
			t.Errorf("normalizeLayout(%q) = %q, kutilgan %q", tc.in, got, tc.want)
		}
	}

	if DefaultEgressLayout != "speaker" {
		t.Errorf("default layout %q — dars yozuvida ekran/ustoz asosiy oynada bo'lishi kerak (speaker)", DefaultEgressLayout)
	}
}

// Konfiguratsiyadan kelgan layout klientga o'rnatilishi kerak.
func TestNewClient_LayoutFromConfig(t *testing.T) {
	base := config.LiveKitConfig{Host: "ws://localhost:7880", APIKey: "k", APISecret: "s"}

	base.EgressLayout = ""
	if c := New(base); c.layout != DefaultEgressLayout {
		t.Errorf("bo'sh env → %q, kutilgan %q", c.layout, DefaultEgressLayout)
	}

	base.EgressLayout = "grid"
	if c := New(base); c.layout != "grid" {
		t.Errorf("env'dan layout o'rnatilmadi: %q", c.layout)
	}

	base.EgressLayout = "chaqmoq"
	if c := New(base); c.layout != DefaultEgressLayout {
		t.Errorf("yaroqsiz env → %q, kutilgan %q", c.layout, DefaultEgressLayout)
	}
}
