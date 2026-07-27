package entity

// RoomToken — LiveKit xonasiga ulanish uchun klientga qaytariladigan ma'lumot.
type RoomToken struct {
	Token    string `json:"token"`     // LiveKit access JWT
	WSURL    string `json:"ws_url"`    // signaling manzili (ws://...)
	RoomName string `json:"room_name"` // LiveKit xona nomi
	Identity string `json:"identity"`  // ishtirokchi identifikatori
	Role     string `json:"role"`      // host | participant
}

// Room rollari
const (
	RoomRoleHost        = "host"
	RoomRoleCoHost      = "co-host"
	RoomRoleParticipant = "participant"
)

// RoomParticipant — xonadagi ishtirokchining ochiq ko'rinishi (host boshqaruvi uchun).
type RoomParticipant struct {
	Identity   string `json:"identity"`
	Name       string `json:"name"`
	JoinedAtMs int64  `json:"joined_at_ms"`
	Active     bool   `json:"active"`
	AudioMuted bool   `json:"audio_muted"`
	VideoMuted bool   `json:"video_muted"`
}
