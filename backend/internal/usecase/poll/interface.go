package poll

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — poll usecase talab qiladigan LiveKit operatsiyasi (DIP).
// Faqat tarqatish: natija e'lon qilinganda xonaga xabar ketadi.
type LiveKit interface {
	Enabled() bool
	SendData(ctx context.Context, room string, data []byte) error
}

type UseCase interface {
	// Create — so'rovnoma yaratish. resultsVisibility: "mentor_only" (default) | "public".
	Create(ctx context.Context, mentorID, lessonID, question string, options []string, resultsVisibility string) (*entity.Poll, error)
	Close(ctx context.Context, mentorID, pollID string) (*entity.PollResults, error)
	ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Poll, error)
	// Vote — ishtirokchi ovoz beradi (voterIdentity va tokenRoom room-token'dan olinadi).
	// tokenRoom poll'ning darsiga bog'lanadi — begona darsning tokeni bilan ovoz berib bo'lmaydi.
	Vote(ctx context.Context, pollID, voterIdentity, tokenRoom string, optionIndex int) error
	// Results — so'rovnoma natijalari.
	//
	// KO'RINUVCHANLIK (№7): o'quvchi natijani faqat `results_visibility=public`
	// VA e'lon qilingan bo'lsa oladi; aks holda 403. Mentor doim oladi.
	// `viewerIdentity` mentorlikni aniqlash uchun (host token identity'si = mentorID).
	Results(ctx context.Context, pollID, viewerIdentity, tokenRoom string) (*entity.PollResults, error)
	// Publish — mentor «E'lon qilish» bosdi: natija o'quvchilarga ochiladi va
	// xonaga data-channel orqali yuboriladi. `mentor_only` so'rovnomada 400.
	Publish(ctx context.Context, mentorID, lessonID, pollID string) (*entity.PollResults, error)
}
