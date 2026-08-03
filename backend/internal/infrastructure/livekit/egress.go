package livekit

import (
	"context"
	"math"
	"net/http"
	"strings"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
)

// S3Config — Egress yozuvni yuklaydigan S3-mos (MinIO) manzil.
type S3Config struct {
	AccessKey string
	Secret    string
	Region    string
	Endpoint  string // http(s)://host:port — Egress konteyneri shu manzilga yozadi
	Bucket    string
}

// DefaultEgressLayout — dars yozuvi uchun standart kompozitsiya shabloni.
//
// ⭐ 2026-08-03 da "speaker" → "single-speaker" ga o'zgartirildi.
//
// Sabab — `livekit/egress:latest` konteyneridagi shablonning KOMPILYATSIYA
// QILINGAN JS'i o'qib chiqildi (`/usr/bin/egress` ichiga embed qilingan):
//
//	SingleSpeakerLayout = ({tracks}) => { const e = sortTracks(tracks, 1);
//	    return e.length === 0 ? null : <FocusLayout trackRef={e[0]}/> }
//
//	SpeakerLayout = ({tracks}) => { const e = sortTracks(tracks, 1),
//	    focus = e.shift(), rest = sortTracks(e, 3);
//	    ... return <div className="lk-focus-layout">
//	          <CarouselLayout tracks={rest}>…</CarouselLayout>
//	          <FocusLayout trackRef={focus}/> </div> }
//
// Ya'ni "speaker" `lk-focus-layout` GRID'ini chizadi: yon karusel ustun ALOHIDA
// joy egallaydi (kadr enining ~1/4 i) va u yerda o'quvchilarning bo'sh avatar
// plitkalari turadi. Kontentga qolgan maydon shu qadar kichrayadi.
// "single-speaker" esa faqat BIRINCHI trekni chizadi — `sortTracks` ekran
// ulashishni oldinga qo'yadi, ya'ni ulashilgan ekran BUTUN kadrni oladi.
//
// docs/PRODUCT.md «Yozuv sifati» o'lchovi: Zoom yozuvida kamera umuman yo'q,
// faqat kontent. Bu — aynan `single-speaker`.
//
// "grid" hech qachon default emas: grid'da slayd matni o'qilmas darajada kichik
// plitkaga tushadi (mobil ilovaning asosiy funksiyasi — ekran ulashish).
const DefaultEgressLayout = "single-speaker"

// egressLayouts — LiveKit default egress template qabul qiladigan qiymatlar.
// Manba (tekshirilgan): livekit/egress:latest ichidagi shablonning kompilyatsiya
// qilingan JS'i — `layout.startsWith("speaker")` → SpeakerLayout,
// `startsWith("single-speaker")` → SingleSpeakerLayout, aks holda GridLayout;
// mavzu esa `layout.endsWith("-light")` bilan aniqlanadi (default: dark).
var egressLayouts = map[string]bool{
	"speaker": true, "speaker-light": true,
	"single-speaker": true, "single-speaker-light": true,
	"grid": true, "grid-light": true,
}

// normalizeLayout konfiguratsiyadan kelgan layout nomini tekshiradi.
// Noma'lum/bo'sh qiymat → DefaultEgressLayout (noto'g'ri env tufayli yozuv
// kutilmagan ko'rinishda chiqib ketmasin).
func normalizeLayout(l string) string {
	l = strings.ToLower(strings.TrimSpace(l))
	if egressLayouts[l] {
		return l
	}
	return DefaultEgressLayout
}

// ⭐ YOZUV HAJMI — nega sozlamalar OSHKORA berilgan.
//
// Sozlama berilmasa LiveKit `H264_720P_30` presetini oladi: 1280x720, 30 fps,
// video 3000 kbps + audio 128 kbps. 2.5 soatlik dars uchun bu ≈ **3.5 GB**.
// Dars mazmuni esa deyarli statik: slayd, PDF, GeoGebra, gapirayotgan bosh.
// Bunday kadrga 3 Mbit keragidan ortiq — fayl kattaligi yozuvni yuklab olishni
// ham, MinIO diskini ham behuda yeydi.
//
// Bazaviy kadr 1280x720, video 900 kbps: o'qilishi kerak bo'lgan narsa harakat
// emas, MATN — ravshanlik saqlanadi, bitrate esa ~3.3 barobar past.
//
// ⭐ 2026-08-03 (docs/PRODUCT.md «Yozuv sifati: Zoom bilan taqqoslash»):
//   - `recFramerate` 15 → **25**. Zoom yozuvida 25 fps; 15 fps sichqoncha va
//     skroll harakatini uzuq-yuluq ko'rsatardi. Bitreytga ta'siri kichik,
//     chunki statik slaydda qo'shimcha kadrlar deyarli bo'sh chiqadi.
//   - `recAudioFreq` OSHKORA **48000**. Proto default'i 44100 (`AudioFrequency`
//     izohiga qara) — WebRTC/Opus manbasi esa 48 kHz, ya'ni egress ovozni
//     BEKORGA qayta diskretlab, 44.1 kHz mono chiqarardi (o'lchangan).
//     Stereo `EncodingOptions` orqali so'ralmaydi (protocol v1.49.0 da
//     `AudioChannels` maydoni YO'Q) — u transkod bosqichida beriladi
//     (`worker.TranscodeConfig.AudioChannels`).
//
// Kalit kadr oralig'i 4 s: brauzerda oldinga/orqaga o'tish shu qadamda ishlaydi
// (kattaroq oraliq faylni yana kichraytiradi, lekin "sakrash" qo'polashadi).
const (
	recBaseWidth    = 1280
	recBaseHeight   = 720
	recFramerate    = 25
	recVideoKbps    = 900
	recAudioKbps    = 64
	recAudioFreq    = 48000
	recKeyFrameSecs = 4.0
)

// ⭐ KADR O'LCHAMI MANBAGA MOSLASHADI — qora yo'llar muammosi.
//
// O'lchangan nuqson (docs/PRODUCT.md): tik (portrait) telefon ekrani qotib
// qolgan 1280x720 kadrga solinganda kontent kadrning atigi **22%** ini
// egallardi — chap va o'ngda 78% qora yo'l. Egress shabloni videoni
// `object-fit: contain` bilan chizadi, ya'ni kadr nisbati manbaga mos
// kelmasa qora yo'l MUQARRAR. Yechim — kadrni manba nisbatiga qurish.
//
// Cheklovlar:
//   - uzun tomon 1280 dan oshmaydi (bitrate/CPU byudjeti) va manbadan ham
//     katta bo'lmaydi — upscale bitreytni yeydi, ravshanlik qo'shmaydi;
//   - uzun tomon `recMinLongSide` dan kichik bo'lmaydi: juda kichik manba
//     (masalan 320x240 kamera) yozuvni o'qib bo'lmas holga keltirmasin;
//   - nisbat [recMinAspect, recMaxAspect] oralig'iga QISILADI: 9:20 telefon
//     (0.45) o'tadi, ammo g'alati/buzuq TrackInfo (masalan 8:1) kadrni
//     ip-ingichka qilib qo'ymaydi;
//   - ikkala o'lcham ham JUFT — H.264 (yuv420p) toq o'lchamni qabul qilmaydi.
const (
	recMaxLongSide = 1280
	recMinLongSide = 640
	recMinAspect   = 0.40 // ≈ 9:22 — eng tik telefon ekrani ham sig'adi
	recMaxAspect   = 2.50 // ≈ 21:9 — ultra-keng monitor
)

// pickRecordingSize manba trek o'lchamidan yozuv kadrini tanlaydi.
//
// SOF funksiya (LiveKit'ga bog'liq emas) — qaror shu yerda, chunki uni
// tarmoqsiz sinash mumkin va noto'g'ri kadr butun yozuvni buzadi.
// srcW/srcH noma'lum (0 yoki manfiy) bo'lsa bazaviy 1280x720 qaytadi.
func pickRecordingSize(srcW, srcH int) (int, int) {
	if srcW <= 0 || srcH <= 0 {
		return recBaseWidth, recBaseHeight
	}
	ar := float64(srcW) / float64(srcH)
	switch {
	case ar < recMinAspect:
		ar = recMinAspect
	case ar > recMaxAspect:
		ar = recMaxAspect
	}

	long := srcW
	if srcH > long {
		long = srcH
	}
	if long > recMaxLongSide {
		long = recMaxLongSide
	}
	if long < recMinLongSide {
		long = recMinLongSide
	}

	var w, h int
	if ar >= 1 { // yotiq yoki kvadrat
		w, h = long, int(math.Round(float64(long)/ar))
	} else { // tik
		h, w = long, int(math.Round(float64(long)*ar))
	}
	return evenDim(w), evenDim(h)
}

// evenDim o'lchamni juft songa keltiradi (H.264 yuv420p talabi) va 16 dan
// kichik bo'lib ketishiga yo'l qo'ymaydi.
func evenDim(v int) int {
	if v < 16 {
		return 16
	}
	return v - v%2
}

// pickVideoBitrate kadr maydoniga qarab video bitreytni tanlaydi.
//
// Bazaviy 900 kbps 1280x720 (≈0.92 Mpx) uchun o'lchangan. Kadr KATTALASHSA
// (masalan 1280x800 = 16:10 planshet) bir xil bitrate piksel boshiga kamayadi
// va matn xiralashardi — shuning uchun maydonga proporsional oshiriladi.
//
// Kadr KICHRAYSA bitrate PASAYTIRILMAYDI (900 — quyi chegara): tik telefon
// ekranida piksel kam, lekin aynan o'sha kadr to'la matndan iborat va
// ortiqcha bitlar to'g'ridan-to'g'ri o'qiluvchanlikka ketadi. Transkod
// bosqichi (CRF) ortiqchasini keyin baribir qirqadi.
func pickVideoBitrate(w, h int) int32 {
	const basePx = recBaseWidth * recBaseHeight
	px := w * h
	if px <= basePx {
		return recVideoKbps
	}
	kbps := recVideoKbps * px / basePx
	if kbps > recMaxVideoKbps {
		kbps = recMaxVideoKbps
	}
	return int32(kbps)
}

// recMaxVideoKbps — yuqori chegara: g'ayrioddiy katta kadrda ham 2.5 soatlik
// dars ~1.6 GB dan oshmasin.
const recMaxVideoKbps = 1400

// roomVideoSize xonadagi ENG MOS video trek o'lchamini qaytaradi.
//
// Ustuvorlik: ekran ulashish → kamera. Sabab — dars yozuvining mazmuni ekran;
// `single-speaker` shabloni ham aynan ekranni birinchi qilib chizadi, ya'ni
// kadr nisbati o'sha trekka mos bo'lishi kerak.
//
// Bir nechta nomzod bo'lsa maydoni kattasi olinadi (simulcast'da `TrackInfo`
// eng yuqori qatlam o'lchamini beradi).
//
// ok=false — xonada video trek yo'q yoki LiveKit javob bermadi. Bu ODATIY
// hol: yozuv birinchi `track_published` da boshlanadi va u ko'pincha MIKROFON
// bo'ladi (ekran ulashish keyinroq keladi). Shuning uchun bu yo'l "eng yaxshi
// harakat", kafolat emas — qora yo'llarni yakuniy yo'q qiluvchi bosqich
// transkoddagi `cropdetect` (internal/worker/videofilter.go).
func (c *Client) roomVideoSize(ctx context.Context, roomName string) (int, int, bool) {
	parts, err := c.ListParticipants(ctx, roomName)
	if err != nil {
		return 0, 0, false
	}
	var bestW, bestH int
	var bestScreen bool
	for _, p := range parts {
		for _, t := range p.GetTracks() {
			if t.GetType() != livekit.TrackType_VIDEO {
				continue
			}
			w, h := int(t.GetWidth()), int(t.GetHeight())
			if w <= 0 || h <= 0 {
				continue
			}
			screen := t.GetSource() == livekit.TrackSource_SCREEN_SHARE
			// Ekran ulashish kameradan HAR DOIM ustun; teng turda maydoni katta.
			if (screen && !bestScreen) || (screen == bestScreen && w*h > bestW*bestH) {
				bestW, bestH, bestScreen = w, h, screen
			}
		}
	}
	if bestW == 0 {
		return 0, 0, false
	}
	return bestW, bestH, true
}

// StartRoomRecording xonani MP4 sifatida MinIO'ga (S3) yozib olishni boshlaydi.
// objectKey — MinIO ichidagi yakuniy fayl yo'li. Egress ID qaytaradi.
func (c *Client) StartRoomRecording(ctx context.Context, roomName, objectKey string, s3 S3Config) (string, error) {
	// Kadr o'lchami xonadagi haqiqiy manbaga moslanadi (qora yo'llar bo'lmasin).
	// Manba topilmasa bazaviy 1280x720 — `pickRecordingSize(0,0)` shuni beradi.
	srcW, srcH, _ := c.roomVideoSize(ctx, roomName)
	w, h := pickRecordingSize(srcW, srcH)

	req := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   normalizeLayout(c.layout),
		Options: &livekit.RoomCompositeEgressRequest_Advanced{
			Advanced: &livekit.EncodingOptions{
				Width:            int32(w),
				Height:           int32(h),
				Framerate:        recFramerate,
				VideoCodec:       livekit.VideoCodec_H264_MAIN,
				VideoBitrate:     pickVideoBitrate(w, h),
				AudioCodec:       livekit.AudioCodec_AAC, // MP4 uchun standart
				AudioBitrate:     recAudioKbps,
				AudioFrequency:   recAudioFreq, // 48 kHz — Opus manbasi bilan bir xil
				KeyFrameInterval: recKeyFrameSecs,
			},
		},
		FileOutputs: []*livekit.EncodedFileOutput{
			{
				FileType: livekit.EncodedFileType_MP4,
				Filepath: objectKey,
				Output: &livekit.EncodedFileOutput_S3{
					S3: &livekit.S3Upload{
						AccessKey:      s3.AccessKey,
						Secret:         s3.Secret,
						Region:         s3.Region,
						Endpoint:       s3.Endpoint,
						Bucket:         s3.Bucket,
						ForcePathStyle: true, // MinIO uchun majburiy
					},
				},
			},
		},
	}
	info, err := c.egress.StartRoomCompositeEgress(ctx, req)
	if err != nil {
		return "", err
	}
	return info.EgressId, nil
}

// StopRecording yozib olishni to'xtatadi.
func (c *Client) StopRecording(ctx context.Context, egressID string) error {
	_, err := c.egress.StopEgress(ctx, &livekit.StopEgressRequest{EgressId: egressID})
	return err
}

// ParseWebhook LiveKit webhook so'rovini imzo bo'yicha tekshiradi va hodisani qaytaradi.
func (c *Client) ParseWebhook(r *http.Request) (*livekit.WebhookEvent, error) {
	provider := auth.NewSimpleKeyProvider(c.apiKey, c.apiSecret)
	return webhook.ReceiveWebhookEvent(r, provider)
}
