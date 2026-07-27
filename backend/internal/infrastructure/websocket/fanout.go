package websocket

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
)

// fanoutChannel — barcha instanslar tinglaydigan Redis pub/sub kanali.
const fanoutChannel = "ws:fanout"

// fanoutEnvelope — instanslararo yuboriladigan xabar konverti.
type fanoutEnvelope struct {
	Origin string `json:"o"` // yuborgan instans ID'si
	UserID string `json:"u"` // qabul qiluvchi userID
	Data   []byte `json:"d"` // tayyor JSON xabar (Message)
}

// EnableFanout Redis pub/sub fan-out'ini yoqadi: boshqa instanslardan kelgan
// xabarlar shu instansning lokal klientlariga yetkaziladi. app.go da bir marta chaqiriladi.
// Yoqilmasa (masalan testlarda) Hub lokal-only ishlaydi.
func (h *Hub) EnableFanout(ctx context.Context, cache redis.Cache) {
	if cache == nil {
		return
	}
	h.cache = cache
	h.instanceID = uuid.NewString()

	ps := cache.Subscribe(ctx, fanoutChannel)
	if ps == nil {
		h.log.Warn(ctx, "ws fanout: subscribe returned nil — running local-only")
		h.cache = nil
		return
	}

	go func() {
		ch := ps.Channel()
		for {
			select {
			case <-h.done:
				_ = ps.Close()
				return
			case m, ok := <-ch:
				if !ok {
					return
				}
				var env fanoutEnvelope
				if err := json.Unmarshal([]byte(m.Payload), &env); err != nil {
					continue
				}
				if env.Origin == h.instanceID {
					continue // o'z xabari — allaqachon lokal yetkazilgan
				}
				h.localSend(env.UserID, env.Data)
			}
		}
	}()

	h.log.Info(ctx, "ws fanout enabled (multi-instance)", logger.String("instance_id", h.instanceID))
}

// localSend faqat shu instansdagi userID ulanishlariga yetkazadi.
// RLock butun sikl davomida ushlanadi — aks holda unregister `close(c.send)` bilan
// send-on-closed panic va concurrent-map-iteration crash yuz beradi. Yuborish
// non-blocking (select default) bo'lgani uchun deadlock yo'q.
func (h *Hub) localSend(userID string, b []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- b:
		default: // sekin klient — tashlab yuboriladi
			metrics.WSDropped.Inc()
		}
	}
}

// publishFanout xabarni boshqa instanslarga tarqatadi (fanout yoqilgan bo'lsa).
func (h *Hub) publishFanout(userID string, b []byte) {
	if h.cache == nil {
		return
	}
	env := fanoutEnvelope{Origin: h.instanceID, UserID: userID, Data: b}
	if err := h.cache.Publish(context.Background(), fanoutChannel, env); err != nil {
		h.log.Warn(context.Background(), "ws fanout: publish failed", logger.SafeString("err", err.Error()))
	}
}
