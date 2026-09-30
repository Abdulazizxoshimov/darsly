package entity

import "time"

// Natija ko'rinuvchanligi rejimi (`polls.results_visibility`).
//
// Asoschi qarori (PRODUCT.md): mentor so'rovnoma YARATISHDA tanlaydi — natija
// «faqat men ko'raman» yoki «hammaga ko'rinadi». Ikkinchi holatda ham natija
// o'quvchilarga faqat mentor «E'lon qilish» tugmasini bosgach chiqadi.
const (
	// PollResultsMentorOnly — natija HECH QACHON o'quvchiga ko'rinmaydi.
	// E'lon qilib ham bo'lmaydi: rejim yaratishda tanlangan va o'zgarmas.
	PollResultsMentorOnly = "mentor_only"
	// PollResultsPublic — natijani o'quvchi ham ko'radi, lekin FAQAT
	// `ResultsPublishedAt` to'ldirilgandan keyin.
	PollResultsPublic = "public"
)

// Poll — dars ichidagi so'rovnoma/viktorina.
type Poll struct {
	ID        string     `json:"id"`
	LessonID  string     `json:"lesson_id"`
	Question  string     `json:"question"`
	Options   []string   `json:"options"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	// ResultsVisibility — "mentor_only" | "public" (yaratishda tanlanadi, o'zgarmas).
	ResultsVisibility string `json:"results_visibility"`
	// ResultsPublishedAt — mentor «E'lon qilish» bosgan vaqt (nil = e'lon qilinmagan).
	ResultsPublishedAt *time.Time `json:"results_published_at,omitempty"`
}

// ResultsVisibleTo — shu so'rovnoma natijasini KO'RSATSA bo'ladimi.
//
// Qoida bitta joyda turadi, chunki u ikki chaqiruvchida kerak (`Results` va
// `Publish`) va ikki mustaqil nusxa vaqt o'tib bir-biridan uzilib qolardi.
func (p *Poll) ResultsVisibleTo(isHost bool) bool {
	if isHost {
		return true // mentor o'z darsining natijasini DOIM ko'radi
	}
	return p.ResultsVisibility == PollResultsPublic && p.ResultsPublishedAt != nil
}

// PollResults — so'rovnoma natijalari.
type PollResults struct {
	Poll   *Poll `json:"poll"`
	Counts []int `json:"counts"` // har variant uchun ovozlar soni
	Total  int   `json:"total"`
}

type CreatePollReq struct {
	Question string   `json:"question" validate:"required,min=1,max=500"`
	Options  []string `json:"options"  validate:"required,min=2,max=10,dive,min=1,max=200"`
	// ResultsVisibility — bo'sh qoldirilsa `mentor_only` (yopiq tomon).
	// Eski klientlar bu maydonni yubormaydi va ular uchun xulq o'zgarmaydi:
	// ilgari ham natija faqat mentor tomonidan e'lon qilinardi (klient qo'lda).
	ResultsVisibility string `json:"results_visibility" validate:"omitempty,oneof=mentor_only public"`
}

// VoteReq — ovoz berish. Token — guest'ning LiveKit access token'i (room-auth).
// Token'ga `validate:"required"` qo'yilmagan: yo'q token 400 emas, 401 bo'lishi kerak
// (qarang [HandReq]); handler o'zi tekshiradi.
type VoteReq struct {
	Token       string `json:"token"`
	OptionIndex int    `json:"option_index" validate:"min=0"`
}

// PollPublishedEvent — natija e'lon qilinganda xonaga ketadigan hodisa
// (LiveKit data-channel). Natijaning O'ZI ham yuboriladi: o'quvchi shu zahoti
// ko'rsin, qo'shimcha so'rov qilmasin (300 kishilik xonada bu 300 ta so'rov).
type PollPublishedEvent struct {
	Kind    string       `json:"kind"` // doim "poll_published"
	Results *PollResults `json:"results"`
}

// NewPollPublishedEvent — [PollPublishedEvent] konstruktori.
func NewPollPublishedEvent(res *PollResults) PollPublishedEvent {
	return PollPublishedEvent{Kind: "poll_published", Results: res}
}
