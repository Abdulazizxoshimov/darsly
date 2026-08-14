package recording

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/audit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/shared"
)

const downloadTTL = time.Hour

type useCase struct {
	repo       repository.RecordingRepository
	lessonRepo repository.LessonRepository
	livekit    LiveKit
	minio      minio.Client
	s3         livekit.S3Config
	// cache — dublikat egress qulfi uchun (`startLockKey`).
	cache redis.Cache
	// retention — yozuv qancha saqlanadi (RECORDING_RETENTION_DAYS). 0 → cheksiz
	// (o'chirish o'chirilgan); bu holda `expires_at` ham qaytarilmaydi.
	retention time.Duration
	// telegram — arxivdan tiklash uchun. nil yoki o'chiq bo'lsa `archived`
	// yozuvni qaytarib bo'lmaydi va mentorga aniq sabab aytiladi.
	telegram Telegram
	// cacheTTL — Telegramdan tiklangan nusxa MinIO'da qancha turadi
	// (RECORDING_CACHE_TTL_HOURS). 0 → default 24 soat.
	cacheTTL time.Duration
	// localMode — client-side (telefon) yozuv rejimi (RECORDING_MODE=local).
	// true bo'lsa server egress AVTOMATIK boshlanmaydi — telefon o'zi yozib
	// yuklaydi. Qo'lda `StartRecording` (egress) baribir ishlaydi (fallback).
	localMode bool
	log       logger.Logger
}

func New(
	repo repository.RecordingRepository,
	lessonRepo repository.LessonRepository,
	lk LiveKit,
	mc minio.Client,
	s3 livekit.S3Config,
	cache redis.Cache,
	retention time.Duration,
	tgc Telegram,
	cacheTTL time.Duration,
	localMode bool,
	log logger.Logger,
) UseCase {
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}
	return &useCase{
		repo: repo, lessonRepo: lessonRepo, livekit: lk, minio: mc, s3: s3,
		cache: cache, retention: retention, telegram: tgc, cacheTTL: cacheTTL,
		localMode: localMode, log: log,
	}
}

// telegramEnabled — arxiv integratsiyasi ishlaydimi.
func (uc *useCase) telegramEnabled() bool { return uc.telegram != nil && uc.telegram.Enabled() }

// withExpiry — `ready` yozuvga hisoblangan `expires_at` ni qo'yadi.
//
// Klient («X kundan keyin o'chadi») shu maydonga tayanadi. Hisoblanadigan
// bo'lishining sababi `entity.Recording.ExpiresAt` izohida.
func (uc *useCase) withExpiry(rec *entity.Recording) *entity.Recording {
	if rec == nil {
		return rec
	}
	// Qoida `entity.ServerExpiry` da (sof, test ostida) — arxiv usecase ham
	// AYNI o'shani chaqiradi, shuning uchun ikki joy ajralib keta olmaydi.
	rec.ExpiresAt = entity.ServerExpiry(rec, uc.retention, uc.telegramEnabled())
	return rec
}

// startLockKey — bitta dars uchun bir vaqtda faqat bitta egress boshlanishini
// ta'minlovchi qulf.
func startLockKey(lessonID string) string { return "rec:lock:" + lessonID }

// startLockTTL — qulf muddati. Egress boshlash `lkCtx` bilan cheklangan
// (~25s), shuning uchun undan biroz uzun. Qisqa bo'lsa qulf ish tugamasdan
// bo'shab poyga qaytadi; uzun bo'lsa nosozlikdan keyin yozuv kechikadi.
const startLockTTL = 45 * time.Second

// activeFor — shu dars uchun hozir yozilayotgan yozuvni qaytaradi (bo'lmasa nil).
func (uc *useCase) activeFor(ctx context.Context, lessonID string) *entity.Recording {
	recs, err := uc.repo.ListByLesson(ctx, lessonID)
	if err != nil {
		// Xato holida "faol yozuv yo'q" deb HISOBLAMAYMIZ: aks holda DB uzilganda
		// har qayta ulanishda yangi egress boshlanib, dublikat yozuvlar to'planardi.
		// Chaqiruvchi buni "aniqlab bo'lmadi" deb qabul qiladi va yangi yozuv
		// boshlamaydi — yo'qolgan yozuvdan ko'ra takrorlanmagani xavfsizroq.
		uc.log.Warn(ctx, "recording.activeFor: list failed", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return &entity.Recording{ID: "", Status: entity.RecordingStatusRecording}
	}
	for _, r := range recs {
		if r.Status == entity.RecordingStatusRecording {
			return r
		}
	}
	return nil
}

// EnsureRecording — majburiy yozib olishning kirish nuqtasi. Izohni
// [UseCase.EnsureRecording] da qara.
func (uc *useCase) EnsureRecording(ctx context.Context, lessonID string) error {
	if uc.localMode {
		// Client-side rejim: yozuvni TELEFON o'zi qiladi (to'liq sifat, lokal),
		// server egress'ini AVTOMATIK boshlamaymiz. Telefon `LocalStart` bilan
		// yozuv qatorini yaratadi. (Qo'lda `StartRecording` egress fallbacki
		// baribir ochiq qoladi.)
		return nil
	}
	if !uc.livekit.Enabled() {
		// Video servis sozlanmagan — bu dev muhitida odatiy holat, xato emas.
		return nil
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return err
	}
	if l.Status != entity.LessonStatusLive {
		// Poyga: dars biz kelgunimizcha yakunlangan bo'lishi mumkin.
		return nil
	}
	if !l.IsRecordingEnabled {
		// Ustoz shu dars uchun yozib olishni ATAYLAB o'chirgan.
		//
		// Qoida: yozib olish **default yoniq** (dars yaratishda belgi o'rnatilgan
		// holda keladi), lekin majburiy emas — ustoz uni o'chira oladi. Avtomatik
		// boshlash shu tanlovni bekor qilmasligi kerak, aks holda "o'chirdim,
		// baribir yozildi" degan holat bo'lardi va bu maxfiylik buzilishi.
		uc.log.Info(ctx, "recording.Ensure: dars uchun o'chirilgan — boshlanmaydi",
			logger.String("lesson_id", lessonID))
		return nil
	}
	if active := uc.activeFor(ctx, lessonID); active != nil {
		uc.log.Info(ctx, "recording.Ensure: allaqachon yozilmoqda", logger.String("lesson_id", lessonID))
		return nil
	}

	// ⭐ DUBLIKAT EGRESS QULFI (TOCTOU).
	//
	// Yuqoridagi `activeFor` tekshiruvi va quyidagi `startEgress` orasida oyna
	// bor. Ustoz kamera va mikrofonni deyarli bir vaqtda yoqsa LiveKit IKKI
	// `track_published` webhook'ini yuboradi; ikkalasi ham "faol yozuv yo'q"
	// deb ko'radi va IKKI parallel egress boshlanadi — 2× CPU/disk va bitta
	// darsdan ikkita fayl. 4 yadroli serverda bu sezilarli.
	//
	// Qulf `SetNX` (atomik) bilan: faqat birinchi chaqiruv o'tadi.
	if uc.cache != nil {
		ok, err := uc.cache.SetNX(ctx, startLockKey(lessonID), "1", startLockTTL)
		if err != nil {
			// Redis yetib bo'lmadi — yozuvni BLOKLAMAYMIZ. Dublikat egress
			// noqulaylik, yozuvning umuman yo'qligi esa ma'lumot yo'qotish.
			uc.log.Warn(ctx, "recording.Ensure: qulfni olib bo'lmadi — qulfsiz davom etamiz",
				logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		} else if !ok {
			uc.log.Info(ctx, "recording.Ensure: boshqa chaqiruv allaqachon boshlamoqda",
				logger.String("lesson_id", lessonID))
			return nil
		}
	}

	rec, err := uc.startEgress(ctx, lessonID)
	if err != nil {
		// Qulfni darhol bo'shatamiz: aks holda keyingi `track_published`
		// urinishi TTL tugagunicha bekorga rad etilib, yozuv butunlay
		// boshlanmay qolishi mumkin edi.
		if uc.cache != nil {
			_ = uc.cache.Del(ctx, startLockKey(lessonID))
		}
		return err
	}
	// Muvaffaqiyatda qulf ATAYLAB bo'shatilmaydi: endi DB'da `recording`
	// holatidagi qator bor va keyingi chaqiruvlarni `activeFor` to'xtatadi.
	// Qulf TTL bilan o'zi yo'qoladi.
	uc.log.Info(ctx, "recording.Ensure: avtomatik boshlandi",
		logger.String("lesson_id", lessonID), logger.String("recording_id", rec.ID))
	return nil
}

// IsRecording — izohni [UseCase.IsRecording] da qara.
func (uc *useCase) IsRecording(ctx context.Context, lessonID string) bool {
	return uc.activeFor(ctx, lessonID) != nil
}

// EnsureForRoom — LiveKit webhook'idan kelgan xona nomi bo'yicha yozuvni boshlaydi.
//
// Webhook faqat xona NOMINI beradi, dars ID sini emas. Begona nom (bizniki
// bo'lmagan xona) jimgina e'tiborsiz qoldiriladi — xato emas.
func (uc *useCase) EnsureForRoom(ctx context.Context, roomName string) error {
	lessonID, ok := shared.LessonIDFromRoom(roomName)
	if !ok {
		return nil
	}
	return uc.EnsureRecording(ctx, lessonID)
}

// StopActiveForLesson — dars yakunlanganda. Izohni [UseCase] da qara.
func (uc *useCase) StopActiveForLesson(ctx context.Context, lessonID string) error {
	if !uc.livekit.Enabled() {
		return nil
	}
	recs, err := uc.repo.ListByLesson(ctx, lessonID)
	if err != nil {
		return err
	}
	for _, r := range recs {
		if r.Status != entity.RecordingStatusRecording {
			continue
		}
		if err := uc.livekit.StopRecording(ctx, r.EgressID); err != nil {
			// To'xtatib bo'lmadi — lekin xona baribir o'chiriladi va egress
			// o'zi tugaydi; webhook yakuniy holatni qo'yadi. Shuning uchun
			// bu xato darsni yakunlashni BLOKLAMAYDI.
			uc.log.Warn(ctx, "recording.StopActive: egress stop failed",
				logger.String("egress_id", r.EgressID), logger.SafeString("err", err.Error()))
			continue
		}
		if err := uc.repo.UpdateStatus(ctx, r.ID, entity.RecordingStatusProcessing); err != nil {
			uc.log.Warn(ctx, "recording.StopActive: status update failed",
				logger.String("recording_id", r.ID), logger.SafeString("err", err.Error()))
		}
	}
	return nil
}

func (uc *useCase) StartRecording(ctx context.Context, mentorID, lessonID string) (*entity.Recording, error) {
	// Avval egalik tekshiriladi — video servis o'chirilgan bo'lsa ham begona dars
	// yozuvi 403 bo'lib qolsin (500 bilan niqoblanmasin).
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	if l.Status != entity.LessonStatusLive {
		return nil, apperr.BadRequest("lesson is not live")
	}

	rec, err := uc.startEgress(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	audit.Record(ctx, uc.log, "recording.start", mentorID,
		logger.String("recording_id", rec.ID), logger.String("lesson_id", lessonID), logger.String("egress_id", rec.EgressID))
	return rec, nil
}

// startEgress — yozib olishning O'ZAGI: egress boshlanadi va yozuv DB'ga yoziladi.
//
// Qo'lda boshlash ([StartRecording]) va majburiy avtomatik boshlash
// ([EnsureRecording]) AYNAN shu koddan foydalanadi. Nusxa ko'chirilganda
// kompensatsiya mantig'i (DB yiqilsa egressni to'xtatish) ikkinchi yo'lda
// unutilishi mumkin edi — bu esa serverda jimgina ishlab turgan, hech kimga
// tegishli bo'lmagan egress qoldirardi.
//
// Ruxsat/holat tekshiruvlari BU YERDA EMAS: ular chaqiruvchida, chunki ikki
// yo'lning talablari boshqa (biri mentor egaligini, ikkinchisi faqat dars
// jonliligini talab qiladi).
func (uc *useCase) startEgress(ctx context.Context, lessonID string) (*entity.Recording, error) {
	recID := uuid.NewString()
	objectKey := fmt.Sprintf("recordings/%s/%s.mp4", lessonID, recID)
	roomName := "lesson_" + lessonID

	// Egress chaqiruvi klient so'rovi hayotidan ajratilgan va cheklangan (WriteTimeout=30s'dan past),
	// shunda mentor uzilib qolsa ham yozib olish boshlanishi buzilmaydi.
	egressCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	egressID, err := uc.livekit.StartRoomRecording(egressCtx, roomName, objectKey, uc.s3)
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("egress_start").Inc()
		uc.log.Error(ctx, "recording.Start: egress failed", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not start recording: %w", err))
	}

	now := time.Now().UTC()
	rec := &entity.Recording{
		ID:        recID,
		LessonID:  lessonID,
		EgressID:  egressID,
		ObjectKey: objectKey,
		Status:    entity.RecordingStatusRecording,
		StartedAt: now,
		CreatedAt: now,
	}
	if err := uc.repo.Create(ctx, rec); err != nil {
		uc.log.Error(ctx, "recording.Start: db error", logger.SafeString("err", err.Error()))
		// Kompensatsiya: DB yozib bo'lmadi — egressni to'xtatamiz, aks holda orphan yozuv qoladi.
		if stopErr := uc.livekit.StopRecording(egressCtx, egressID); stopErr != nil {
			uc.log.Error(ctx, "recording.Start: orphan egress stop failed", logger.String("egress_id", egressID), logger.SafeString("err", stopErr.Error()))
		}
		return nil, err
	}

	metrics.RecordingsStarted.Inc()
	return rec, nil
}

// ─── Client-side (lokal) yozuv ─────────────────────────────────────────────

// LocalStart — telefon lokal yozuvni boshlaganda yozuv qatorini yaratadi.
func (uc *useCase) LocalStart(ctx context.Context, mentorID, lessonID string) (*entity.Recording, error) {
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}
	if l.Status != entity.LessonStatusLive {
		return nil, apperr.BadRequest("dars jonli emas")
	}
	// Idempotent: shu dars uchun allaqachon faol yozuv bo'lsa o'shani qaytar
	// (telefon qayta ulanib LocalStart'ni takror chaqirsa dublikat bo'lmasin).
	if active := uc.activeFor(ctx, lessonID); active != nil {
		return active, nil
	}
	recID := uuid.NewString()
	now := time.Now().UTC()
	rec := &entity.Recording{
		ID:       recID,
		LessonID: lessonID,
		// Sintetik egress_id: NOT NULL+UNIQUE cheklovni qondiradi va mavjud
		// egress-kalitли metodlar (MarkReady/EnqueueTranscode/EnqueueTelegram)
		// migratsiyasiz qayta ishlatiladi. `local:` prefiksi haqiqiy egress
		// ID'laridan (`EG_...`) ajratib turadi.
		EgressID:  "local:" + recID,
		ObjectKey: fmt.Sprintf("recordings/%s/%s.mp4", lessonID, recID),
		Status:    entity.RecordingStatusRecording,
		StartedAt: now,
		CreatedAt: now,
	}
	if err := uc.repo.Create(ctx, rec); err != nil {
		uc.log.Error(ctx, "recording.LocalStart: db error", logger.SafeString("err", err.Error()))
		return nil, err
	}
	metrics.RecordingsStarted.Inc()
	uc.log.Info(ctx, "recording.LocalStart: telefon lokal yozuvi boshlandi",
		logger.String("lesson_id", lessonID), logger.String("recording_id", recID))
	return rec, nil
}

// LocalUploadURL — telefon faylni to'g'ridan MinIO'ga PUT qilishi uchun havola.
func (uc *useCase) LocalUploadURL(ctx context.Context, mentorID, recordingID string) (string, error) {
	rec, err := uc.owned(ctx, mentorID, recordingID)
	if err != nil {
		return "", err
	}
	u, err := uc.minio.PresignedPutURL(ctx, rec.ObjectKey, downloadTTL)
	if err != nil {
		uc.log.Error(ctx, "recording.LocalUploadURL: presign put failed",
			logger.String("recording_id", recordingID), logger.SafeString("err", err.Error()))
		return "", apperr.Internal(err)
	}
	return u, nil
}

// LocalComplete — telefon yuklab bo'lgach yozuvni tayyor qiladi + navbatga qo'yadi.
func (uc *useCase) LocalComplete(ctx context.Context, mentorID, recordingID string, durationSec int, endedAt time.Time) error {
	rec, err := uc.owned(ctx, mentorID, recordingID)
	if err != nil {
		return err
	}
	// ⭐ TASDIQLASH: fayl haqiqatan MinIO'da bormi va o'lchamи. Telefon
	// "yukladim" desa ham, ishonch server tekshiruvidan keladi — aks holda
	// bo'sh/yo'q faylni "tayyor" deb belgilab, retention server nusxasini
	// o'chirishга ruxsat berardi (ma'lumot yo'qolishi).
	size, err := uc.minio.Stat(ctx, rec.ObjectKey)
	if err != nil {
		uc.log.Warn(ctx, "recording.LocalComplete: fayl MinIO'da topilmadi",
			logger.String("recording_id", recordingID), logger.SafeString("err", err.Error()))
		return apperr.BadRequest("yozuv fayli topilmadi — qayta yuklang")
	}
	if endedAt.IsZero() {
		endedAt = time.Now().UTC()
	}
	// Egress `completed` shoxi bilan AYNAN bir xil (sintetik egress_id bilan).
	if err := uc.repo.MarkReady(ctx, rec.EgressID, rec.ObjectKey, durationSec, size, endedAt); err != nil {
		uc.log.Error(ctx, "recording.LocalComplete: mark ready failed", logger.SafeString("err", err.Error()))
		return err
	}
	if err := uc.repo.EnqueueTranscode(ctx, rec.EgressID); err != nil {
		uc.log.Warn(ctx, "recording.LocalComplete: enqueue transcode failed",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
	}
	if uc.telegramEnabled() {
		if err := uc.repo.EnqueueTelegram(ctx, rec.EgressID); err != nil {
			uc.log.Warn(ctx, "recording.LocalComplete: enqueue telegram failed",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
	uc.log.Info(ctx, "recording.LocalComplete: yozuv tayyor",
		logger.String("recording_id", rec.ID), logger.String("size_bytes", fmt.Sprint(size)))
	return nil
}

func (uc *useCase) StopRecording(ctx context.Context, mentorID, recordingID string) error {
	// Yaroqsiz UUID DB'ga yetmasin (22P02 → 500). lessonID'ni OwnedLesson tekshiradi.
	if err := shared.ValidateID(recordingID, "recording"); err != nil {
		return err
	}
	rec, err := uc.repo.GetByID(ctx, recordingID)
	if err != nil {
		return err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, rec.LessonID); err != nil {
		return err
	}
	if rec.Status != entity.RecordingStatusRecording {
		return apperr.BadRequest("recording is not active")
	}

	if err := uc.livekit.StopRecording(ctx, rec.EgressID); err != nil {
		uc.log.Error(ctx, "recording.Stop: egress stop failed", logger.String("egress_id", rec.EgressID), logger.SafeString("err", err.Error()))
		return apperr.Internal(fmt.Errorf("could not stop recording: %w", err))
	}
	// Yakuniy holat webhook orqali keladi; hozircha "processing".
	if err := uc.repo.UpdateStatus(ctx, rec.ID, entity.RecordingStatusProcessing); err != nil {
		return err
	}
	audit.Record(ctx, uc.log, "recording.stop", mentorID, logger.String("recording_id", rec.ID))
	return nil
}

func (uc *useCase) ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Recording, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	recs, err := uc.repo.ListByLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	for _, rec := range recs {
		uc.withExpiry(rec)
	}
	return recs, nil
}

// Get — bitta yozuv (egalik tekshiruvi bilan). Izohni [UseCase.Get] da qara.
func (uc *useCase) Get(ctx context.Context, mentorID, recordingID string) (*entity.Recording, error) {
	rec, err := uc.owned(ctx, mentorID, recordingID)
	if err != nil {
		return nil, err
	}
	return uc.withExpiry(rec), nil
}

// owned — ID validatsiyasi + yozuvni o'qish + dars egaligini tekshirish.
//
// Uch metod (`Get`, `Restore`, `DownloadURL`) aynan shu uchlikni bajaradi.
// Nusxalanganda ulardan birida egalik tekshiruvi tushib qolishi mumkin edi —
// bu esa begona dars yozuvini yuklab olish (IDOR) degani.
func (uc *useCase) owned(ctx context.Context, mentorID, recordingID string) (*entity.Recording, error) {
	if err := shared.ValidateID(recordingID, "recording"); err != nil {
		return nil, err
	}
	rec, err := uc.repo.GetByID(ctx, recordingID)
	if err != nil {
		return nil, err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, rec.LessonID); err != nil {
		return nil, err
	}
	return rec, nil
}

// Restore — arxivdan qaytarib olishni boshlaydi. Izohni [UseCase.Restore] da qara.
func (uc *useCase) Restore(ctx context.Context, mentorID, recordingID string) (*entity.RecordingRestore, error) {
	rec, err := uc.owned(ctx, mentorID, recordingID)
	if err != nil {
		return nil, err
	}
	switch rec.Status {
	case entity.RecordingStatusReady:
		// Allaqachon serverda — tiklashning hojati yo'q. Xato emas: klient
		// «tiklash» tugmasini eski holat bilan bosgan bo'lishi mumkin.
		return &entity.RecordingRestore{Status: entity.RecordingStatusReady}, nil
	case entity.RecordingStatusRestoring:
		// Ish allaqachon ketmoqda (idempotentlik).
		return &entity.RecordingRestore{Status: entity.RecordingStatusRestoring, PollAfterS: restorePollAfterS}, nil
	case entity.RecordingStatusArchived:
		// asosiy yo'l — pastda
	default:
		return nil, apperr.BadRequest("recording cannot be restored")
	}
	if !uc.telegramEnabled() {
		return nil, apperr.BadRequest("telegram archive is not configured")
	}
	if rec.TelegramFileID == nil || *rec.TelegramFileID == "" {
		return nil, apperr.BadRequest("recording has no telegram copy")
	}

	claimed, err := uc.repo.ClaimRestore(ctx, rec.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		// Poyga: boshqa so'rov (yoki instans) ulgurdi. Bu ham muvaffaqiyat —
		// klient uchun natija bir xil.
		return &entity.RecordingRestore{Status: entity.RecordingStatusRestoring, PollAfterS: restorePollAfterS}, nil
	}
	audit.Record(ctx, uc.log, "recording.restore", mentorID, logger.String("recording_id", rec.ID))
	return &entity.RecordingRestore{Status: entity.RecordingStatusRestoring, PollAfterS: restorePollAfterS}, nil
}

// restorePollAfterS — klient qancha kutib qayta so'rasin.
//
// 5 soniya: tiklash 30-60 s davom etadi, ya'ni ~10 so'rov. Tezroq poll
// qilish serverga foydasiz yuk, sekinroq esa "tayyor bo'ldi, lekin UI hali
// eski" degan noqulaylik.
const restorePollAfterS = 5

// RunRestore — Telegram → vaqtinchalik fayl → MinIO. Izohni [UseCase.RunRestore] da.
func (uc *useCase) RunRestore(ctx context.Context, recordingID string) error {
	rec, err := uc.repo.GetByID(ctx, recordingID)
	if err != nil {
		return err
	}
	if rec.Status != entity.RecordingStatusRestoring {
		// Boshqa ishchi tugatgan yoki bekor qilingan — qilinadigan ish yo'q.
		return nil
	}
	if !uc.telegramEnabled() || rec.TelegramFileID == nil {
		uc.failRestore(ctx, rec.ID, "telegram archive is not available")
		return apperr.BadRequest("telegram archive is not available")
	}

	if err := uc.restoreObject(ctx, rec); err != nil {
		uc.log.Error(ctx, "recording.Restore: tiklab bo'lmadi",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		uc.failRestore(ctx, rec.ID, err.Error())
		return err
	}

	cachedUntil := time.Now().UTC().Add(uc.cacheTTL)
	if err := uc.repo.FinishRestore(ctx, rec.ID, rec.ObjectKey, cachedUntil); err != nil {
		return err
	}
	uc.log.Info(ctx, "recording restored from telegram",
		logger.String("recording_id", rec.ID),
		logger.String("cached_until", cachedUntil.Format(time.RFC3339)))
	return nil
}

// restoreObject — faylni Telegramdan olib MinIO'ga qo'yadi.
func (uc *useCase) restoreObject(ctx context.Context, rec *entity.Recording) error {
	f, err := uc.telegram.GetFile(ctx, *rec.TelegramFileID)
	if err != nil {
		return fmt.Errorf("getFile: %w", err)
	}
	dir, err := os.MkdirTemp("", "darsly-restore-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	// Katta fayl (1 GB gacha) diskda qolib ketmasin — hatto xato yo'lida ham.
	defer os.RemoveAll(dir)

	local := filepath.Join(dir, "restored.mp4")
	if err := uc.telegram.DownloadFile(ctx, f, local); err != nil {
		return fmt.Errorf("download: %w", err)
	}
	st, err := os.Stat(local)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	fh, err := os.Open(local)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer fh.Close()

	// AYNI `object_key` ga qaytariladi: yozuvning ichki manzili o'zgarmaydi va
	// tarixdagi havolalar (agar keshda bo'lsa) ishlayveradi.
	if _, err := uc.minio.Upload(ctx, rec.ObjectKey, "video/mp4", fh, st.Size()); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	return nil
}

// failRestore — holatni `archived` ga qaytaradi (Telegramdagi nusxa joyida).
//
// Kontekst BEKOR bo'lgan bo'lishi mumkin (klient uzildi, shutdown), shuning
// uchun yozuv alohida kontekstda — aks holda yozuv abadiy `restoring` da
// qotib qolardi va mentor qayta urina olmasdi.
func (uc *useCase) failRestore(ctx context.Context, id, msg string) {
	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if err := uc.repo.FailRestore(fctx, id, truncate(msg, 500)); err != nil {
		uc.log.Error(fctx, "recording.Restore: holatni qaytarib bo'lmadi",
			logger.String("recording_id", id), logger.SafeString("err", err.Error()))
	}
}

// truncate — xato matnini DB ustuni va UI uchun qisqartiradi.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func (uc *useCase) DownloadURL(ctx context.Context, mentorID, recordingID string) (*entity.RecordingDownload, error) {
	rec, err := uc.owned(ctx, mentorID, recordingID)
	if err != nil {
		return nil, err
	}
	// Serverda fayl yo'q — presigned havola 404 beradigan "ishlaydigan" URL
	// qaytarish o'rniga sababni aniq aytamiz.
	switch rec.Status {
	case entity.RecordingStatusExpired:
		return nil, apperr.BadRequest("recording has expired and was deleted")
	case entity.RecordingStatusArchived:
		// Yozuv YO'QOLMAGAN — Telegramda. Klient `POST /recordings/:id/restore`
		// ga yo'naltiriladi. Ataylab avtomatik boshlanmaydi: tiklash bir
		// necha yuz megabayt trafik va mentor buni bilib turib boshlasin.
		return nil, apperr.BadRequest("recording is archived — restore it first")
	case entity.RecordingStatusRestoring:
		return nil, apperr.BadRequest("recording is being restored — try again shortly")
	case entity.RecordingStatusReady:
		// asosiy yo'l
	default:
		return nil, apperr.BadRequest("recording is not ready yet")
	}

	url, err := uc.minio.PresignedURL(ctx, rec.ObjectKey, downloadTTL)
	if err != nil {
		uc.log.Error(ctx, "recording.Download: presign failed", logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not generate download link: %w", err))
	}
	return &entity.RecordingDownload{
		URL:         url,
		ExpiresInS:  int(downloadTTL.Seconds()),
		DurationSec: rec.DurationSec,
		SizeBytes:   rec.SizeBytes,
	}, nil
}

func (uc *useCase) HandleEgress(ctx context.Context, egressID string, completed bool, objectKey string, durationSec int, sizeBytes int64) error {
	rec, err := uc.repo.GetByEgressID(ctx, egressID)
	if err != nil {
		// Noma'lum egress (bizniki emas yoki allaqachon o'chirilgan) — retry foydasiz,
		// qabul qilamiz (webhook 200 oladi).
		uc.log.Warn(ctx, "recording.HandleEgress: unknown egress", logger.String("egress_id", egressID))
		return nil
	}
	now := time.Now().UTC()
	if !completed {
		if err := uc.repo.MarkFailed(ctx, egressID, now); err != nil {
			uc.log.Error(ctx, "recording.HandleEgress: mark failed error", logger.SafeString("err", err.Error()))
			return err // tranzient DB xatosi — LiveKit qayta yuborsin
		}
		metrics.EgressResults.WithLabelValues("failed").Inc()
		uc.log.Warn(ctx, "recording failed", logger.String("recording_id", rec.ID), logger.String("egress_id", egressID))
		return nil
	}
	key := objectKey
	if key == "" {
		key = rec.ObjectKey // fallback: so'ralgan yo'l
	}
	if err := uc.repo.MarkReady(ctx, egressID, key, durationSec, sizeBytes, now); err != nil {
		// DB tranzient bo'lishi mumkin — non-200 qaytaramiz, LiveKit hodisani qayta yuboradi
		// (aks holda tayyor yozuv abadiy "processing"da qolardi).
		uc.log.Error(ctx, "recording.HandleEgress: mark ready failed", logger.SafeString("err", err.Error()))
		return err
	}
	metrics.EgressResults.WithLabelValues("ready").Inc()
	uc.log.Info(ctx, "recording ready", logger.String("recording_id", rec.ID), logger.String("object_key", key))

	// Qayta kodlash navbatiga (CRF) — fon ishchisi oladi.
	// Xato JIM yutiladi: yozuv allaqachon tayyor va yuklab olinadi; navbatga
	// tushmagani sifatga emas, faqat hajmga ta'sir qiladi. Bu yerda `err`
	// qaytarish LiveKit'ni webhook'ni qayta yuborishga majburlardi va tayyor
	// yozuv ustidan ikkinchi marta `MarkReady` ishlardi.
	if err := uc.repo.EnqueueTranscode(ctx, egressID); err != nil {
		uc.log.Warn(ctx, "recording: enqueue transcode failed",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
	}

	// Telegram arxivi navbatiga (PRODUCT.md «Dars arxivi va Telegram saqlash»).
	//
	// Xato JIM yutiladi — transkodlash bilan bir xil sabab: yozuv allaqachon
	// tayyor, `err` qaytarish LiveKit'ni webhook'ni qayta yuborishga majburlab,
	// tayyor yozuv ustidan ikkinchi `MarkReady` ishlatardi. Navbatga tushmagan
	// yozuvni retention ham o'chirmaydi (`telegram_sent_at` bo'sh) — ya'ni
	// eng yomon holat "arxivlanmagan", "yo'qolgan" emas.
	//
	// Integratsiya o'chiq bo'lsa navbatga umuman qo'yilmaydi: aks holda hech
	// kim olmaydigan ishlar to'planib, `ClaimTelegramUpload` har tick'da
	// bo'sh aylanardi.
	if uc.telegramEnabled() {
		if err := uc.repo.EnqueueTelegram(ctx, egressID); err != nil {
			uc.log.Warn(ctx, "recording: enqueue telegram failed",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
	return nil
}
