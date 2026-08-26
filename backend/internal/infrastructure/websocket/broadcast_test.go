package websocket

// B-8 — BroadcastToRoom + subscribeRoom/unsubscribeRoom fan-out.
//
// Xona-asosidagi tarqatish chat/poll/room-state real-time yangilanishlarining
// yagona kanali. Testlar: obuna bo'lgan ulanish xabarni oladi, obunani bekor
// qilgan OLMAYDI, va sekin iste'molchi (to'lgan bufer) BLOKLAMASDAN tashlanadi.

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/zoom/darsly/internal/pkg/logger"
)

func newTestHub() *Hub {
	return NewHub(logger.New("error", "test", "test"))
}

func roomMsg(room, body string) Message {
	return newMsg(TypeNotification, room, map[string]string{"body": body})
}

// Obuna bo'lgan ulanish xabarni oladi; boshqa xonadagi olmaydi.
// Bug: BroadcastToRoom noto'g'ri xona map'iga yozsa yoki hammaga yuborsa —
// bir darsning xabari boshqa darsga sizib chiqardi.
func TestBroadcastToRoom_DeliversOnlyToSubscribers(t *testing.T) {
	h := newTestHub()

	sub := registerFake(h, "u1")
	other := registerFake(h, "u2")
	h.subscribeRoom(sub, "lesson:A")
	h.subscribeRoom(other, "lesson:B") // boshqa xona

	h.BroadcastToRoom(roomMsg("lesson:A", "salom"))

	select {
	case b := <-sub.send:
		var m Message
		if err := json.Unmarshal(b, &m); err != nil || m.Room != "lesson:A" {
			t.Fatalf("kutilmagan xabar: %s (err=%v)", b, err)
		}
	default:
		t.Fatal("obuna bo'lgan ulanish xabarni olishi kerak")
	}

	select {
	case b := <-other.send:
		t.Fatalf("boshqa xonadagi ulanish xabar olmasligi kerak: %s", b)
	default:
		// yaxshi — sizib chiqmadi
	}
}

// Obunani bekor qilgan ulanish endi xabar OLMAYDI (aks holda chiqib ketgan
// ishtirokchi tark etgan xona xabarlarini olishda davom etardi).
func TestUnsubscribeRoom_StopsDelivery(t *testing.T) {
	h := newTestHub()
	c := registerFake(h, "u1")

	h.subscribeRoom(c, "lesson:A")
	h.unsubscribeRoom(c, "lesson:A")

	h.BroadcastToRoom(roomMsg("lesson:A", "salom"))

	select {
	case b := <-c.send:
		t.Fatalf("obunadan chiqqan ulanish xabar olmasligi kerak: %s", b)
	default:
	}
}

// Bo'sh xona (Room=="") tarqatilmaydi — noto'g'ri xabar butun mapni aylanib
// resurs sarflamasin.
func TestBroadcastToRoom_EmptyRoomIsNoop(t *testing.T) {
	h := newTestHub()
	c := registerFake(h, "u1")
	h.subscribeRoom(c, "")

	h.BroadcastToRoom(roomMsg("", "salom"))

	select {
	case b := <-c.send:
		t.Fatalf("bo'sh xona tarqatilmasligi kerak: %s", b)
	default:
	}
}

// ⭐ Sekin iste'molchi guard (back-pressure): send buferi to'lganda ulanish
// TASHLANADI, tarqatuvchi bloklanmaydi. Bug: `select default` bo'lmasa bitta
// sekin klient butun xonaga (barcha ishtirokchilar) real-time'ni to'xtatardi
// va gorutinani abadiy bloklab socket-leak yasardi.
func TestBroadcastToRoom_SlowConsumerDroppedNotBlocking(t *testing.T) {
	h := newTestHub()
	// Buferi 1 bo'lgan sekin klient (registerFake 8 beradi — kichigroq yasaymiz).
	slow := &client{userID: "slow", send: make(chan []byte, 1), hub: h, rooms: map[string]struct{}{}}
	h.register(slow)
	h.subscribeRoom(slow, "lesson:A")

	// Buferni to'ldiramiz, keyin yana ko'p yuboramiz — bloklashsiz qaytishi kerak.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			h.BroadcastToRoom(roomMsg("lesson:A", "spam"))
		}
		close(done)
	}()

	select {
	case <-done:
		// yaxshi — tarqatuvchi bloklanmadi
	case <-time.After(2 * time.Second):
		t.Fatal("sekin iste'molchi tarqatuvchini bloklab qo'ydi (back-pressure guard yo'q)")
	}

	// Bufer sig'imidan ortiq xabar YIG'ILMAGAN bo'lishi kerak (ortig'i tashlangan).
	if got := len(slow.send); got > cap(slow.send) {
		t.Fatalf("bufer sig'imidan oshdi: %d > %d", got, cap(slow.send))
	}
}
