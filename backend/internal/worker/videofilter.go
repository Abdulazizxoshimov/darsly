package worker

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Bu fayl — transkod bosqichining QAROR mantiqi: ffmpeg tahlil chiqishidan
// (`cropdetect`, `blackdetect`, `silencedetect`) qanday kesish qilinishini
// hisoblaydi.
//
// ## Nega sof funksiyalar (ffmpeg chaqiruvidan ajratilgan)
// Noto'g'ri `crop` yoki noto'g'ri `-ss` — bu **yozuvning bir qismini butunlay
// yo'q qilish**. Ustoz uchun bu dars yo'qolgani bilan barobar. Shuning uchun
// qaror haqiqiy video yasamasdan, o'nlab chekka holat bilan sinaladi; ffmpeg
// chaqiruvining o'zi esa (transcode.go) shunchaki simlash bo'lib qoladi.

// CropRect — ffmpeg `crop=W:H:X:Y` parametri.
type CropRect struct{ W, H, X, Y int }

// timeRange — soniyalardagi oraliq (blackdetect / silencedetect natijasi).
type timeRange struct{ Start, End float64 }

var cropRe = regexp.MustCompile(`crop=(-?\d+):(-?\d+):(-?\d+):(-?\d+)`)

// parseCropDetect ffmpeg chiqishidan barcha `crop=w:h:x:y` natijalarini oladi.
//
// Yaroqsiz qatorlar (`crop=-1:-1:-1:-1` — butun kadr qora bo'lganda cropdetect
// shuni beradi) TASHLAB YUBORILADI: ular kontent yo'qligini bildiradi, kesish
// o'lchamini emas.
func parseCropDetect(ffmpegLog string) []CropRect {
	var out []CropRect
	for _, m := range cropRe.FindAllStringSubmatch(ffmpegLog, -1) {
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		x, _ := strconv.Atoi(m[3])
		y, _ := strconv.Atoi(m[4])
		if w <= 0 || h <= 0 || x < 0 || y < 0 {
			continue
		}
		out = append(out, CropRect{W: w, H: h, X: x, Y: y})
	}
	return out
}

// unionCrop bir nechta namunani QAMROVCHI to'rtburchakni qaytaradi.
//
// ⚠️ Bu ataylab "eng kattasini tanlash" emas, balki BIRLASHMA. Sabab: bitta
// lahzada slayd qora fonli bo'lsa cropdetect kontentni kichikroq ko'radi;
// boshqa lahzada kattaroq. Birlashma har bir namunadan katta yoki teng, ya'ni
// «eng katta kontent maydoni» talabini qanoatlantiradi va ustiga qo'shimcha
// xavfsizlik beradi — hech bir namunadagi kontent kesilib ketmaydi.
func unionCrop(rs []CropRect) (CropRect, bool) {
	if len(rs) == 0 {
		return CropRect{}, false
	}
	x1, y1 := rs[0].X, rs[0].Y
	x2, y2 := rs[0].X+rs[0].W, rs[0].Y+rs[0].H
	for _, r := range rs[1:] {
		x1 = min(x1, r.X)
		y1 = min(y1, r.Y)
		x2 = max(x2, r.X+r.W)
		y2 = max(y2, r.Y+r.H)
	}
	return CropRect{W: x2 - x1, H: y2 - y1, X: x1, Y: y1}, true
}

// Kesish qarorining chegaralari.
const (
	// cropMinAreaSaving — kesish faqat maydonning shuncha ulushi tejalsa
	// qo'llanadi. Bir necha pikselli "kesish" uchun butun faylni qayta kodlash
	// (va xato qilish xavfi) mantiqsiz.
	cropMinAreaSaving = 0.05
	// cropMinSideRatio — natija manbaning shuncha ulushidan kichik bo'lsa
	// cropdetect ADASHGAN deb hisoblanadi (masalan butun dars qorong'i slayd
	// bo'lsa). Bunday holda kesilmaydi: buzilgan yozuvdan ko'ra qora yo'lli
	// yozuv ming marta yaxshi.
	cropMinSideRatio = 0.25
	// cropMinSide — mutlaq quyi chegara (piksel).
	cropMinSide = 160
)

// decideCrop namunalardan yakuniy `crop` parametrini hisoblaydi.
//
// ok=false — kesilmaydi va manba fayl kadr o'lchamida qoladi. Bu YAGONA
// xavfsiz default: har qanday shubhada tegmaymiz.
func decideCrop(samples []CropRect, srcW, srcH int) (CropRect, bool) {
	if srcW <= 0 || srcH <= 0 {
		return CropRect{}, false
	}
	u, ok := unionCrop(samples)
	if !ok {
		return CropRect{}, false
	}

	// Manba chegarasiga qisamiz: cropdetect kadrdan chiqib ketgan qiymat
	// bermasligi kerak, lekin bergan taqdirda ham ffmpeg yiqilmasin.
	x1 := max(0, u.X)
	y1 := max(0, u.Y)
	x2 := min(srcW, u.X+u.W)
	y2 := min(srcH, u.Y+u.H)
	if x2 <= x1 || y2 <= y1 {
		return CropRect{}, false
	}

	// JUFT piksel (H.264 yuv420p): boshlanish nuqtasini pastga, o'lchamni
	// imkon bo'lsa YUQORIGA yaxlitlaymiz — kesish kontentni yemasin.
	x1 -= x1 % 2
	y1 -= y1 % 2
	w := x2 - x1
	h := y2 - y1
	if w%2 != 0 {
		if x1+w < srcW {
			w++
		} else {
			w--
		}
	}
	if h%2 != 0 {
		if y1+h < srcH {
			h++
		} else {
			h--
		}
	}

	// Aql-idrok tekshiruvlari — cropdetect adashgan bo'lsa tegmaymiz.
	if w < cropMinSide || h < cropMinSide {
		return CropRect{}, false
	}
	if float64(w) < cropMinSideRatio*float64(srcW) || float64(h) < cropMinSideRatio*float64(srcH) {
		return CropRect{}, false
	}
	if w > srcW || h > srcH || x1+w > srcW || y1+h > srcH {
		return CropRect{}, false
	}
	// Sezilarli tejamkorlik bo'lmasa qayta kodlash uchun sabab yo'q.
	if float64(w*h) > (1-cropMinAreaSaving)*float64(srcW*srcH) {
		return CropRect{}, false
	}
	return CropRect{W: w, H: h, X: x1, Y: y1}, true
}

var (
	blackRe    = regexp.MustCompile(`black_start:(-?[0-9.]+)\s+black_end:(-?[0-9.]+)`)
	silStartRe = regexp.MustCompile(`silence_start:\s*(-?[0-9.]+)`)
	silEndRe   = regexp.MustCompile(`silence_end:\s*(-?[0-9.]+)`)
)

// parseBlackDetect `blackdetect` filtridan qora oraliqlarni oladi.
func parseBlackDetect(ffmpegLog string) []timeRange {
	var out []timeRange
	for _, m := range blackRe.FindAllStringSubmatch(ffmpegLog, -1) {
		s, err1 := strconv.ParseFloat(m[1], 64)
		e, err2 := strconv.ParseFloat(m[2], 64)
		if err1 != nil || err2 != nil || e <= s {
			continue
		}
		out = append(out, timeRange{Start: s, End: e})
	}
	return out
}

// parseSilenceDetect `silencedetect` filtridan jimlik oraliqlarini oladi.
//
// Filtr `silence_start` va `silence_end` ni ALOHIDA qatorlarda chiqaradi,
// shuning uchun ular ketma-ket juftlanadi. Oxirgi `silence_start` juftsiz
// qolsa (jimlik fayl oxirigacha davom etgan) `duration` bilan yopiladi.
func parseSilenceDetect(ffmpegLog string, duration float64) []timeRange {
	var out []timeRange
	open := false
	var start float64
	for _, line := range strings.Split(ffmpegLog, "\n") {
		if m := silStartRe.FindStringSubmatch(line); m != nil {
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			// Manfiy qiymat REAL: silencedetect birinchi jimlikni ba'zan
			// -0.02 dan boshlaydi (buferlash). Nolga qisamiz.
			start = math.Max(0, v)
			open = true
			continue
		}
		if m := silEndRe.FindStringSubmatch(line); m != nil && open {
			v, err := strconv.ParseFloat(m[1], 64)
			if err != nil {
				continue
			}
			if v > start {
				out = append(out, timeRange{Start: start, End: v})
			}
			open = false
		}
	}
	if open && duration > start {
		out = append(out, timeRange{Start: start, End: duration})
	}
	return out
}

// Boshidagi "o'lik" qismni kesish chegaralari.
const (
	// leadStartTol — oraliq "boshidan" hisoblanishi uchun qancha kechikishga
	// yo'l qo'yiladi.
	leadStartTol = 0.35
	// leadSafety — kesishdan qoldiriladigan zaxira: birinchi haqiqiy kadr yoki
	// birinchi so'zning boshi qirqilib ketmasin.
	leadSafety = 0.30
	// leadMinTrim — bundan kichik kesish arzimaydi.
	leadMinTrim = 1.5
	// leadMaxTrim — yuqori chegara. Undan uzun "qora va jim" boshlanish
	// ehtimoli past va bunday holda tahlil xato bo'lgan bo'lishi mumkin.
	leadMaxTrim = 60.0
)

// leadingDeadSeconds yozuv boshidan necha soniya kesilishini qaytaradi.
//
// ## ⭐ NEGA blackdetect YOLG'IZ YETARLI EMAS
// Egress yozuvi boshida ~15 soniya qora kadr bo'ladi: LiveKit egress shabloni
// birinchi AUDIO trek e'lon qilinishi bilan yozishni boshlaydi (kompilyatsiya
// qilingan shablonda: `!hasVideo && subscribed && elapsed>500ms` →
// `startRecording()`), ekran ulashish esa keyinroq keladi.
//
// Lekin o'sha 15 soniyada ustoz KO'PINCHA gapiradi («salom, hozir ekranni
// ulashaman»). Faqat qora kadrga qarab kesish — o'sha gapni O'CHIRIB yuborish,
// ya'ni mazmun yo'qotish. Dars yozuvida esa asosiy mazmun — ovoz.
//
// Shuning uchun kesish faqat video QORA **va** ovoz JIM bo'lgan boshlang'ich
// qismga qo'llanadi. Ikkovi kesishgan joy — egress ulanib, hali hech qanday
// media kelmagan payt, ya'ni haqiqatan bo'sh oraliq.
//
// hasAudio=false (ovoz oqimi umuman yo'q) bo'lsa faqat qora oraliq ishlatiladi —
// bu holda yo'qotadigan ovozning o'zi yo'q.
func leadingDeadSeconds(black, silence []timeRange, duration float64, hasAudio bool) float64 {
	b := leadingEnd(black)
	if b <= 0 {
		return 0 // boshi qora emas — kesadigan narsa yo'q
	}
	trim := b
	if hasAudio {
		s := leadingEnd(silence)
		if s <= 0 {
			// Qora, lekin OVOZLI boshlanish: ustoz gapiryapti — tegmaymiz.
			return 0
		}
		trim = math.Min(trim, s)
	}
	if trim > leadMaxTrim {
		trim = leadMaxTrim
	}
	trim -= leadSafety
	if trim < leadMinTrim {
		return 0
	}
	// Yozuvning yarmidan ko'pini kesish — deyarli aniq tahlil xatosi.
	if duration > 0 && trim > duration/2 {
		return 0
	}
	return math.Round(trim*10) / 10
}

// leadingEnd — faylning O'ZIDAN boshlanadigan oraliqning tugash vaqti (yo'q bo'lsa 0).
func leadingEnd(rs []timeRange) float64 {
	for _, r := range rs {
		if r.Start <= leadStartTol {
			return r.End
		}
	}
	return 0
}
