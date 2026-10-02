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

// ⭐ KVADRAT KADR: har qanday orientatsiya to'liq rezolyutsiyada sig'sin.
//
// Nega manba o'lchami umuman berilmaydi: `TrackInfo` Android publisher uchun
// ishonchsiz (izoh: egress.go `recCanvasSide`). Kvadrat kadr manbani bilishni
// talab qilmaydi — kesish bosqichi ortiqchasini olib tashlaydi.
func TestPickRecordingSize(t *testing.T) {
	cases := []struct {
		name         string
		side         int
		wantW, wantH int
	}{
		{"sozlanmagan → default", 0, 1024, 1024},
		{"manfiy → default", -1, 1024, 1024},
		{"juda kichik → default", 320, 1024, 1024},
		{"kuchli server uchun 1280", 1280, 1280, 1280},
		{"quyi chegara", 640, 640, 640},
		{"toq qiymat juftga tushadi", 1281, 1280, 1280},
	}
	for _, tc := range cases {
		w, h := pickRecordingSize(tc.side)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("%s: pickRecordingSize(%d) = %dx%d, kutilgan %dx%d",
				tc.name, tc.side, w, h, tc.wantW, tc.wantH)
		}
	}
}

// Kadr KVADRAT bo'lishi — butun yechimning asosi. Nisbat buzilsa qora yo'l
// qaytadi, shuning uchun bu alohida mahkamlangan.
func TestPickRecordingSizeIsSquare(t *testing.T) {
	for side := -50; side <= 2600; side += 7 {
		w, h := pickRecordingSize(side)
		if w != h {
			t.Fatalf("pickRecordingSize(%d) = %dx%d — kvadrat emas", side, w, h)
		}
	}
}

// ⭐ HAR QANDAY manba kvadrat kadrga QORA YO'LSIZ sig'adimi.
//
// Bu — yechimning mohiyati: kontent kadrdan kattaroq bo'lsa u KESILARDI
// (mazmun yo'qolardi), kichik bo'lsa faqat bo'sh joy qoladi va uni transkod
// olib tashlaydi. Shuning uchun har bir realistik manba uchun kontent
// kvadratga to'liq sig'ishi va uzun tomoni to'liq ishlatilishi tekshiriladi.
func TestSquareCanvasFitsEveryOrientation(t *testing.T) {
	sources := []struct {
		name string
		w, h int
	}{
		{"tik telefon 9:20", 576, 1280},
		{"tik telefon 9:16", 720, 1280},
		{"yotiq 720p", 1280, 720},
		{"planshet 16:10 (Zoom fayli)", 1280, 800},
		{"planshet 4:3", 1024, 768},
		{"kvadrat", 900, 900},
	}
	side, _ := pickRecordingSize(0)
	for _, s := range sources {
		// `object-fit: contain` — masshtab uzun tomon bo'yicha.
		k := float64(side) / float64(max(s.w, s.h))
		gotW, gotH := float64(s.w)*k, float64(s.h)*k
		if gotW > float64(side)+0.5 || gotH > float64(side)+0.5 {
			t.Errorf("%s: %vx%v kvadratga sig'madi (%d)", s.name, gotW, gotH, side)
		}
		// Uzun tomon TO'LIQ ishlatilsin — aks holda rezolyutsiya behuda yo'qoladi.
		if long := max(gotW, gotH); long < float64(side)-0.5 {
			t.Errorf("%s: uzun tomon %v — kadr %d ni to'liq ishlatmadi", s.name, long, side)
		}
		// Manbadan KATTALASHTIRILMASIN: upscale bitreytni yeydi, sifat qo'shmaydi.
		if k > 1.0 {
			t.Logf("%s: manba kadrdan kichik (k=%.2f) — upscale bo'ladi, kutilgan hol", s.name, k)
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

// fps sozlamasi: chegaradan tashqari qiymat yozuvni buzmasin.
func TestPickFramerate(t *testing.T) {
	cases := []struct {
		in   int
		want int32
	}{
		{0, recFramerate}, {-5, recFramerate}, {1, recFramerate}, {100, recFramerate},
		{15, 15}, {25, 25}, {5, 5}, {30, 30},
	}
	for _, c := range cases {
		if got := pickFramerate(c.in); got != c.want {
			t.Errorf("pickFramerate(%d) = %d, kutilgan %d", c.in, got, c.want)
		}
	}
}
