package archive

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// urlTTL — arxivdagi presigned havolalar umri (video va materiallar).
//
// `recording.downloadTTL` va `chat.ChatFileURLTTL` bilan bir xil bir soat:
// arxiv sahifasi aynan o'sha obyektlarni ko'rsatadi va uch xil muddat
// "video ishlaydi, fayl esa yo'q" degan tushunarsiz holat yasardi.
const urlTTL = time.Hour

type useCase struct {
	lessonRepo repository.LessonRepository
	chatRepo   repository.ChatRepository
	recRepo    repository.RecordingRepository
	// storage — nil bo'lishi mumkin (MinIO sozlanmagan muhit): shunda arxiv
	// havolasiz, lekin ISHLAYDI — chat va metama'lumot baribir kerak.
	storage Storage
	// retention — yozuv saqlash muddati (`expires_at` hisoblash uchun).
	// 0 → cheksiz (klientga `expires_at` qaytmaydi).
	retention time.Duration
	// telegramEnabled — Telegram arxivi yoqilganmi. Yoqilgan bo'lsa Telegramda
	// TASDIQLANMAGAN yozuv muddat bo'yicha o'chirilmaydi va unga sana
	// ko'rsatilmaydi (`entity.ServerExpiry`).
	telegramEnabled bool
	log             logger.Logger
}

func New(
	lessonRepo repository.LessonRepository,
	chatRepo repository.ChatRepository,
	recRepo repository.RecordingRepository,
	storage Storage,
	retention time.Duration,
	telegramEnabled bool,
	log logger.Logger,
) UseCase {
	return &useCase{
		lessonRepo: lessonRepo, chatRepo: chatRepo, recRepo: recRepo,
		storage: storage, retention: retention,
		telegramEnabled: telegramEnabled, log: log,
	}
}

func (uc *useCase) Get(ctx context.Context, mentorID, lessonID string) (*entity.LessonArchive, error) {
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}

	rec := uc.pickRecording(ctx, lessonID)
	// Ko'rinuvchanlik `chat.History` bilan bir xil: mentorID = host identity,
	// ya'ni ommaviy xabarlar + ustozning O'Z shaxsiy yozishmalari.
	msgs, err := uc.chatRepo.ListAllByLesson(ctx, lessonID, mentorID, 0)
	if err != nil {
		return nil, err
	}

	base := uc.timeBase(l, rec)
	chat := make([]entity.ArchiveChatMessage, 0, len(msgs))
	materials := make([]entity.ArchiveMaterial, 0)
	for _, m := range msgs {
		file := uc.signedFile(ctx, m)
		chat = append(chat, entity.ArchiveChatMessage{
			ID:             m.ID,
			SenderIdentity: m.SenderIdentity,
			SenderName:     m.SenderName,
			Body:           m.Body,
			ToIdentity:     m.ToIdentity,
			File:           file,
			CreatedAt:      m.CreatedAt,
			OffsetSec:      offsetSec(base, m.CreatedAt),
		})
		if file != nil {
			materials = append(materials, entity.ArchiveMaterial{
				Name: file.Name, Size: file.Size, Mime: file.Mime,
				URL: file.URL, CreatedAt: m.CreatedAt,
			})
		}
	}

	return &entity.LessonArchive{
		Lesson:    l,
		Recording: uc.archiveRecording(ctx, rec),
		Chat:      chat,
		Materials: materials,
	}, nil
}

// pickRecording — dars uchun ko'rsatiladigan yozuvni tanlaydi.
//
// Bitta darsda bir nechta yozuv bo'lishi mumkin (uzilishdan keyin qayta
// boshlash, muvaffaqiyatsiz egress). Arxivda BITTA pleyer bor, shuning uchun:
//  1. eng yangi `ready` — ijro etsa bo'ladigan yagona holat;
//  2. bo'lmasa eng yangi `expired` — Telegramdan qaytarib olish oqimi (PRODUCT.md)
//     aynan shu holatga tayanadi, ya'ni uni yashirish klientdan tiklash
//     imkonini yashirardi;
//  3. bo'lmasa eng yangisi (recording/processing/failed) — ustoz "yozuv
//     tayyorlanmoqda" yoki "yozuv olinmadi" deb ko'rishi kerak, "yozuv yo'q"
//     deb emas.
//
// Yozuvlarni o'qish yiqilsa xato QAYTARILMAYDI (shuning uchun `error` ham yo'q):
// arxivning qolgan qismi — chat va materiallar — hamon foydali va butun
// sahifani 500 bilan yopish zarari kattaroq. Klient uchun bu "yozuv yo'q"
// bilan bir xil ko'rinadi, sabab esa logda qoladi.
func (uc *useCase) pickRecording(ctx context.Context, lessonID string) *entity.Recording {
	recs, err := uc.recRepo.ListByLesson(ctx, lessonID)
	if err != nil {
		uc.log.Warn(ctx, "archive: recordings unavailable",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil
	}
	var best *entity.Recording
	for _, r := range recs {
		if best == nil || recRank(r) > recRank(best) ||
			(recRank(r) == recRank(best) && r.StartedAt.After(best.StartedAt)) {
			best = r
		}
	}
	return best
}

// recRank — [useCase.pickRecording] dagi ustuvorlik (kattasi yutadi).
func recRank(r *entity.Recording) int {
	switch r.Status {
	case entity.RecordingStatusReady:
		return 5
	// `archived`/`restoring` — video YO'QOLMAGAN, Telegramda turibdi va tiklash
	// mumkin. Ular ilgari `default` (failed) ga tushardi: bitta darsda bir necha
	// yozuv bo'lsa (uzilishdan keyin qayta boshlash) YIQILGAN yozuv Telegramdagi
	// tirik nusxadan ustun chiqib, klientda "tiklash" tugmasi umuman
	// ko'rinmasdi — ya'ni mavjud video foydalanuvchi uchun yo'qolgan bo'lardi.
	case entity.RecordingStatusRestoring:
		return 4
	case entity.RecordingStatusArchived:
		return 3
	case entity.RecordingStatusExpired:
		return 2
	case entity.RecordingStatusRecording, entity.RecordingStatusProcessing:
		return 1
	default: // failed
		return 0
	}
}

// archiveRecording — yozuvni klient shakliga o'giradi va havolani imzolaydi.
func (uc *useCase) archiveRecording(ctx context.Context, rec *entity.Recording) *entity.ArchiveRecording {
	if rec == nil {
		return nil
	}
	out := &entity.ArchiveRecording{
		ID:          rec.ID,
		Status:      rec.Status,
		DurationSec: rec.DurationSec,
		SizeBytes:   rec.SizeBytes,
		ExpiresAt:   uc.expiresAt(rec),
		// Telegram havolasi holatдан QAT'I NAZAR (hatto `expired` da ham):
		// Telegramdagi nusxa abadiy, server nusxasi o'chsa ham ochiladi.
		TelegramURL: telegramArchiveURL(rec),
	}
	// Havola FAQAT `ready` da. `expired` da fayl MinIO'dan o'chirilgan va
	// presigned havola 404 beradigan "ishlaydigan" URL bo'lardi — klient uni
	// pleyerga berib, tushunarsiz xato ko'rsatardi.
	if rec.Status != entity.RecordingStatusReady || uc.storage == nil || rec.ObjectKey == "" {
		return out
	}
	url, err := uc.storage.PresignedURL(ctx, rec.ObjectKey, urlTTL)
	if err != nil {
		// Imzolash yiqilsa `url: null` — qolgan arxiv baribir qaytadi.
		uc.log.Warn(ctx, "archive: recording presign failed",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		return out
	}
	out.URL = &url
	return out
}

// telegramArchiveURL — arxiv guruhidagi videoga `t.me/c/<id>/<msg>` havolasi.
//
// Faqat SUPERGURUH/kanal uchun ishlaydi: ularning chat ID'si `-100XXXXXXXXXX`
// ko'rinishida bo'ladi va `t.me/c/` uchun boshidagi `-100` olib tashlanadi.
// Oddiy guruhlar (kichik manfiy ID) uchun bunday havola yo'q → nil.
// Havola faqat guruh A'ZOLARIGA ochiladi (private guruh).
func telegramArchiveURL(rec *entity.Recording) *string {
	if rec.TelegramMessageID == nil || rec.TelegramChatID == nil {
		return nil
	}
	chat := *rec.TelegramChatID
	if chat >= 0 {
		return nil
	}
	digits := strconv.FormatInt(-chat, 10)
	if !strings.HasPrefix(digits, "100") || len(digits) <= 3 {
		return nil
	}
	url := fmt.Sprintf("https://t.me/c/%s/%d", digits[3:], *rec.TelegramMessageID)
	return &url
}

// expiresAt — yozuv qachon serverdan o'chadi (`recording.withExpiry` bilan
// bir xil hisob: `COALESCE(ended_at, created_at) + retention`).
func (uc *useCase) expiresAt(rec *entity.Recording) *time.Time {
	// Qoida `entity.ServerExpiry` da — `recording` usecase ham shuni chaqiradi.
	return entity.ServerExpiry(rec, uc.retention, uc.telegramEnabled)
}

// signedFile — chat xabaridagi faylga vaqtinchalik havola qo'yadi (nusxada).
//
// Nusxa MUHIM: repozitoriy qaytargan struct'ni joyida o'zgartirish keshlangan
// yoki boshqa joyda ishlatilayotgan obyektga havola yozib qo'yish xavfini
// tug'diradi. `URL` esa vaqtinchalik va DB'ga hech qachon tegmasligi kerak.
func (uc *useCase) signedFile(ctx context.Context, m *entity.ChatMessage) *entity.ChatFile {
	if m.File == nil {
		return nil
	}
	f := *m.File
	if uc.storage == nil || f.Key == "" {
		return &f
	}
	url, err := uc.storage.PresignedURL(ctx, f.Key, urlTTL)
	if err != nil {
		uc.log.Warn(ctx, "archive: file presign failed",
			logger.String("message_id", m.ID), logger.SafeString("err", err.Error()))
		return &f
	}
	f.URL = url
	f.ExpiresInS = int(urlTTL.Seconds())
	return &f
}

// timeBase — `offset_sec` hisobining nol nuqtasi.
//
// ## ⭐ NEGA YOZUV BIRINCHI, DARS EMAS
// `offset_sec` ning yagona vazifasi — pleyerni to'g'ri lahzaga sakratish.
// Demak nol nuqta VIDEO FAYLINING boshi bo'lishi shart. Dars boshlanishi esa
// undan oldinroq: egress birinchi trek e'lon qilinganda ishga tushadi va
// transkod fayl boshidagi o'lik qismni kesadi ([entity.Recording.PlaybackZero]).
//
// Ilgari bu yerda `lesson.started_at` birinchi turardi va shu sababli barcha
// xabarlar o'nlab soniyaga siljigan holda ko'rsatilardi (2026-08-03 da yozuv
// boshini kesish qo'shilganda farq yanada kattalashdi).
//
// Yozuv yo'q bo'lsagina darsning boshlanish vaqtiga tushamiz: video ham yo'q,
// ya'ni sakraydigan joy yo'q va `offset_sec` faqat «dars boshidan qancha
// o'tgan» degan ma'noni bildiradi. Ikkalasi ham bo'lmasa nol vaqt qaytadi va
// `offsetSec` hammasini 0 qiladi — noto'g'ri sakrashdan ko'ra sakramaslik
// yaxshi.
func (uc *useCase) timeBase(l *entity.Lesson, rec *entity.Recording) time.Time {
	if zero, ok := rec.PlaybackZero(); ok {
		return zero
	}
	if l.StartedAt != nil {
		return *l.StartedAt
	}
	return time.Time{}
}

// offsetSec — xabarning dars boshidan siljishi (soniya), hech qachon manfiy emas.
//
// Manfiy qiymat REAL holat: dars boshlanishidan oldin (kutish xonasida yoki
// oldingi seansda) yozilgan xabar. Uni klientga manfiy berib yuborish pleyerni
// yaroqsiz vaqtga sakratardi, shuning uchun 0 ga qisiladi.
func offsetSec(base, at time.Time) int {
	if base.IsZero() {
		return 0
	}
	d := at.Sub(base)
	if d <= 0 {
		return 0
	}
	return int(d.Seconds())
}
