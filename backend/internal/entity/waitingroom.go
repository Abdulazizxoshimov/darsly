package entity

import "time"

// WaitingRoomRequest — kutish xonasidagi kirish so'rovi.
type WaitingRoomRequest struct {
	ID            string     `json:"id"`
	LessonID      string     `json:"lesson_id"`
	RequesterName string     `json:"requester_name"`
	GuestIdentity string     `json:"-"` // admit bo'lganda ishlatiladigan LiveKit identity
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	DecidedAt     *time.Time `json:"decided_at,omitempty"`
}

// Kutish so'rovi statuslari
const (
	WaitingStatusPending  = "pending"
	WaitingStatusAdmitted = "admitted"
	WaitingStatusRejected = "rejected"
)

// WaitingRoomStatusResp — guest so'rovining hozirgi holati (public polling / WS uchun).
// Status=="admitted" bo'lsa Room to'ldiriladi.
type WaitingRoomStatusResp struct {
	RequestID string     `json:"request_id"`
	Status    string     `json:"status"`
	Room      *RoomToken `json:"room,omitempty"`
}
