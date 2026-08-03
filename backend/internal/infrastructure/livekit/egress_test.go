package livekit

import (
	"testing"

	"github.com/zoom/darsly/internal/pkg/config"
)

// BE-8: yozuv layout'i. Default `DefaultEgressLayout` bo'lishi va noma'lum
// qiymat default'ga qaytishi kerak (noto'g'ri env tufayli yozuv buzilmasin).
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

	// docs/PRODUCT.md «Yozuv sifati»: Zoom yozuvida kamera plitkasi YO'Q, faqat
	// kontent. "speaker" yon karusel ustunini chizib kadr enining ~1/4 ini olardi.
	if DefaultEgressLayout != "single-speaker" {
		t.Errorf("default layout %q — ulashilgan ekran BUTUN kadrni egallashi kerak (single-speaker)", DefaultEgressLayout)
	}
}

// ⭐ Qora yo'llar muammosi: kadr manba nisbatiga moslashadimi.
func TestPickRecordingSize(t *testing.T) {
	cases := []struct {
		name         string
		srcW, srcH   int
		wantW, wantH int
	}{
		{"noma'lum manba → bazaviy", 0, 0, 1280, 720},
		{"manfiy → bazaviy", -1, 720, 1280, 720},
		{"720p yotiq o'zgarmaydi", 1280, 720, 1280, 720},
		{"1080p → 720p ga tushadi", 1920, 1080, 1280, 720},
		{"16:10 planshet (Zoom fayli)", 1280, 800, 1280, 800},
		{"tik telefon 9:16", 1080, 1920, 720, 1280},
		{"tik telefon 9:20 (zamonaviy)", 1080, 2400, 576, 1280},
		{"kvadrat", 800, 800, 800, 800},
		{"kichik manba upscale QILINMAYDI", 640, 360, 640, 360},
		{"juda kichik manba minimalga ko'tariladi", 320, 240, 640, 480},
		{"ultra-keng nisbat qisiladi", 3840, 600, 1280, 512},
		{"g'alati ip-ingichka tik nisbat qisiladi", 100, 1000, 400, 1000},
	}
	for _, tc := range cases {
		w, h := pickRecordingSize(tc.srcW, tc.srcH)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("%s: pickRecordingSize(%d,%d) = %dx%d, kutilgan %dx%d",
				tc.name, tc.srcW, tc.srcH, w, h, tc.wantW, tc.wantH)
		}
	}
}

// H.264 (yuv420p) toq o'lchamni qabul qilmaydi — hech qanday manba toq
// natija bermasligi kerak.
func TestPickRecordingSizeAlwaysEven(t *testing.T) {
	for w := 1; w <= 2600; w += 7 {
		for _, h := range []int{1, 33, 361, 719, 1081, 2399} {
			gw, gh := pickRecordingSize(w, h)
			if gw%2 != 0 || gh%2 != 0 {
				t.Fatalf("pickRecordingSize(%d,%d) = %dx%d — toq o'lcham", w, h, gw, gh)
			}
			if gw < 16 || gh < 16 {
				t.Fatalf("pickRecordingSize(%d,%d) = %dx%d — juda kichik", w, h, gw, gh)
			}
		}
	}
}

// Bitrate: kadr kattalashsa oshadi, kichraysa PASAYMAYDI (matn o'qiluvchanligi).
func TestPickVideoBitrate(t *testing.T) {
	cases := []struct {
		w, h int
		want int32
	}{
		{1280, 720, 900},   // bazaviy
		{720, 1280, 900},   // tik — piksel bir xil
		{576, 1280, 900},   // tik va kichikroq → quyi chegara
		{640, 360, 900},    // kichik → quyi chegara
		{1280, 800, 1000},  // 16:10 → maydonga proporsional
		{2560, 1440, 1400}, // yuqori chegara
	}
	for _, tc := range cases {
		if got := pickVideoBitrate(tc.w, tc.h); got != tc.want {
			t.Errorf("pickVideoBitrate(%d,%d) = %d, kutilgan %d", tc.w, tc.h, got, tc.want)
		}
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
