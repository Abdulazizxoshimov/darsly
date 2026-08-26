// sfu_media_load — SFU FAN-OUT ni HAQIQIY media bilan yuklaydigan test.
//
// NEGA KERAK (audit topilmasi):
//
//	Eski `livekit_load` faqat HTTP token berish TEZLIGINI (join/s) o'lchaydi.
//	U xonaga ulanadi, lekin nil callback bilan — HECH NARSAGA obuna bo'lmaydi,
//	media qabul qilmaydi, dekodlamaydi. Ya'ni asosiy bo'g'iq joy —
//	SFU FAN-OUT (1 publisher → N subscriber video oqimini oladi) — hech qachon
//	sinovdan o'tmagan. Real dars sig'imi ~60–100 ishtirokchi/xona (200 EMAS),
//	va u ham tasdiqlanmagan.
//
// BU VOSITA nima qiladi:
//   - Har xonada 1 ta HAQIQIY publisher (audio + ixtiyoriy video, belgilangan bitrate).
//   - N ta HAQIQIY subscriber: auto-subscribe YOQILGAN, OnTrackSubscribed callback
//     RTP paketlarini o'qiydi va kadr/bayt hisoblaydi (haqiqatan media OLADI).
//   - Sessiyani belgilangan muddat ushlab turadi; davriy hisobot beradi:
//     ulangan soni, media OLAYOTGAN subscriber soni, bayt/s, paket/s, xatolar.
//
// REJIMLAR:
//
//	media     (standart) — L-1: 1 pub + N sub, media fan-out.
//	multiroom (-rooms>1) — L-2: ~1000 ishtirokchini 10–16 xonaga yoyib, agregat
//	                       SFU CPU/bandwidth o'lchash.
//	reconnect (-reconnect) — L-3: ommaviy bir vaqtli uzilish+qayta ulanish
//	                       (blip / SFU restart) va tiklanish vaqtini o'lchash.
//
// TOKEN: to'g'ridan-to'g'ri LiveKit API kalit/sekret bilan mint qilinadi
// (livekit_load bilan bir xil default: devkey / secret_...). Bu backend
// rate-limit'iga bog'lanmasdan sof SFU sig'imini o'lchashga imkon beradi.
// Backend joinlink oqimini `livekit_load` sinaydi — bu vosita ATAYIN faqat SFU.
//
// ⚠️ KLIENT O'ZI BO'G'IQ JOY: ~150 dan ortiq WebRTC peer'ni bitta JARAYON
// ko'tara olmaydi (ICE/DTLS handshake raqobati; livekit_load da hujjatlangan).
// Shuning uchun 1000 ga yetish uchun BIR NECHA JARAYON/MASHINADA ishga tushiring:
//
//	-rooms 16 -n 80 → agregat, lekin bitta jarayonda OG'IR (CPU/FD chegarasi).
//	Amaliyroq: har xonaga alohida jarayon:
//	  P0: sfu_media_load -rooms 1 -n 80 -room-offset 0
//	  P1: sfu_media_load -rooms 1 -n 80 -room-offset 1   ... (turli mashinalarda)
//	`-batch` / `-batch-delay` to'lqin-to'lqin ulanish klient chegarasini yumshatadi.
//
// Ishga tushirish (LiveKit :7880 ishlab turgan holda):
//
//	go run ./tests/load/sfu_media_load -n 60 -dur 60s
//	go run ./tests/load/sfu_media_load -rooms 16 -n 80 -dur 120s      # L-2 agregat
//	go run ./tests/load/sfu_media_load -n 60 -reconnect -reconnect-cycles 3  # L-3
package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

// subMetrics — bitta subscriber uchun jonli hisoblagichlar (atomik).
type subMetrics struct {
	packets   atomic.Int64
	bytes     atomic.Int64
	receiving atomic.Bool // kamida bitta media paket keldimi
	connected atomic.Bool
}

type config struct {
	lkHost    string
	lkKey     string
	lkSecret  string
	roomPref  string
	roomOff   int
	rooms     int
	n         int
	dur       time.Duration
	warmup    time.Duration
	report    time.Duration
	videoKbps int
	fps       int
	pubVideo  bool
	batch     int
	batchDly  time.Duration
	tokenTTL  time.Duration
	// L-3
	reconnect bool
	rcCycles  int
	rcStagger time.Duration
}

func main() {
	c := config{}
	flag.StringVar(&c.lkHost, "lk", "ws://localhost:7880", "livekit ws url")
	flag.StringVar(&c.lkKey, "lk-key", "devkey", "livekit api key")
	flag.StringVar(&c.lkSecret, "lk-secret", "secret_at_least_32_characters_long_000000", "livekit api secret")
	flag.StringVar(&c.roomPref, "room-prefix", "sfuload_", "xona nomi prefiksi")
	flag.IntVar(&c.roomOff, "room-offset", 0, "xona indeks boshlanishi (ko'p jarayonda takrorlanmasin)")
	flag.IntVar(&c.rooms, "rooms", 1, "xonalar soni (L-2 multiroom uchun 10–16)")
	flag.IntVar(&c.n, "n", 60, "har xonadagi subscriber soni (~60–100 real sig'im)")
	flag.DurationVar(&c.dur, "dur", 60*time.Second, "sessiyani ushlab turish muddati")
	flag.DurationVar(&c.warmup, "warmup", 5*time.Second, "hammasi ulangach media o'lchashdan oldingi kutish")
	flag.DurationVar(&c.report, "report", 5*time.Second, "hisobot oralig'i")
	flag.IntVar(&c.videoKbps, "video-kbps", 600, "publisher video bitrate maqsadi (kbps)")
	flag.IntVar(&c.fps, "fps", 30, "publisher video kadr tezligi")
	flag.BoolVar(&c.pubVideo, "publish-video", true, "video trek e'lon qilish (fan-out og'ir yo'li)")
	flag.IntVar(&c.batch, "batch", 25, "bir to'lqinda ochiladigan subscriber soni")
	flag.DurationVar(&c.batchDly, "batch-delay", 1500*time.Millisecond, "to'lqinlar orasidagi pauza")
	flag.DurationVar(&c.tokenTTL, "token-ttl", 6*time.Hour, "mint qilingan token amal muddati")
	flag.BoolVar(&c.reconnect, "reconnect", false, "L-3: ommaviy uzilish+qayta ulanish bo'roni")
	flag.IntVar(&c.rcCycles, "reconnect-cycles", 1, "L-3: qayta ulanish sikllari soni")
	flag.DurationVar(&c.rcStagger, "reconnect-stagger", 0, "L-3: qayta ulanishlar orasidagi tarqoqlik (0=bir vaqtda)")
	flag.Parse()

	fmt.Printf("SFU media load: rooms=%d n/room=%d (jami sub=%d) video=%v %dkbps dur=%s\n",
		c.rooms, c.n, c.rooms*c.n, c.pubVideo, c.videoKbps, c.dur)

	var wg sync.WaitGroup
	fail := int64(0)
	for r := 0; r < c.rooms; r++ {
		roomName := fmt.Sprintf("%s%d", c.roomPref, c.roomOff+r)
		wg.Add(1)
		go func(room string) {
			defer wg.Done()
			if err := runRoom(&c, room); err != nil {
				fmt.Printf("[%s] XONA XATOSI: %v\n", room, err)
				atomic.AddInt64(&fail, 1)
			}
		}(roomName)
	}
	wg.Wait()

	if fail > 0 {
		fmt.Printf("\n%d xona xato bilan tugadi\n", fail)
		os.Exit(1)
	}
	fmt.Println("\ntamom.")
}

func runRoom(c *config, roomName string) error {
	// 1) Publisher — xonani ochadi va media manbasini beradi.
	pubStop := make(chan struct{})
	pubRoom, err := startPublisher(c, roomName, pubStop)
	if err != nil {
		return fmt.Errorf("publisher: %w", err)
	}
	defer func() {
		close(pubStop)
		pubRoom.Disconnect()
	}()

	// 2) Subscriber'lar.
	metrics := make([]*subMetrics, c.n)
	rooms := make([]*lksdk.Room, c.n)
	tokens := make([]string, c.n)
	for i := range metrics {
		metrics[i] = &subMetrics{}
	}

	connectAll := func() {
		var wg sync.WaitGroup
		for i := 0; i < c.n; i += c.batch {
			end := i + c.batch
			if end > c.n {
				end = c.n
			}
			for j := i; j < end; j++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					tk := tokens[idx]
					if tk == "" {
						tk = mintToken(c, roomName, fmt.Sprintf("sub-%s-%03d", roomName, idx), false)
						tokens[idx] = tk
					}
					room, err := connectSubscriber(c, tk, metrics[idx])
					if err != nil {
						fmt.Printf("  [%s/%03d] ulanish xatosi: %v\n", roomName, idx, err)
						return
					}
					rooms[idx] = room
					metrics[idx].connected.Store(true)
				}(j)
			}
			if end < c.n {
				time.Sleep(c.batchDly)
			}
		}
		wg.Wait()
	}

	start := time.Now()
	connectAll()
	connected := countConnected(metrics)
	fmt.Printf("[%s] ulangan %d/%d (%s)\n", roomName, connected, c.n, time.Since(start).Round(time.Millisecond))

	// Media o'rnashsin.
	time.Sleep(c.warmup)

	if c.reconnect {
		reconnectStorm(c, roomName, rooms, tokens, metrics)
	}

	// 3) O'lchov davri — davriy hisobot.
	holdAndReport(c, roomName, metrics)

	// Yakuniy xulosa.
	rx := countReceiving(metrics)
	totPkts, totBytes := totals(metrics)
	fmt.Printf("[%s] YAKUN: ulangan=%d/%d media_olayotgan=%d paket=%d bayt=%.1fMB\n",
		roomName, countConnected(metrics), c.n, rx, totPkts, float64(totBytes)/1e6)

	for _, room := range rooms {
		if room != nil {
			room.Disconnect()
		}
	}
	return nil
}

// startPublisher — xonaga ulanib audio (+ixtiyoriy video) e'lon qiladi.
func startPublisher(c *config, roomName string, stop <-chan struct{}) (*lksdk.Room, error) {
	token := mintToken(c, roomName, "publisher-"+roomName, true)
	room, err := lksdk.ConnectToRoomWithToken(c.lkHost, token, lksdk.NewRoomCallback())
	if err != nil {
		return nil, err
	}

	// Audio (Opus) — publish_probe bilan bir xil isbotlangan yo'l.
	atrack, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2,
	})
	if err != nil {
		room.Disconnect()
		return nil, err
	}
	if _, err := room.LocalParticipant.PublishTrack(atrack, &lksdk.TrackPublicationOptions{
		Name: "audio", Source: livekit.TrackSource_MICROPHONE,
	}); err != nil {
		room.Disconnect()
		return nil, err
	}
	go func() {
		silence := []byte{0xf8, 0xff, 0xfe} // Opus 20ms jimlik
		t := time.NewTicker(20 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				_ = atrack.WriteSample(media.Sample{Data: silence, Duration: 20 * time.Millisecond}, nil)
			}
		}
	}()

	// Video (H264) — fan-out ning OG'IR yo'li. Maqsadli bitrate uchun kadr
	// o'lchamini hisoblaymiz. SFU media'ni dekodlamaydi — faqat uzatadi —
	// shuning uchun soxta (start-code + tasodifiy) NAL yetarli: RTP oqadi,
	// subscriber baytlarni sanaydi. Bu paket darajasida fan-out yukini beradi.
	if c.pubVideo {
		vtrack, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeH264, ClockRate: 90000,
		})
		if err != nil {
			room.Disconnect()
			return nil, err
		}
		if _, err := room.LocalParticipant.PublishTrack(vtrack, &lksdk.TrackPublicationOptions{
			Name: "video", Source: livekit.TrackSource_CAMERA, VideoWidth: 1280, VideoHeight: 720,
		}); err != nil {
			room.Disconnect()
			return nil, err
		}
		fps := c.fps
		if fps <= 0 {
			fps = 30
		}
		bytesPerFrame := (c.videoKbps * 1000 / 8) / fps
		if bytesPerFrame < 64 {
			bytesPerFrame = 64
		}
		frameDur := time.Second / time.Duration(fps)
		go func() {
			t := time.NewTicker(frameDur)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					_ = vtrack.WriteSample(media.Sample{Data: fakeNAL(bytesPerFrame), Duration: frameDur}, nil)
				}
			}
		}()
	}
	return room, nil
}

// fakeNAL — Annex-B start-code + tasodifiy payload'li soxta NAL birligi.
// SFU faqat uzatadi (dekodlamaydi), shuning uchun mazmun muhim emas —
// muhimi maqsadli bitrate'da haqiqiy RTP paketlar oqishi.
func fakeNAL(size int) []byte {
	b := make([]byte, size)
	b[0], b[1], b[2], b[3] = 0x00, 0x00, 0x00, 0x01
	b[4] = 0x41 // non-IDR slice NAL header (soxta)
	_, _ = rand.Read(b[5:])
	return b
}

// connectSubscriber — auto-subscribe YOQILGAN holda ulanadi va kelgan har bir
// trekni goroutine'da o'qib kadr/bayt sanaydi (HAQIQATAN media OLADI).
func connectSubscriber(c *config, token string, m *subMetrics) (*lksdk.Room, error) {
	cb := lksdk.NewRoomCallback()
	cb.OnTrackSubscribed = func(track *webrtc.TrackRemote, pub *lksdk.RemoteTrackPublication, rp *lksdk.RemoteParticipant) {
		go drainTrack(track, m)
	}
	return lksdk.ConnectToRoomWithToken(c.lkHost, token, cb, lksdk.WithAutoSubscribe(true))
}

// drainTrack — RTP paketlarini uzluksiz o'qiydi (SFU'dan media qabul qilinishi).
// Bu bo'lmasa subscriber passiv qoladi va fan-out yuki YO'Q — aynan eski
// vositaning nuqsoni shu edi.
func drainTrack(track *webrtc.TrackRemote, m *subMetrics) {
	buf := make([]byte, 1500)
	for {
		n, _, err := track.Read(buf)
		if err != nil {
			return
		}
		m.packets.Add(1)
		m.bytes.Add(int64(n))
		if n > 0 {
			m.receiving.Store(true)
		}
	}
}

// holdAndReport — sessiyani ushlab turadi, har `report` oralig'ida delta beradi.
func holdAndReport(c *config, roomName string, metrics []*subMetrics) {
	deadline := time.Now().Add(c.dur)
	var prevPkts, prevBytes int64
	tick := time.NewTicker(c.report)
	defer tick.Stop()
	// staticcheck S1000: bitta case'li select o'rniga `for range`.
	for range tick.C {
		pkts, bytes := totals(metrics)
		dp := pkts - prevPkts
		db := bytes - prevBytes
		prevPkts, prevBytes = pkts, bytes
		sec := c.report.Seconds()
		fmt.Printf("[%s] t+%02.0fs ulangan=%d media=%d %.0f paket/s %.2f Mbit/s (agregat)\n",
			roomName, time.Until(deadline).Seconds()*-1+c.dur.Seconds(),
			countConnected(metrics), countReceiving(metrics),
			float64(dp)/sec, float64(db)*8/1e6/sec)
		if time.Now().After(deadline) {
			return
		}
	}
}

// reconnectStorm (L-3) — hamma subscriber'ni bir vaqtda uzadi, keyin qayta
// ulaydi va to'liq tiklanish (hammasi qayta media olguncha) vaqtini o'lchaydi.
func reconnectStorm(c *config, roomName string, rooms []*lksdk.Room, tokens []string, metrics []*subMetrics) {
	for cyc := 1; cyc <= c.rcCycles; cyc++ {
		fmt.Printf("[%s] ⚡ reconnect storm sikl %d/%d — hammasi uzilyapti\n", roomName, cyc, c.rcCycles)
		for i, room := range rooms {
			if room != nil {
				room.Disconnect()
			}
			metrics[i].connected.Store(false)
			metrics[i].receiving.Store(false)
		}
		time.Sleep(500 * time.Millisecond) // blip

		t0 := time.Now()
		var wg sync.WaitGroup
		for i := 0; i < c.n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				if c.rcStagger > 0 {
					time.Sleep(time.Duration(idx) * c.rcStagger)
				}
				room, err := connectSubscriber(c, tokens[idx], metrics[idx])
				if err != nil {
					fmt.Printf("  [%s/%03d] qayta ulanish xatosi: %v\n", roomName, idx, err)
					return
				}
				rooms[idx] = room
				metrics[idx].connected.Store(true)
			}(i)
		}
		wg.Wait()
		reConn := time.Since(t0)

		// Media qayta oqguncha kutamiz (max 15s).
		mediaBack := waitReceiving(metrics, c.n*8/10, 15*time.Second)
		fmt.Printf("[%s] ✅ sikl %d tiklanish: qayta_ulanish=%s media_qaytdi(80%%)=%s ulangan=%d media=%d\n",
			roomName, cyc, reConn.Round(time.Millisecond), mediaBack.Round(time.Millisecond),
			countConnected(metrics), countReceiving(metrics))
	}
}

// waitReceiving — kamida `want` subscriber media olguncha kutadi, o'tgan vaqtni qaytaradi.
func waitReceiving(metrics []*subMetrics, want int, timeout time.Duration) time.Duration {
	t0 := time.Now()
	deadline := t0.Add(timeout)
	for time.Now().Before(deadline) {
		if countReceiving(metrics) >= want {
			return time.Since(t0)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return time.Since(t0)
}

// mintToken — LiveKit access token'ni to'g'ridan-to'g'ri mint qiladi.
func mintToken(c *config, room, identity string, publish bool) string {
	sub, pub := true, publish
	grant := &auth.VideoGrant{
		RoomJoin:     true,
		Room:         room,
		CanSubscribe: &sub,
		CanPublish:   &pub,
	}
	at := auth.NewAccessToken(c.lkKey, c.lkSecret).
		SetIdentity(identity).
		SetName(identity).
		SetVideoGrant(grant).
		SetValidFor(c.tokenTTL)
	tk, err := at.ToJWT()
	if err != nil {
		fmt.Println("token mint xatosi:", err)
		os.Exit(1)
	}
	return tk
}

func countConnected(m []*subMetrics) int {
	n := 0
	for _, s := range m {
		if s.connected.Load() {
			n++
		}
	}
	return n
}

func countReceiving(m []*subMetrics) int {
	n := 0
	for _, s := range m {
		if s.receiving.Load() {
			n++
		}
	}
	return n
}

func totals(m []*subMetrics) (pkts, bytes int64) {
	for _, s := range m {
		pkts += s.packets.Load()
		bytes += s.bytes.Load()
	}
	return
}
