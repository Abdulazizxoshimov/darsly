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
// "speaker": faol so'zlovchi/ekran ulashish asosiy oynada, qolganlar yon plitkalarda.
// "grid" emas, chunki grid'da slayd matni o'qilmas darajada kichik plitkaga tushadi
// (mobil ilovaning asosiy funksiyasi — ekran ulashish).
const DefaultEgressLayout = "speaker"

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
// Bunday kadrga 30 fps ham, 3 Mbit ham keragidan ortiq — fayl kattaligi
// yozuvni yuklab olishni ham, MinIO diskini ham behuda yeydi.
//
// Quyidagi qiymatlar Zoom bulut yozuvi bilan bir xil mantiqda tanlangan:
// past kadr chastotasi + past bitrate, lekin TO'LIQ 720p ravshanlik — chunki
// o'qilishi kerak bo'lgan narsa harakat emas, MATN.
//
//	1280x720 · 15 fps · video 900 kbps · audio 64 kbps (mono nutq uchun yetarli)
//	→ 2.5 soat ≈ 1.05 GB (avvalgi 3.5 GB o'rniga, ya'ni ~3.3 barobar kichik)
//
// Kalit kadr oralig'i 4 s: brauzerda oldinga/orqaga o'tish shu qadamda ishlaydi
// (kattaroq oraliq faylni yana kichraytiradi, lekin "sakrash" qo'polashadi).
const (
	recWidth        = 1280
	recHeight       = 720
	recFramerate    = 15
	recVideoKbps    = 900
	recAudioKbps    = 64
	recKeyFrameSecs = 4.0
)

// StartRoomRecording xonani MP4 sifatida MinIO'ga (S3) yozib olishni boshlaydi.
// objectKey — MinIO ichidagi yakuniy fayl yo'li. Egress ID qaytaradi.
func (c *Client) StartRoomRecording(ctx context.Context, roomName, objectKey string, s3 S3Config) (string, error) {
	req := &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   normalizeLayout(c.layout),
		Options: &livekit.RoomCompositeEgressRequest_Advanced{
			Advanced: &livekit.EncodingOptions{
				Width:            recWidth,
				Height:           recHeight,
				Framerate:        recFramerate,
				VideoCodec:       livekit.VideoCodec_H264_MAIN,
				VideoBitrate:     recVideoKbps,
				AudioCodec:       livekit.AudioCodec_AAC, // MP4 uchun standart
				AudioBitrate:     recAudioKbps,
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
