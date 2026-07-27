package websocket

import (
	"encoding/json"
	"time"
)

type MessageType string

const (
	TypeNotification MessageType = "notification"
	TypeSubscribe    MessageType = "subscribe"
	TypeUnsubscribe  MessageType = "unsubscribe"
)

// Message is the envelope sent over the WebSocket connection.
type Message struct {
	Type      MessageType     `json:"type"`
	Room      string          `json:"room,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// Send delivers msg to all connections of userID — lokal darhol, va ko'p-instans
// muhitida boshqa instanslarga Redis fan-out orqali. Sekin klientlar tashlanadi.
func (h *Hub) Send(userID string, msg Message) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.localSend(userID, b)     // shu instans — darhol
	h.publishFanout(userID, b) // boshqa instanslar — Redis orqali
}

func newMsg(msgType MessageType, room string, payload any) Message {
	raw, _ := json.Marshal(payload)
	return Message{Type: msgType, Room: room, Payload: raw, CreatedAt: time.Now().UTC()}
}

// NewNotificationMsg — foydalanuvchiga bildirishnoma xabari.
func NewNotificationMsg(userID string, payload any) Message {
	return newMsg(TypeNotification, "user:"+userID, payload)
}
