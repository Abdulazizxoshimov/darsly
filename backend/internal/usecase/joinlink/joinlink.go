package joinlink

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

// Passcode brute-force himoyasi (ikki darajali — izohi `Join` da).
const (
	// maxPasscodeAttempts — bitta klient (slug+IP) uchun urinishlar chegarasi.
	maxPasscodeAttempts = 5
	// maxSlugAttempts — butun dars uchun chegara. Ataylab yuqori: u faqat
	// TAQSIMLANGAN hujumni tutishi kerak, halol foydalanuvchilarni emas.
	// 30 kishilik darsda har biri 1 marta xato qilsa ham bu chegaraga yetmaydi.
	maxSlugAttempts      = 50
	passcodeFailWindow   = 10 * time.Minute
	passcodeLockCooldown = 5 * time.Minute
)

type useCase struct {
	lessonRepo  repository.LessonRepository
	userRepo    repository.UserRepository
	hasher      hasher.Hasher
	cache       redis.Cache
	room        room.UseCase
	waitingRoom waitingroom.UseCase
	// blocklist — mentor darajasidagi doimiy qora ro'yxat (№4); nil bo'lishi
	// mumkin (eski testlar), har chaqiruvdan oldin tekshiriladi.
	blocklist repository.BlocklistRepository
	log       logger.Logger
}

func New(lessonRepo repository.LessonRepository, userRepo repository.UserRepository, h hasher.Hasher, cache redis.Cache, roomUC room.UseCase, waitingUC waitingroom.UseCase, blocklist repository.BlocklistRepository, log logger.Logger) UseCase {
	return &useCase{lessonRepo: lessonRepo, userRepo: userRepo, hasher: h, cache: cache, room: roomUC, waitingRoom: waitingUC, blocklist: blocklist, log: log}
}

// Preview yakunlangan/bekor qilingan dars uchun ham 200 qaytaradi (№3):
// havola "o'lik sahifa" emas, "dars tugagan" ma'lumot sahifasiga aylanadi.
// Status javobning ichida — klient shunga qarab ko'rsatadi. Kirish (POST)
// esa bunday darsga token bermaydi (`Join` dagi lesson_ended).
func (uc *useCase) Preview(ctx context.Context, slug string) (*entity.LessonPublic, error) {
	l, err := uc.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return uc.toPublic(ctx, l), nil
}

// Join — havola orqali darsga kirish.
//
// `clientIP` — brute-force lockout'ni klient bo'yicha ajratish uchun (M7).
// Uni ATAYLAB handler beradi (`c.ClientIP()`), so'rov tanasidan olinmaydi:
// tanadagi maydonni hujumchi har urinishda o'zgartirib cheklovni chetlab
// o'tardi.
func (uc *useCase) Join(ctx context.Context, slug, clientIP string, req *entity.JoinLessonReq) (*entity.JoinLessonResp, error) {
	l, err := uc.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Yakunlangan/bekor qilingan dars (№3): havola JOIN qilmaydi — token ham,
	// kutish so'rovi ham yo'q. Xato emas, holat: klient preview ma'lumotini
	// ko'rsatib "dars tugagan" deydi (aniq status Lesson.Status'da).
	if l.Status == entity.LessonStatusEnded || l.Status == entity.LessonStatusCancelled {
		return &entity.JoinLessonResp{
			Lesson:   uc.toPublic(ctx, l),
			NextStep: entity.JoinNextStepLessonEnded,
		}, nil
	}

	// Qulflangan dars — yangi ishtirokchilar kira olmaydi.
	if l.IsLocked {
		return nil, apperr.Forbidden("lesson is locked by the host")
	}

	// Mentorning doimiy qora ro'yxati (№4): bloklangan ism token ham, kutish
	// xonasi so'rovi ham ololmaydi. Parol tekshiruvidan OLDIN — bloklangan odam
	// parol brute-force hisoblagichlarini ham band qilmasin.
	if uc.isBlockedName(ctx, l.MentorID, req.GuestName) {
		return nil, apperr.Forbidden("you have been removed from this mentor's lessons")
	}

	// Parol tekshiruvi (brute-force lockout bilan).
	//
	// # Nega lockout IKKI DARAJALI (M7)
	//
	// Avval hisoblagich ham, qulf ham FAQAT slug bo'yicha edi — ya'ni bitta odam
	// 5 marta noto'g'ri parol kiritsa BUTUN SINF 5 daqiqa kira olmasdi. Bu himoya
	// emas, arzon xizmat-rad etish quroli: dars boshlanishida bir necha so'rov
	// yuborib 30 kishilik darsni buzish mumkin edi (ko'pincha bexosdan — parolni
	// noto'g'ri eslagan bitta o'quvchi hammani qulflab qo'yardi).
	//
	// Endi:
	//   - IP darajasi (qattiq, 5 urinish) — aybdorni qulflaydi, qolganlarga tegmaydi.
	//   - Slug darajasi (bo'sh, 50 urinish) — TAQSIMLANGAN hujum (ko'p IP) uchun
	//     zaxira. Bitta halol foydalanuvchi bu chegaraga hech qachon yetmaydi.
	//
	// Faqat IP bo'yicha cheklash yetmasdi (hujumchi IP almashtiradi), faqat slug
	// bo'yicha esa yuqoridagi DoS ni beradi — shuning uchun ikkalasi ham kerak.
	if l.PasscodeHash != nil {
		if locked, _ := uc.cache.Get(ctx, lockKey(l.JoinSlug, clientIP)); locked != "" {
			uc.log.Warn(ctx, "joinlink.Join: client locked (brute-force)",
				logger.String("lesson_id", l.ID))
			return nil, apperr.Forbidden("too many failed attempts, try again later")
		}
		if locked, _ := uc.cache.Get(ctx, slugLockKey(l.JoinSlug)); locked != "" {
			uc.log.Warn(ctx, "joinlink.Join: slug locked (distributed brute-force)",
				logger.String("lesson_id", l.ID))
			return nil, apperr.Forbidden("too many failed attempts, try again later")
		}
		if req.Passcode == nil || *req.Passcode == "" {
			return nil, apperr.Unauthorized("passcode required")
		}
		if !uc.hasher.Check(*req.Passcode, *l.PasscodeHash) {
			n, _ := uc.cache.Incr(ctx, failKey(l.JoinSlug, clientIP), passcodeFailWindow)
			if n >= maxPasscodeAttempts {
				_ = uc.cache.Set(ctx, lockKey(l.JoinSlug, clientIP), "1", passcodeLockCooldown)
				uc.log.Warn(ctx, "joinlink.Join: passcode locked for client after too many attempts",
					logger.String("lesson_id", l.ID))
			}
			sn, _ := uc.cache.Incr(ctx, slugFailKey(l.JoinSlug), passcodeFailWindow)
			if sn >= maxSlugAttempts {
				_ = uc.cache.Set(ctx, slugLockKey(l.JoinSlug), "1", passcodeLockCooldown)
				uc.log.Error(ctx, "joinlink.Join: slug locked — taqsimlangan brute-force shubhasi",
					logger.String("lesson_id", l.ID))
			}
			return nil, apperr.Unauthorized("invalid passcode")
		}
		// Muvaffaqiyat — shu klientning hisoblagichini tozalaymiz. Slug
		// hisoblagichi ATAYLAB tegilmaydi: bitta to'g'ri parol (hujumchining
		// o'zi ham kiritishi mumkin) taqsimlangan hujum belgisini o'chirmasin.
		_ = uc.cache.Del(ctx, failKey(l.JoinSlug, clientIP))
	}

	resp := &entity.JoinLessonResp{
		Lesson:   uc.toPublic(ctx, l),
		NextStep: entity.JoinNextStepJoin,
	}

	// Kutish xonasi yoqilgan bo'lsa — so'rov yaratiladi, mentor tasdiqlashini kutadi.
	if l.IsWaitingRoomEnabled {
		guestName := ""
		if req.GuestName != nil {
			guestName = *req.GuestName
		}
		wr, err := uc.waitingRoom.CreateRequest(ctx, l, guestName)
		if err != nil {
			return nil, err
		}
		resp.NextStep = entity.JoinNextStepWaitingRoom
		resp.RequestID = wr.ID
		uc.log.Info(ctx, "joinlink.Join: waiting room", logger.String("lesson_id", l.ID), logger.String("request_id", wr.ID))
		return resp, nil
	}

	// Dars hali jonli emas (scheduled) — token BERILMAYDI: aks holda participant
	// token bilan bo'sh xona yaratilib, host kirmasdan egress/auto-end holatlari
	// chalkashardi. Klient host kirgach qayta urinadi.
	if l.Status != entity.LessonStatusLive {
		resp.NextStep = entity.JoinNextStepWaitingForHost
		uc.log.Info(ctx, "joinlink.Join: host hali kirmagan", logger.String("lesson_id", l.ID))
		return resp, nil
	}

	// To'g'ridan-to'g'ri kirish — participant tokeni beriladi.
	identity := "guest_" + uuid.NewString()
	displayName := "Mehmon"
	if req.GuestName != nil && *req.GuestName != "" {
		displayName = *req.GuestName
	}
	rt, err := uc.room.ParticipantToken(ctx, l, identity, displayName)
	if err != nil {
		return nil, err
	}
	resp.Room = rt

	uc.log.Info(ctx, "joinlink.Join: token issued", logger.String("lesson_id", l.ID), logger.String("identity", identity))
	return resp, nil
}

// resolve slug bo'yicha darsni topadi (holatidan qat'i nazar — yakunlangan dars
// ham topiladi; kirish siyosati `Join` da, ko'rsatish `Preview` da hal bo'ladi).
func (uc *useCase) resolve(ctx context.Context, slug string) (*entity.Lesson, error) {
	l, err := uc.lessonRepo.GetBySlug(ctx, slug)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("lesson")
		}
		return nil, err
	}
	return l, nil
}

// isBlockedName — guest ismi mentorning doimiy qora ro'yxatidami (№4).
// Fail-open: DB xatosi butun sinfni to'sib qo'ymasin (shared.IsBanned falsafasi).
func (uc *useCase) isBlockedName(ctx context.Context, mentorID string, guestName *string) bool {
	if uc.blocklist == nil || guestName == nil || *guestName == "" {
		return false
	}
	blocked, err := uc.blocklist.IsBlocked(ctx, mentorID, *guestName)
	if err != nil {
		uc.log.Warn(ctx, "joinlink: blocklist tekshirib bo'lmadi (fail-open)",
			logger.SafeString("err", err.Error()))
		return false
	}
	return blocked
}

// Kalitlar: klient darajasi (slug+IP) va slug darajasi alohida.
func failKey(slug, clientIP string) string { return "joinfail:" + slug + ":" + clientIP }
func lockKey(slug, clientIP string) string { return "joinlock:" + slug + ":" + clientIP }
func slugFailKey(slug string) string       { return "joinfail:slug:" + slug }
func slugLockKey(slug string) string       { return "joinlock:slug:" + slug }

func (uc *useCase) toPublic(ctx context.Context, l *entity.Lesson) *entity.LessonPublic {
	return &entity.LessonPublic{
		ID:                   l.ID,
		Title:                l.Title,
		MentorName:           uc.mentorName(ctx, l.MentorID),
		ScheduledAt:          l.ScheduledAt,
		Status:               l.Status,
		HasPasscode:          l.PasscodeHash != nil,
		IsWaitingRoomEnabled: l.IsWaitingRoomEnabled,
		// Ovoz siyosati — klient mikrofon tugmasini to'g'ri ko'rsatsin
		// (`entity.LessonPublic` izohiga qara). Sir emas: kirgan har bir
		// ishtirokchi buni baribir birinchi publish'da his qiladi.
		MuteOnEntry:     l.MuteOnEntry,
		AllowSelfUnmute: l.AllowSelfUnmute,
	}
}

// mentorNameCacheTTL — "mentorname:<id>" keshi muddati (qisqa: invalidatsiya yo'q).
const mentorNameCacheTTL = 5 * time.Minute

// mentorName mentor ismini qaytaradi — Redis cache bilan (har join'da DB o'qishni kamaytiradi).
func (uc *useCase) mentorName(ctx context.Context, mentorID string) string {
	key := "mentorname:" + mentorID
	if name, err := uc.cache.Get(ctx, key); err == nil && name != "" {
		return name
	}
	name := ""
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		name = m.FullName
	}
	if name != "" {
		// TTL qisqa (5 min): ism o'zgarganda (user.UpdateCurrentUser/admin Update)
		// kesh invalidatsiya qilinmaydi — user usecase'da cache yo'q, uni ulash invaziv.
		// Eskirgan ism ko'pi bilan mentorNameCacheTTL ko'rinadi.
		_ = uc.cache.Set(ctx, key, name, mentorNameCacheTTL)
	}
	return name
}
