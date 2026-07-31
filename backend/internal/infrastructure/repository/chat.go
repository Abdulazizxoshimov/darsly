package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type ChatRepository interface {
	Create(ctx context.Context, m *entity.ChatMessage) error
	// ListByLesson — eng yangi xabarlarni qaytaradi (created_at DESC). before!=nil bo'lsa
	// faqat undan eski xabarlar (kursor-paginatsiya). limit sahifa hajmi.
	//
	// viewerIdentity — KO'RINUVCHANLIK filtri va u repozitoriyda (SQL'da) qo'llanadi,
	// usecase'da emas: filtrni yuqori qatlamga chiqarish "hammasini o'qib, keyin
	// ortiqchasini tashlash" degani bo'lardi — ya'ni begona shaxsiy xabarlar
	// baribir jarayon xotirasiga kelardi va bitta e'tiborsizlik ularni tashqariga
	// chiqarardi. Bo'sh satr → faqat ommaviy xabarlar.
	//
	// O'chirilgan (moderatsiya qilingan) xabarlar HECH QACHON qaytmaydi — bu ham
	// SQL'da, xuddi shu sababdan.
	ListByLesson(ctx context.Context, lessonID, viewerIdentity string, before *time.Time, limit int) ([]*entity.ChatMessage, error)

	// SoftDelete — mentor moderatsiyasi (№6): xabarni tarixdan yashiradi.
	//
	// ATOMIK: `WHERE id=… AND lesson_id=… AND deleted_at IS NULL` + RETURNING.
	//   · `lesson_id` shartda — begona darsning xabarini o'chirib bo'lmasin
	//     (egalik tekshiruvi dars bo'yicha qilinadi, xabar bo'yicha emas);
	//   · `deleted_at IS NULL` — takroriy o'chirish (ikki marta bosish yoki ikki
	//     qurilma) ikkinchi marta hodisa tarqatmasin;
	//   · RETURNING — tarqatish uchun kerakli maydonlar (shaxsiy xabarni faqat
	//     ikki tomonga yuborish) qo'shimcha SELECT'siz keladi, ya'ni o'qish bilan
	//     yozish orasida poyga oynasi yo'q.
	//
	// Topilmasa/allaqachon o'chirilgan bo'lsa `apperr.NotFound("chat message")`.
	SoftDelete(ctx context.Context, lessonID, messageID, deletedBy string) (*entity.ChatMessage, error)
}
