package entity

import "time"

// RaisedHand — qo'l ko'targan ishtirokchi.
//
// Nega bu holat SERVERDA: avval qo'l faqat LiveKit data-channel'da uchardi, ya'ni
// hech qayerda saqlanmasdi. Oqibati: kech ulangan ustoz ko'tarilgan qo'llarni
// ko'rmasdi, qayta ulanish hammasini o'chirardi, navbat tartibi yo'q edi va ustoz
// birovning qo'lini tushira olmasdi. Endi haqiqat manbai — server.
type RaisedHand struct {
	Identity string    `json:"identity"`
	Name     string    `json:"name"`
	RaisedAt time.Time `json:"raised_at"`
}

// RoomState — dars xonasining o'tkinchi holati (kech kirgan klient shu bilan tiklanadi).
type RoomState struct {
	// Hands — KO'TARILGAN VAQT bo'yicha tartiblangan (kim birinchi so'ragan bo'lsa
	// birinchi turadi). Tartib serverda hisoblanadi: klientlar mustaqil tartiblasa
	// ustoz va o'quvchi turli navbat ko'rishi mumkin edi.
	Hands []RaisedHand `json:"hands"`
	// Recording — dars hozir yozib olinyaptimi.
	//
	// Nega bu YERDA: yozuv holati ilgari faqat `GET /lessons/:id/recordings`
	// dan olinardi, u esa mentor huquqini talab qiladi — ya'ni O'QUVCHI o'zi
	// yozib olinayotganini BILA OLMASDI. Bu shunchaki qulaylik emas, maxfiylik
	// talabi: odam yozuvga tushayotganini ko'rishi kerak. Xona holati esa
	// o'quvchida allaqachon bor (room-token bilan) va u shu endpointni
	// muntazam so'raydi — qo'shimcha so'rov kerak emas.
	Recording bool `json:"recording"`
}

// ── So'rov turlari ────────────────────────────────────────────────────────────

// HandReq — o'quvchi qo'l ko'taradi/tushiradi. Token — LiveKit room-token
// (guest'da JWT yo'q; bu naqsh so'rovnomada allaqachon ishlatilgan).
type HandReq struct {
	// Token'ga `validate:"required"` QO'YILMAGAN ataylab: uning yo'qligi so'rov
	// shakli emas, AUTENTIFIKATSIYA muammosi. Bog'lash validatsiyasi 400 berardi,
	// holbuki bir xil holat `GET /state` da 401 qaytaradi — bir xil sabab uchun
	// ikki xil status klient uchun tushunarsiz bo'lardi. Handler 401 beradi.
	Token  string `json:"token"`
	Raised bool   `json:"raised"`
}

// LowerHandReq — host bitta ishtirokchining qo'lini tushiradi.
// `identity` yo'lda emas, tanada: guest identity'si ixtiyoriy satr bo'lishi mumkin.
type LowerHandReq struct {
	Identity string `json:"identity" validate:"required"`
}

// ReactionReq — emoji reaksiya (saqlanmaydi, faqat tarqatiladi).
type ReactionReq struct {
	Token string `json:"token"` // tekshiruv handler'da (401) — [HandReq] izohiga qara
	// Emoji uzunligi cheklangan: bitta emoji ko'p baytli bo'lishi mumkin
	// (masalan ❤️ = 6 bayt), lekin 16 baytdan oshsa bu emoji emas.
	Emoji string `json:"emoji" validate:"required,min=1,max=16"`
}
