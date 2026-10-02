package websocket

import (
	"testing"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// Bitta foydalanuvchi maxConnsPerUser'dan ortiq ulanish ocholmaydi; boshqasiga ta'sir yo'q.
func TestRegister_PerUserCap(t *testing.T) {
	h := NewHub(logger.New("error", "test", "test"))
	for i := 0; i < maxConnsPerUser; i++ {
		if !h.register(&client{userID: "u1", send: make(chan []byte, 1), hub: h}) {
			t.Fatalf("%d-ulanish qabul qilinishi kerak", i)
		}
	}
	if h.register(&client{userID: "u1", send: make(chan []byte, 1), hub: h}) {
		t.Fatal("per-user cap oshdi, rad etilishi kerak")
	}
	if !h.register(&client{userID: "u2", send: make(chan []byte, 1), hub: h}) {
		t.Fatal("boshqa foydalanuvchi bloklanmasligi kerak")
	}
	if got := h.Count(); got != int64(maxConnsPerUser)+1 {
		t.Fatalf("count=%d", got)
	}
}
