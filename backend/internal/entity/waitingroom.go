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

// AdmitAllResp — «Hammasini kiritish» natijasi.
//
// # Nega shunchaki 204 emas
//
// Amal QISMAN bajarilishi mumkin: ro'yxat o'qilgandan keyin ustoz kimnidir
// alohida admit/reject qilib ulgurishi mumkin (atomik `TransitionFromPending`
// uni o'tkazib yuboradi), yoki bitta guest uchun token yaratish uzilishi
// mumkin. Bitta xato butun amalni bekor qilmaydi — "hammasi yoki hech nima"
// bu yerda foydalanuvchiga zarar: 9 kishi kirmay qolgandan ko'ra 9 kishi
// kirgani yaxshi. Shuning uchun natija SON bilan qaytadi va klient aniq
// nima bo'lganini ko'rsata oladi.
type AdmitAllResp struct {
	// Total — amal boshlanganda kutayotganlar soni.
	Total int `json:"total"`
	// Admitted — haqiqatan kiritilganlar soni.
	Admitted int `json:"admitted"`
	// Failed — kiritib bo'lmaganlar (Total - Admitted): oraliqda boshqa qaror
	// chiqqan yoki token yaratilmagan.
	Failed int `json:"failed"`
}

// WaitingRoomStatusResp — guest so'rovining hozirgi holati (public polling / WS uchun).
// Status=="admitted" bo'lsa Room to'ldiriladi.
type WaitingRoomStatusResp struct {
	RequestID string     `json:"request_id"`
	Status    string     `json:"status"`
	Room      *RoomToken `json:"room,omitempty"`
}
