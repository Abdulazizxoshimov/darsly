package worker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Bu testlar `videofilter.go` dagi QARORni qamrab oladi.
//
// Nega alohida va batafsil: noto'g'ri `crop` yoki noto'g'ri `-ss` — dars
// mazmunini butunlay yo'q qilish demak (docs/PRODUCT.md «Yozuv sifati»).
// Shuning uchun har bir «kesmaslik» sharti aniq test bilan mahkamlangan —
// keyinchalik chegaralarni beparvo yumshatib yuborishning iloji bo'lmasin.

func TestParseCropDetect_SkipsInvalidRows(t *testing.T) {
	log := `
[Parsed_cropdetect_0 @ 0x1] x1:279 x2:998 y1:0 y2:719 w:720 h:720 x:280 y:0 pts:1 t:0.04 crop=720:720:280:0
[Parsed_cropdetect_0 @ 0x1] x1:0 x2:0 y1:0 y2:0 w:0 h:0 x:0 y:0 pts:2 t:0.08 crop=-1:-1:-1:-1
[Parsed_cropdetect_0 @ 0x1] crop=720:718:280:2
`
	got := parseCropDetect(log)
	require.Equal(t, []CropRect{
		{W: 720, H: 720, X: 280, Y: 0},
		{W: 720, H: 718, X: 280, Y: 2},
	}, got, "crop=-1:... butun kadr qora degani — o'lcham emas, tashlanishi kerak")
}

func TestParseCropDetect_Empty(t *testing.T) {
	require.Empty(t, parseCropDetect("hech qanday cropdetect yo'q"))
}

// Birlashma: har bir namunadan KATTA yoki teng. Aks holda bir lahzada qora
// bo'lgan slayd butun yozuvni kesib tashlardi.
func TestUnionCrop_CoversEverySample(t *testing.T) {
	got, ok := unionCrop([]CropRect{
		{W: 100, H: 100, X: 50, Y: 50},  // x 50..150 · y 50..150
		{W: 100, H: 100, X: 100, Y: 20}, // x 100..200 · y 20..120
	})
	require.True(t, ok)
	// x: 50..200 → 150 · y: 20..150 → 130
	require.Equal(t, CropRect{X: 50, Y: 20, W: 150, H: 130}, got)
}

func TestUnionCrop_NoSamples(t *testing.T) {
	_, ok := unionCrop(nil)
	require.False(t, ok)
}

// ⭐ ASOSIY HOL: tik telefon ekrani 1280x720 kadrga solingan (2026-08-01 nuqsoni).
// Kontent 405x720 markazda, chap/o'ngda qora yo'l. Kesish QO'LLANISHI kerak.
func TestDecideCrop_PortraitPillarbox(t *testing.T) {
	samples := []CropRect{{W: 406, H: 720, X: 437, Y: 0}}
	got, ok := decideCrop(samples, 1280, 720)
	require.True(t, ok, "78%% qora yo'l — albatta kesilishi kerak")
	// X 437 (toq) → 436 ga tushadi; kenglik 843-436=407 (toq) → 408 ga
	// KO'TARILADI. Ya'ni kontent (437..843) butunlay ichkarida qoladi —
	// yaxlitlash hech qachon kontentni yemaydi.
	require.Equal(t, CropRect{W: 408, H: 720, X: 436, Y: 0}, got)
	require.LessOrEqual(t, got.X, 437, "chap chegara kontentdan tashqarida")
	require.GreaterOrEqual(t, got.X+got.W, 437+406, "o'ng chegara kontentdan tashqarida")
}

// Toq o'lcham H.264 (yuv420p) da ishlamaydi — natija HAR DOIM juft.
func TestDecideCrop_AlwaysEven(t *testing.T) {
	cases := [][4]int{
		{437, 1, 405, 719}, // x,y,w,h — hammasi toq
		{1, 3, 999, 555},
	}
	for _, c := range cases {
		got, ok := decideCrop([]CropRect{{X: c[0], Y: c[1], W: c[2], H: c[3]}}, 1280, 720)
		if !ok {
			continue // kesilmasa ham mayli — bu test faqat JUFTLIKni tekshiradi
		}
		require.Zero(t, got.X%2, "X juft bo'lsin: %+v", got)
		require.Zero(t, got.Y%2, "Y juft bo'lsin: %+v", got)
		require.Zero(t, got.W%2, "W juft bo'lsin: %+v", got)
		require.Zero(t, got.H%2, "H juft bo'lsin: %+v", got)
		require.LessOrEqual(t, got.X+got.W, 1280)
		require.LessOrEqual(t, got.Y+got.H, 720)
	}
}

// Arzimas kesish (bir necha piksel) uchun butun faylni qayta kodlash mantiqsiz.
func TestDecideCrop_TinySavingRejected(t *testing.T) {
	// 1272x716 ≈ manba maydonining 99% i — 5% chegaradan past.
	_, ok := decideCrop([]CropRect{{W: 1272, H: 716, X: 4, Y: 2}}, 1280, 720)
	require.False(t, ok, "5%%dan kam tejamkorlikda tegilmasin")
}

// ⭐ XAVFSIZLIK: cropdetect adashsa (butun dars qorong'i slayd) KESILMAYDI.
// Buzilgan yozuvdan ko'ra qora yo'lli yozuv ming marta yaxshi.
func TestDecideCrop_SuspiciouslySmallRejected(t *testing.T) {
	_, ok := decideCrop([]CropRect{{W: 200, H: 100, X: 500, Y: 300}}, 1280, 720)
	require.False(t, ok, "manbaning 25%%idan kichik natija — tahlil xatosi")
}

func TestDecideCrop_OutOfBoundsClamped(t *testing.T) {
	// cropdetect kadrdan chiqib ketgan qiymat bergan holat.
	got, ok := decideCrop([]CropRect{{W: 800, H: 900, X: 600, Y: 0}}, 1280, 720)
	require.True(t, ok)
	require.LessOrEqual(t, got.X+got.W, 1280)
	require.LessOrEqual(t, got.Y+got.H, 720)
}

func TestDecideCrop_NoSamplesOrBadSource(t *testing.T) {
	_, ok := decideCrop(nil, 1280, 720)
	require.False(t, ok)
	_, ok = decideCrop([]CropRect{{W: 400, H: 400, X: 0, Y: 0}}, 0, 0)
	require.False(t, ok, "manba o'lchami noma'lum — tegilmasin")
}

func TestParseBlackDetect(t *testing.T) {
	log := `
[blackdetect @ 0x1] black_start:0 black_end:13.44 black_duration:13.44
[blackdetect @ 0x1] black_start:80.2 black_end:81.0 black_duration:0.8
[blackdetect @ 0x1] black_start:90 black_end:90
`
	require.Equal(t, []timeRange{{0, 13.44}, {80.2, 81.0}}, parseBlackDetect(log),
		"end<=start bo'lgan qator tashlanadi")
}

func TestParseSilenceDetect_PairsLines(t *testing.T) {
	log := `
[silencedetect @ 0x1] silence_start: -0.02
[silencedetect @ 0x1] silence_end: 12.5 | silence_duration: 12.5
[silencedetect @ 0x1] silence_start: 40
[silencedetect @ 0x1] silence_end: 41.25 | silence_duration: 1.25
`
	require.Equal(t, []timeRange{{0, 12.5}, {40, 41.25}}, parseSilenceDetect(log, 120),
		"manfiy start nolga qisiladi (buferlash tufayli real hol)")
}

// Yopilmagan `silence_start` — jimlik skanerlangan oxirgacha davom etgan.
func TestParseSilenceDetect_UnclosedUsesDuration(t *testing.T) {
	log := "[silencedetect @ 0x1] silence_start: 5\n"
	require.Equal(t, []timeRange{{5, 30}}, parseSilenceDetect(log, 30))
}

// ⭐ ASOSIY HOL: egress boshidagi ~13 s qora VA jim qism kesiladi.
func TestLeadingDead_BlackAndSilent(t *testing.T) {
	got := leadingDeadSeconds(
		[]timeRange{{0, 13.4}},
		[]timeRange{{0, 14.0}},
		3600, true,
	)
	// min(13.4, 14.0) - leadSafety(0.30) = 13.1
	require.InDelta(t, 13.1, got, 0.01)
}

// ⭐ ENG MUHIM KAFOLAT: qora, LEKIN ustoz gapiryapti → KESILMAYDI.
// Dars yozuvida asosiy mazmun ovoz; «salom, hozir ekranni ulashaman» ni
// o'chirish — mazmun yo'qotish.
func TestLeadingDead_BlackButSpeaking(t *testing.T) {
	require.Zero(t, leadingDeadSeconds(
		[]timeRange{{0, 20}}, // 20 s qora
		nil,                  // jimlik yo'q — gapiryapti
		3600, true,
	), "ovozli boshlanish hech qachon kesilmaydi")
}

// Jimlik qoradan qisqa bo'lsa — ovoz boshlangan payt chegara bo'ladi.
func TestLeadingDead_TakesShorterOfTwo(t *testing.T) {
	got := leadingDeadSeconds(
		[]timeRange{{0, 30}}, // 30 s qora
		[]timeRange{{0, 8}},  // 8 s dan keyin gapira boshladi
		3600, true,
	)
	require.InDelta(t, 7.7, got, 0.01, "ovoz boshlangan joygacha kesiladi")
}

// Ovoz oqimi umuman yo'q — yo'qotadigan narsa yo'q, faqat qora bo'yicha.
func TestLeadingDead_NoAudioStream(t *testing.T) {
	got := leadingDeadSeconds([]timeRange{{0, 10}}, nil, 3600, false)
	require.InDelta(t, 9.7, got, 0.01)
}

func TestLeadingDead_Guards(t *testing.T) {
	t.Run("boshi qora emas", func(t *testing.T) {
		require.Zero(t, leadingDeadSeconds([]timeRange{{5, 20}}, []timeRange{{0, 20}}, 3600, true))
	})
	t.Run("arzimas kesish", func(t *testing.T) {
		require.Zero(t, leadingDeadSeconds([]timeRange{{0, 1.6}}, []timeRange{{0, 1.6}}, 3600, true),
			"1.6-0.3=1.3 < leadMinTrim")
	})
	t.Run("yozuvning yarmidan ko'pi", func(t *testing.T) {
		require.Zero(t, leadingDeadSeconds([]timeRange{{0, 20}}, []timeRange{{0, 20}}, 30, true),
			"30 s yozuvdan 19.7 s kesish — deyarli aniq tahlil xatosi")
	})
	t.Run("yuqori chegara", func(t *testing.T) {
		got := leadingDeadSeconds([]timeRange{{0, 300}}, []timeRange{{0, 300}}, 7200, true)
		require.InDelta(t, leadMaxTrim-leadSafety, got, 0.01)
	})
	t.Run("hech narsa yo'q", func(t *testing.T) {
		require.Zero(t, leadingDeadSeconds(nil, nil, 3600, true))
	})
}

// O'lchov nuqtalari fayl ichida qoladi (`-ss` fayl oxiridan chiqib ketmasin).
func TestSampleOffsets_StayInsideFile(t *testing.T) {
	for _, dur := range []float64{0, 5, 12, 60, 9000} {
		for _, off := range sampleOffsets(dur) {
			require.GreaterOrEqual(t, off, 0.0, "duration=%v", dur)
			if dur >= 3*cropSampleSecs {
				require.LessOrEqual(t, off+cropSampleSecs, dur+0.001, "duration=%v", dur)
			}
		}
	}
	require.Len(t, sampleOffsets(3600), 3, "uzun yozuvda uch nuqta")
	require.Len(t, sampleOffsets(4), 1, "qisqa yozuvda bitta")
}
