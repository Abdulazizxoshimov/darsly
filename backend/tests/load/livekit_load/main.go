// LiveKit load test — N ta ishtirokchini bitta guruh darsiga haqiqiy ulaydi.
//
// Ishga tushirish (backend + LiveKit ishlab turgan holda):
//
//	go run ./tests/load/livekit_load -base http://localhost:8087 \
//	    -email dilnoza@darsly.uz -password parol12345 -n 20
//
// Bosqichlar: mentor login → dars yaratish → host token (xona ochiladi) →
// N guest joinlink orqali token oladi va LiveKit'ga ulanadi → ListParticipants tekshiradi.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

func main() {
	base := flag.String("base", "http://localhost:8087", "backend base URL")
	email := flag.String("email", "dilnoza@darsly.uz", "mentor email")
	password := flag.String("password", "parol12345", "mentor password")
	n := flag.Int("n", 20, "participant count")
	lkHost := flag.String("lk", "ws://localhost:7880", "livekit ws url")
	lkKey := flag.String("lk-key", "devkey", "livekit api key")
	lkSecret := flag.String("lk-secret", "secret_at_least_32_characters_long_000000", "livekit api secret")
	flag.Parse()

	access := login(*base, *email, *password)
	lessonID, slug := createLesson(*base, access)
	fmt.Printf("lesson=%s slug=%s\n", lessonID, slug)

	// Host token — xonani ochadi (dars live bo'ladi).
	postJSON(*base+"/api/v1/lessons/"+lessonID+"/token", access, nil)
	roomName := "lesson_" + lessonID

	var connected int64
	var wg sync.WaitGroup
	start := time.Now()
	rooms := make([]*lksdk.Room, *n)

	for i := 0; i < *n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			token := joinToken(*base, slug, fmt.Sprintf("Load-%02d", idx))
			if token == "" {
				fmt.Printf("  [%02d] no token\n", idx)
				return
			}
			room, err := lksdk.ConnectToRoomWithToken(*lkHost, token, nil)
			if err != nil {
				fmt.Printf("  [%02d] connect err: %v\n", idx, err)
				return
			}
			rooms[idx] = room
			atomic.AddInt64(&connected, 1)
		}(i)
	}
	wg.Wait()
	fmt.Printf("connected %d/%d in %s\n", connected, *n, time.Since(start).Round(time.Millisecond))

	// Serverdan haqiqiy ishtirokchilar sonini tekshirish.
	time.Sleep(2 * time.Second)
	rsc := lksdk.NewRoomServiceClient(httpFromWS(*lkHost), *lkKey, *lkSecret)
	resp, err := rsc.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{Room: roomName})
	if err != nil {
		fmt.Println("ListParticipants err:", err)
		os.Exit(1)
	}
	active := 0
	for _, p := range resp.Participants {
		if p.State == livekit.ParticipantInfo_ACTIVE {
			active++
		}
	}
	fmt.Printf("LiveKit xonasida ACTIVE ishtirokchilar: %d\n", active)

	for _, r := range rooms {
		if r != nil {
			r.Disconnect()
		}
	}
	if int(connected) < *n {
		os.Exit(1)
	}
}

func login(base, email, password string) string {
	body := postJSON(base+"/api/v1/auth/login", "", map[string]string{"email": email, "password": password})
	return gjson(body, "data", "access_token")
}

func createLesson(base, access string) (string, string) {
	body := postJSON(base+"/api/v1/lessons", access, map[string]any{
		"title": "Load Test Guruh Darsi", "is_waiting_room_enabled": false,
	})
	return gjson(body, "data", "id"), gjson(body, "data", "join_slug")
}

func joinToken(base, slug, name string) string {
	body := postJSON(base+"/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": name})
	return gjson(body, "data", "room", "token")
}

func postJSON(url, access string, payload any) []byte {
	var buf io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		buf = bytes.NewReader(b)
	}
	req, _ := http.NewRequest("POST", url, buf)
	req.Header.Set("Content-Type", "application/json")
	if access != "" {
		req.Header.Set("Authorization", "Bearer "+access)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("http err:", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data
}

// gjson — bog'liqliksiz mayda JSON path o'quvchi (faqat obyekt kalitlari).
func gjson(data []byte, path ...string) string {
	var v any
	if json.Unmarshal(data, &v) != nil {
		return ""
	}
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		v = m[k]
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func httpFromWS(u string) string {
	if len(u) > 5 && u[:5] == "wss:/" {
		return "https:" + u[4:]
	}
	if len(u) > 4 && u[:4] == "ws:/" {
		return "http:" + u[3:]
	}
	return u
}
