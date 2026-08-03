package livekit

import (
	"context"
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
//   - `recFramerate` 15 → 25 (Zoom pariteti) — LEKIN 2026-08-04 da o'lchov
//     asosida 15 ga QAYTARILDI, quyidagi izohga qara.
//   - `recAudioFreq` OSHKORA **48000**. Proto default'i 44100 (`AudioFrequency`
//     izohiga qara) — WebRTC/Opus manbasi esa 48 kHz, ya'ni egress ovozni
//     BEKORGA qayta diskretlab, 44.1 kHz mono chiqarardi (o'lchangan).
//     Stereo `EncodingOptions` orqali so'ralmaydi (protocol v1.49.0 da
//     `AudioChannels` maydoni YO'Q) — u transkod bosqichida beriladi
//     (`worker.TranscodeConfig.AudioChannels`).
//
// Kalit kadr oralig'i 4 s: brauzerda oldinga/orqaga o'tish shu qadamda ishlaydi
// (kattaroq oraliq faylni yana kichraytiradi, lekin "sakrash" qo'polashadi).
//
// ⚠️ 2026-08-04 — `recFramerate` 25 dan **15** ga QAYTARILDI (o'lchov bilan).
// Sabab `recCanvasSide` izohidagi CPU bo'limida: 4 yadroli serverda egress
// 25 fps'da mashinani to'liq yeb qo'ydi va bir yozuv «pipeline frozen» bilan
// YO'QOLDI. Dars yozuvida asosiy talab — slayd MATNI o'qilishi, ya'ni
// rezolyutsiya; fps esa faqat sichqoncha/skroll silliqligiga ta'sir qiladi.
// Shuning uchun CPU byudjeti fps'dan olinib rezolyutsiyaga berildi.
// Kuchli serverda `RECORDING_FPS=25` bilan Zoom pariteti qaytariladi.
const (
	recBaseWidth    = 1280
	recBaseHeight   = 720
	recFramerate    = 15
	recMinFramerate = 5
	recMaxFramerate = 30
	recVideoKbps    = 900
	recAudioKbps    = 64
	recAudioFreq    = 48000
	recKeyFrameSecs = 4.0
)

// pickFramerate sozlamadagi fps'ni tekshiradi (0/chegaradan tashqari → default).
func pickFramerate(fps int) int32 {
	if fps < recMinFramerate || fps > recMaxFramerate {
		return recFramerate
	}
	return int32(fps)
}

// ⭐⭐ KADR KVADRAT — HAR QANDAY ORIENTATSIYA TO'LIQ REZOLYUTSIYADA SAQLANADI.
//
// ## Nima uchun kvadrat (2026-08-03, o'lchov bilan)
// Egress shabloni videoni `object-fit: contain` bilan chizadi: kadr nisbati
// manbaga mos kelmasa qora yo'l MUQARRAR va kontent kichrayadi. O'lchangan
// nuqson — tik telefon ekrani 1280x720 kadrga solinganda kontent kadrning
// 22% ini egallardi va yakuniy fayl 324x718 chiqardi (docs/PRODUCT.md).
//
// Birinchi urinish kadrni MANBA nisbatiga qurish edi. U ISHLAMAYDI, chunki:
//
//  1. **Vaqt.** Yozuv birinchi `track_published` da boshlanadi — u mikrofon
//     yoki kamera. Kadr tanlanayotganda ekran ulashish xonada hali YO'Q.
//  2. **LiveKit Android SDK 2.27.0 yolg'on o'lcham aytadi** (bytecode'dan
//     tekshirilgan): `AddTrackRequest` ga `options.captureParams` xom holda
//     ketadi, `LocalScreencastVideoTrack.startCapture()` esa orientatsiyani
//     faqat capturer uchun almashtiradi. Ya'ni tik telefonda kadr 576x1280
//     oqadi, `TrackInfo` esa 1280x576 deydi. Keyin ham yangilanmaydi.
//
// Kvadrat kadr IKKALA sababni ham chetlab o'tadi — manbani BILISH shart emas:
//
//	tik telefon 576x1280  → 1280x1280 ichida to'liq sig'adi → kesib → 576x1280
//	yotiq       1280x720  → to'liq sig'adi                  → kesib → 1280x720
//	planshet    1280x800  → to'liq sig'adi                  → kesib → 1280x800
//
// Ortiqcha maydonni transkoddagi `cropdetect` olib tashlaydi
// (internal/worker/videofilter.go) — u shu fayl uchun ISHLAB TURIBDI va
// asoschining haqiqiy yozuvida o'lchangan.
//
// ## ⚠️ CPU — DEFAULT 1024, 1280 EMAS (2026-08-04, serverda o'lchangan)
// Egress = headless Chrome + Xvfb + x264. Yuk kadr maydoniga ham, fps'ga ham
// proporsional. 4 yadroli serverda (u yerda postgres/redis/minio/caddy/
// prometheus/grafana/loki va ikki Telegram boti ham ishlaydi) o'lchandi:
//
//	kanvas    fps  piksel/s   natija
//	1280x720   15  13.8 M     ishlagan (eski holat)
//	1280x1280  25  41.0 M     ❌ «pipeline frozen» — YOZUV YO'QOLDI
//	1024x1024  25  26.2 M     ishlagan, lekin egress 1027% CPU, yuk 10-13
//	1024x1024  15  15.7 M     ✅ tanlangan default
//
// Yozuvni YO'QOTISH — sifatdan beqiyos qimmat, shuning uchun default eng
// xavfsiz nuqtada. Kuchliroq serverda `RECORDING_CANVAS=1280` +
// `RECORDING_FPS=25` bilan to'liq parite (tik kadr 576x1280) qaytariladi.
//
// ## Agar kesish ishlamay qolsa
// Fayl kvadrat bo'lib, kontent o'rtada qoladi: kompozitsiya bugungidan yomon,
// ammo REZOLYUTSIYA baribir yuqori. Ya'ni eng yomon holatda ham sifat pasaymaydi.
const (
	// recCanvasSide — kvadrat kadrning tomoni. `RECORDING_CANVAS` env bilan
	// almashtiriladi (quyi chegara `recMinCanvasSide`).
	recCanvasSide    = 1024
	recMinCanvasSide = 640
)

// pickRecordingSize yozuv kadrini qaytaradi.
//
// side — sozlamadagi kvadrat tomon (0 yoki juda kichik bo'lsa `recCanvasSide`).
// SOF funksiya: qaror shu yerda, chunki noto'g'ri kadr butun yozuvni buzadi va
// buni tarmoqsiz sinash kerak.
func pickRecordingSize(side int) (int, int) {
	if side < recMinCanvasSide {
		side = recCanvasSide
	}
	s := evenDim(side)
	return s, s
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

// ⚠️ `roomVideoSize` ATAYLAB YO'Q (2026-08-03 da olib tashlandi).
//
// U `ListParticipants` dan `TrackInfo.width/height` ni o'qib kadr nisbatini
// tanlardi. Ikki sababdan ishonchsiz — batafsili `recCanvasSide` izohida:
// (1) yozuv boshlanganda ekran ulashish hali xonada yo'q; (2) LiveKit Android
// SDK ekran treki uchun YOTIQ o'lcham e'lon qiladi, kadr esa tik oqadi.
// Kvadrat kadr ikkalasini ham keraksiz qiladi.

// StartRoomRecording xonani MP4 sifatida MinIO'ga (S3) yozib olishni boshlaydi.
// objectKey — MinIO ichidagi yakuniy fayl yo'li. Egress ID qaytaradi.
func (c *Client) StartRoomRecording(ctx context.Context, roomName, objectKey string, s3 S3Config) (string, error) {
	// Kadr KVADRAT — manba nisbatidan qat'i nazar kontent to'liq rezolyutsiyada
	// sig'adi; ortiqcha maydonni transkod kesadi (izoh: `recCanvasSide`).
	w, h := pickRecordingSize(c.canvas)

	req := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   normalizeLayout(c.layout),
		Options: &livekit.RoomCompositeEgressRequest_Advanced{
			Advanced: &livekit.EncodingOptions{
				Width:            int32(w),
				Height:           int32(h),
				Framerate:        pickFramerate(c.fps),
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
