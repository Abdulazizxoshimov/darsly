// Package archive — o'tgan darsning to'liq surati (video + chat + materiallar).
//
// PRODUCT.md «Dars arxivi va Telegram saqlash» (2026-08-01), ish №20.
//
// # Nega alohida usecase
//
// Arxiv uchta domenning o'qish-kesimi: `lesson` (egalik va boshlanish vaqti),
// `recording` (video va uning holati), `chat` (yozishma va fayllar). Uni
// mavjud domenlardan biriga tiqish o'sha domenni qolgan ikkitasiga bog'lab
// qo'yardi — masalan chat usecase'i yozuv repozitoriysini bilishi kerak
// bo'lardi va "chat" nomi endi haqiqatni ifodalamasdi.
//
// Bu paket FAQAT O'QIYDI: hech narsa yaratmaydi, o'zgartirmaydi, o'chirmaydi.
package archive

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

// Storage — presigned havola yasovchi (DIP: `minio.Client` qondiradi).
//
// Arxiv obyekt YOZMAYDI, shuning uchun interfeysda `Upload` yo'q: usecase
// o'zi so'ray olmaydigan huquqni qo'lida tutmasligi kerak.
type Storage interface {
	PresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error)
}

type UseCase interface {
	// Get — dars arxivi (faqat dars EGASI).
	//
	// Bitta so'rov, chunki klient uchun bu bitta ekran — sabab
	// `entity.LessonArchive` izohida.
	//
	// Yozuv yo'q bo'lsa `recording: null`, chat bo'sh bo'lsa `chat: []` —
	// ikkalasi ham xato EMAS: "hali yozuv tayyor emas" va "chatsiz dars"
	// mutlaqo normal holatlar va klient ularni oddiy ko'rinish sifatida
	// ko'rsatadi, xato ekrani sifatida emas.
	Get(ctx context.Context, mentorID, lessonID string) (*entity.LessonArchive, error)
}
