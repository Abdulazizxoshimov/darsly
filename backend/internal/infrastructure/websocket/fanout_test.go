package websocket

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/pkg/logger"
)

func testCache(t *testing.T) redis.Cache {
	t.Helper()
	host, port := "localhost", "6399"
	if v := os.Getenv("TEST_REDIS_HOST"); v != "" {
		host = v
	}
	if v := os.Getenv("TEST_REDIS_PORT"); v != "" {
		port = v
	}
	c, err := redis.New(config.RedisConfig{Host: host, Port: port})
	if err != nil {
		t.Skipf("redis mavjud emas: %v", err)
	}
	if err := c.Ping(context.Background()); err != nil {
		t.Skipf("redis ping: %v", err)
	}
	return c
}

func registerFake(h *Hub, userID string) *client {
	c := &client{userID: userID, send: make(chan []byte, 8), hub: h, rooms: map[string]struct{}{}}
	h.register(c)
	return c
}

// Instans A'dan yuborilgan xabar instans B'dagi klientga yetishi kerak (ko'p-instans).
func TestFanout_CrossInstance(t *testing.T) {
	cache := testCache(t)
	log := logger.New("error", "test", "test")
	ctx := context.Background()

	hubA := NewHub(log)
	hubA.EnableFanout(ctx, cache)
	defer hubA.Stop()
	hubB := NewHub(log)
	hubB.EnableFanout(ctx, cache)
	defer hubB.Stop()

	// user "u1" faqat B instansida ulangan.
	cB := registerFake(hubB, "u1")
	time.Sleep(300 * time.Millisecond) // subscribe goroutine tayyor bo'lsin

	hubA.Send("u1", NewNotificationMsg("u1", map[string]string{"hello": "world"}))

	select {
	case b := <-cB.send:
		if !strings.Contains(string(b), "world") {
			t.Fatalf("kutilmagan xabar: %s", b)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("instanslararo xabar yetkazilmadi (fanout ishlamadi)")
	}
}

// O'z instansida yuborilgan xabar takror (double) yetkazilmasligi kerak.
func TestFanout_NoDoubleDeliveryLocal(t *testing.T) {
	cache := testCache(t)
	log := logger.New("error", "test", "test")
	ctx := context.Background()

	hub := NewHub(log)
	hub.EnableFanout(ctx, cache)
	defer hub.Stop()

	c := registerFake(hub, "u2")
	time.Sleep(300 * time.Millisecond)

	hub.Send("u2", NewNotificationMsg("u2", map[string]string{"x": "1"}))

	// Bitta xabar kelishi kerak (lokal), fanout echo skip qilinadi.
	select {
	case <-c.send:
	case <-time.After(2 * time.Second):
		t.Fatal("lokal xabar yetkazilmadi")
	}
	select {
	case b := <-c.send:
		t.Fatalf("takroriy xabar yetkazildi (double-delivery): %s", b)
	case <-time.After(700 * time.Millisecond):
		// yaxshi — takror yo'q
	}
}
