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
	ListByLesson(ctx context.Context, lessonID, viewerIdentity string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
}
