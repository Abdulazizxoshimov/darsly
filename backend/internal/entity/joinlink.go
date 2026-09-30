package entity

import "time"

// JoinLessonReq — havola (slug) orqali darsga kirish so'rovi.
type JoinLessonReq struct {
	GuestName *string `json:"guest_name" validate:"omitempty,min=2,max=100"`
	Passcode  *string `json:"passcode"   validate:"omitempty,max=20"`
}

// LessonPublic — slug orqali ochiq ko'rsatiladigan xavfsiz maydonlar (parol/mentor ID yashirin).
type LessonPublic struct {
	ID                   string     `json:"id"`
	Title                string     `json:"title"`
	MentorName           string     `json:"mentor_name"`
	ScheduledAt          *time.Time `json:"scheduled_at,omitempty"`
	Status               string     `json:"status"`
	HasPasscode          bool       `json:"has_passcode"`
	IsWaitingRoomEnabled bool       `json:"is_waiting_room_enabled"`
	// Ovoz siyosati — o'quvchi mikrofon tugmasini TO'G'RI ko'rsatishi uchun.
	//
	// # Nega SERVERDAN
	//
	// Avval o'quvchi `allow_self_unmute` ni faqat USTOZ KLIENTI yuboradigan
	// data-message'dan bilardi. Mobil ustoz esa uni umuman yubormaydi, ya'ni
	// telefondan o'tilgan darsda o'quvchining tugmasi "yoqish mumkin" deb
	// yolg'on ko'rsatardi: bosilganda server jimgina qayta mute qilardi
	// (`room.EnforceAudioPolicy`) va foydalanuvchi uchun bu "mikrofon buzuq"
	// bo'lib ko'rinardi. Siyosat serverda saqlanadi — javobda ham serverdan
	// kelishi kerak, klientlararo kelishuvdan emas.
	//
	// Bu FAQAT ko'rsatkich: haqiqiy chegara baribir serverda (webhook).
	MuteOnEntry     bool `json:"mute_on_entry"`
	AllowSelfUnmute bool `json:"allow_self_unmute"`
}

// JoinLessonResp — kirish so'rovi natijasi.
// NextStep: "waiting_room" (mentor tasdiqlashi kerak), "join" (to'g'ridan-to'g'ri)
// yoki "lesson_ended" (dars yakunlangan/bekor qilingan — havola endi faqat
// ma'lumot ko'rsatadi, token BERILMAYDI; aniq holat Lesson.Status'da).
type JoinLessonResp struct {
	Lesson    *LessonPublic `json:"lesson"`
	NextStep  string        `json:"next_step"`            // waiting_room | join | lesson_ended | waiting_for_host
	Room      *RoomToken    `json:"room,omitempty"`       // next_step=="join" bo'lsa — LiveKit tokeni
	RequestID string        `json:"request_id,omitempty"` // next_step=="waiting_room" bo'lsa — WS/status uchun
}

// JoinLessonResp.NextStep qiymatlari (klientlar bilan shartnoma).
const (
	JoinNextStepJoin        = "join"
	JoinNextStepWaitingRoom = "waiting_room"
	JoinNextStepLessonEnded = "lesson_ended"
	// JoinNextStepWaitingForHost — dars hali `live` emas (host kirmagan): token
	// BERILMAYDI, klient keyinroq qayta urinadi (polling/retry).
	JoinNextStepWaitingForHost = "waiting_for_host"
)
