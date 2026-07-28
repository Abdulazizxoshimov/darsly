// publish_probe — xonaga ulanib HAQIQIY audio trek e'lon qiladi.
//
// NEGA KERAK: yozib olish `track_published` webhook'ida boshlanadi (5 daqiqalik
// "Start signal not received" tuzog'i tuzatilgandan keyin). Buni tekshirish
// uchun xonada haqiqatan media bo'lishi shart — shunchaki ulanish YETARLI EMAS,
// va aynan shu farq tuzoqni tug'dirgan edi. `livekit_load` klienti ulanadi,
// lekin trek e'lon qilmaydi, ya'ni bu bo'shliqni yopa olmaydi.
//
//	go run ./tests/load/publish_probe -lk wss://... -token <room-token> -sec 25
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

func main() {
	lk := flag.String("lk", "", "LiveKit ws URL")
	token := flag.String("token", "", "room access token")
	sec := flag.Int("sec", 25, "necha soniya e'lon qilib tursin")
	flag.Parse()
	if *lk == "" || *token == "" {
		fmt.Println("-lk va -token majburiy")
		os.Exit(2)
	}

	room, err := lksdk.ConnectToRoomWithToken(*lk, *token, nil)
	if err != nil {
		fmt.Println("ulanmadi:", err)
		os.Exit(1)
	}
	defer room.Disconnect()
	fmt.Println("✅ xonaga ulandi:", room.Name())

	track, err := lksdk.NewLocalSampleTrack(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2,
	})
	if err != nil {
		fmt.Println("trek yasalmadi:", err)
		os.Exit(1)
	}
	if _, err := room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
		Name: "probe", Source: 2, // MICROPHONE
	}); err != nil {
		fmt.Println("e'lon qilinmadi:", err)
		os.Exit(1)
	}
	fmt.Println("✅ audio trek E'LON QILINDI — track_published webhook'i ketishi kerak")

	// Opus jimlik kadri (20 ms). Mazmun muhim emas — muhimi trek TIRIK bo'lishi.
	silence := []byte{0xf8, 0xff, 0xfe}
	deadline := time.Now().Add(time.Duration(*sec) * time.Second)
	for time.Now().Before(deadline) {
		if err := track.WriteSample(media.Sample{Data: silence, Duration: 20 * time.Millisecond}, nil); err != nil {
			fmt.Println("sample yozilmadi:", err)
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	fmt.Printf("✅ %d soniya e'lon qilindi, chiqilyapti\n", *sec)
}
