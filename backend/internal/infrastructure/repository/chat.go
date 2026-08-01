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

	// ListAllByLesson — darsning TO'LIQ chat tarixi, ESKIDAN YANGIGA (created_at ASC).
	//
	// [ListByLesson] dan farqi ataylab: u JONLI chat uchun (eng yangisi birinchi,
	// kursor bilan orqaga varaqlanadi), bu esa ARXIV uchun — transkript va dars
	// arxivi suhbatni boshidan oxirigacha o'qiydi. Tartibni chaqiruvchida
	// teskarilash mumkin edi, lekin u holda "oxirgi 50 ta" kursor mantig'i ham
	// birga kelardi va arxiv jimgina chala chiqardi.
	//
	// Ko'rinuvchanlik va o'chirilgan xabar filtri [ListByLesson] bilan AYNAN bir
	// xil va bir xil sababdan SQL'da qo'llanadi.
	//
	// max — himoya chegarasi (0 → repozitoriy default'i). Uzun darsning butun
	// chati bir o'qishda xotiraga keladi, shuning uchun yuqori, lekin CHEKLI.
	ListAllByLesson(ctx context.Context, lessonID, viewerIdentity string, max int) ([]*entity.ChatMessage, error)

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
