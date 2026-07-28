package room

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// MuteAll uchun bir vaqtdagi chaqiruvlar soni. 8 — LiveKit'ni bo'kdirmaydigan,
// lekin 100 kishilik darsni ~2 soniyada qoplaydigan oraliq.
const muteAllConcurrency = 8

// Hamma chiqib ketgach xona shuncha soniya ochiq turadi (qayta ulanish uchun).
const roomEmptyTimeoutSec = 300

// Xona yaratilgani Redis'da shuncha vaqt eslab qolinadi (join hot-path'da qayta
// CreateRoom chaqirmaslik uchun — kechikishni kamaytiradi).
const roomReadyTTL = 6 * time.Hour

// LiveKit API chaqiruvlari uchun qat'iy deadline — backend↔LiveKit tarmog'i
// yomonlashsa request WriteTimeout'gacha osilib qolmasin (SDK klienti timeout'siz).
const lkOpTimeout = 10 * time.Second

// lkCtx LiveKit chaqiruvini lkOpTimeout bilan cheklaydi (chaqiruvchi ctx bekor bo'lsa
// ham darhol uzadi). Har chaqiruvdan keyin cancel() chaqirilishi shart.
func lkCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, lkOpTimeout)
}

// Recorder — dars yakunlanganda yozuvni to'xtatish uchun minimal shartnoma
// (DIP; iste'molchi tomonda e'lon qilingan — import sikli bo'lmaydi).
//
// Yozuvni BOSHLASH bu yerda emas: u `track_published` webhook'ida bo'ladi
// (`recording.EnsureForRoom`). To'xtatish esa aynan shu yerga tegishli —
// "dars tugadi" haqiqati `EndLesson` da tug'iladi. "Jonli bo'ldi" haqiqati aynan shu yerda ([useCase.HostToken])
// tug'iladi, shuning uchun signal ham shu yerdan chiqadi. Muqobil — handler
// qatlamida ikkita usecase'ni ketma-ket chaqirish — biznes qoidasini HTTP
// qatlamiga sizdirardi (CLAUDE.md: "Handler faqat HTTP parse").
type Recorder interface {
	StopActiveForLesson(ctx context.Context, lessonID string) error
}

// Hands — qo'l ko'tarish holatining minimal shartnomasi (DIP; iste'molchi tomonda
// e'lon qilingan, `roomstate` paketi import qilinmaydi).
//
// Nega `room` bunga muhtoj: "so'zga ruxsat berildi" va "dars tugadi" haqiqatlari
// aynan shu yerda tug'iladi, qo'lning taqdiri esa ulardan KELIB CHIQADI:
//   - ruxsat berilgan o'quvchining qo'li ko'tarilgan turishi ma'nosiz (navbat axlatga to'ladi);
//   - dars tugagach qo'llar hech kimga kerak emas.
//
// Buni handler qatlamida ikkita usecase'ni ketma-ket chaqirish bilan qilish
// biznes qoidasini HTTP qatlamiga sizdirardi (CLAUDE.md: "Handler faqat HTTP parse").
type Hands interface {
	LowerHand(ctx context.Context, mentorID, lessonID, identity string) error
	Clear(ctx context.Context, lessonID string) error
}

type useCase struct {
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	livekit    LiveKit
	cache      redis.Cache
	log        logger.Logger
	// recorder — nil bo'lishi mumkin (masalan testlarda yoki yozib olish
	// o'chirilgan muhitda); har chaqiruvdan oldin tekshiriladi.
	recorder Recorder
	// hands — nil bo'lishi mumkin (eski testlar); har chaqiruvdan oldin tekshiriladi.
	hands Hands
	// ensureG — bir xona uchun bir vaqtda faqat bitta EnsureRoom (thundering herd:
	// 500 talaba bir vaqtda kirsa 500 ta SFU CreateRoom o'rniga bitta chaqiruv).
	ensureG singleflight.Group
}

func New(lessonRepo repository.LessonRepository, userRepo repository.UserRepository, lk LiveKit, cache redis.Cache, log logger.Logger, recorder Recorder, hands Hands) UseCase {
	return &useCase{lessonRepo: lessonRepo, userRepo: userRepo, livekit: lk, cache: cache, log: log, recorder: recorder, hands: hands}
}

func roomReadyKey(lessonID string) string { return "room:ready:" + lessonID }

// banKey — darsdan chiqarilgan ishtirokchi.
//
// Nega kerak: LiveKit'da `RemoveParticipant` faqat JORIY ulanishni uzadi. Token
// esa hali yaroqli (TTL soatlar bilan o'lchanadi) va `room.auto_create` yoqilgan —
// ya'ni chiqarilgan buzg'unchi darhol qaytib ulanadi va hatto YOPILGAN xonani
// qayta yaratadi. Zoom'da "remove" = qaytib kira olmaslik; bizda ham shunday
// bo'lishi kerak.
//
// TTL — darsning oqilona uzunligi: dars tugagach ban ham keraksiz (yangi darsda
// yangi xona, yangi token). Redis o'zi tozalaydi — alohida ish yuritish shart emas.
func banKey(lessonID, identity string) string { return "room:ban:" + lessonID + ":" + identity }

// banTTL — chiqarilgan ishtirokchi shuncha vaqt qayta kira olmaydi.
const banTTL = 6 * time.Hour

// markRoomReady xona yaratilganini Redis'ga belgilaydi.
func (uc *useCase) markRoomReady(ctx context.Context, lessonID string) {
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, roomReadyKey(lessonID), "1", roomReadyTTL)
	}
}

// ensureRoomOnce faqat xona hali yaratilmagan bo'lsa CreateRoom chaqiradi
// (join hot-path'da ortiqcha SFU chaqiruvini oldini oladi).
func (uc *useCase) ensureRoomOnce(ctx context.Context, lessonID, rn string) {
	if uc.cache != nil {
		if v, _ := uc.cache.Get(ctx, roomReadyKey(lessonID)); v != "" {
			return // allaqachon yaratilgan
		}
	}
	// singleflight: bir xona uchun bir vaqtda faqat bitta EnsureRoom bajariladi
	// (qolgan parallel join'lar shu natijani kutadi — SFU thundering herd yo'q).
	_, _, _ = uc.ensureG.Do(roomReadyKey(lessonID), func() (any, error) {
		lctx, cancel := lkCtx(ctx)
		defer cancel()
		if err := uc.livekit.EnsureRoom(lctx, rn, roomEmptyTimeoutSec); err != nil {
			uc.log.Warn(ctx, "room.ensureRoomOnce: ensure failed (auto-create fallback)", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
			return nil, nil
		}
		uc.markRoomReady(ctx, lessonID)
		return nil, nil
	})
}

func (uc *useCase) HostToken(ctx context.Context, mentorID, lessonID string) (*entity.RoomToken, error) {
	// Avval mavjudlik/egalik/holat tekshiriladi — video servis o'chirilgan bo'lsa ham
	// yo'q dars 404, begona dars 403 bo'lib qolsin (500 bilan niqoblanmasin).
	// ValidateID: yaroqsiz UUID Postgres'ga yetmasin (22P02 → 500). OwnedLesson'ga
	// o'tmaymiz — bu yerdagi xato matni ("not the host") API kontraktida saqlanadi.
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if l.MentorID != mentorID {
		return nil, apperr.Forbidden("you are not the host of this lesson")
	}
	if l.Status == entity.LessonStatusEnded || l.Status == entity.LessonStatusCancelled {
		return nil, apperr.BadRequest("lesson is not active")
	}
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}

	start := time.Now()
	defer func() { metrics.LiveKitOpDuration.WithLabelValues("host_token").Observe(time.Since(start).Seconds()) }()

	roomName := roomName(l.ID)
	ensureCtx, ensureCancel := lkCtx(ctx)
	err = uc.livekit.EnsureRoom(ensureCtx, roomName, roomEmptyTimeoutSec)
	ensureCancel()
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("create_room").Inc()
		uc.log.Error(ctx, "room.HostToken: ensure room failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not create video room: %w", err))
	}
	uc.markRoomReady(ctx, l.ID) // keyingi join'larda CreateRoom takrorlanmasin

	// Darsni jonli holatga o'tkazish (birinchi marta host qo'shilganda).
	if l.Status != entity.LessonStatusLive {
		now := time.Now().UTC()
		l.Status = entity.LessonStatusLive
		if l.StartedAt == nil {
			l.StartedAt = &now
		}
		if err := uc.lessonRepo.Update(ctx, l); err != nil {
			uc.log.Warn(ctx, "room.HostToken: could not mark lesson live", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}

	// YOZIB OLISH bu yerdan boshlanMAYDI — u `track_published` webhook'ida
	// boshlanadi (`recording.EnsureForRoom`). Sabab: egress xonaga kirib media
	// kutadi va 5 daqiqada kutgani kelmasa bekor bo'ladi, token berilishi bilan
	// media paydo bo'lishi orasida esa ruxsat so'rash va ulanish bor.
	// Batafsil: `recording.UseCase.EnsureRecording` izohi.

	hostName := "Host"
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		hostName = m.FullName
	}

	token, err := uc.livekit.AccessToken(roomName, mentorID, hostName, true)
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("host_token").Inc()
		return nil, apperr.Internal(fmt.Errorf("room.HostToken generate token: %w", err))
	}

	metrics.RoomTokensIssued.WithLabelValues("host").Inc()
	uc.log.Info(ctx, "room.HostToken: issued", logger.String("lesson_id", l.ID), logger.String("mentor_id", mentorID))
	return &entity.RoomToken{
		Token:    token,
		WSURL:    uc.livekit.WSURL(),
		RoomName: roomName,
		Identity: mentorID,
		Role:     entity.RoomRoleHost,
		LessonID: l.ID,
	}, nil
}

func (uc *useCase) ParticipantToken(ctx context.Context, l *entity.Lesson, identity, displayName string) (*entity.RoomToken, error) {
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	// Chiqarilgan ishtirokchiga YANGI token ham berilmaydi — aks holda "chiqarish"
	// atigi bir necha soniyalik noqulaylik bo'lardi.
	if uc.isBanned(ctx, l.ID, identity) {
		return nil, apperr.Forbidden("you were removed from this lesson")
	}
	roomName := roomName(l.ID)
	// Xonani faqat kerak bo'lsa yaratamiz (hot-path'da ortiqcha SFU chaqiruvi yo'q).
	uc.ensureRoomOnce(ctx, l.ID, roomName)
	if displayName == "" {
		displayName = "Mehmon"
	}

	token, err := uc.livekit.AccessToken(roomName, identity, displayName, false)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("room.ParticipantToken generate token: %w", err))
	}

	metrics.RoomTokensIssued.WithLabelValues("participant").Inc()
	uc.log.Info(ctx, "room.ParticipantToken: issued", logger.String("lesson_id", l.ID), logger.String("identity", identity))
	return &entity.RoomToken{
		Token:    token,
		WSURL:    uc.livekit.WSURL(),
		RoomName: roomName,
		Identity: identity,
		Role:     entity.RoomRoleParticipant,
		LessonID: l.ID,
	}, nil
}

func (uc *useCase) EndLesson(ctx context.Context, mentorID, lessonID string) error {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return err
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return err
	}
	if l.MentorID != mentorID {
		return apperr.Forbidden("you are not the host of this lesson")
	}

	now := time.Now().UTC()
	l.Status = entity.LessonStatusEnded
	l.EndedAt = &now
	if err := uc.lessonRepo.Update(ctx, l); err != nil {
		uc.log.Error(ctx, "room.EndLesson: update failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return err
	}

	// Yozuvni xona o'chirilishidan OLDIN to'xtatamiz. Tartib muhim: xona
	// o'chirilsa egress o'zi ham tugaydi, lekin u holda DB'dagi yozuv webhook
	// kelguncha "recording" bo'lib turadi va ustoz yozuvlar ro'yxatida
	// "hali yozilmoqda" degan yolg'onni ko'radi.
	if uc.recorder != nil {
		if err := uc.recorder.StopActiveForLesson(ctx, l.ID); err != nil {
			// Darsni yakunlashni bloklamaydi — xona o'chirilishi baribir
			// egress'ni tugatadi va webhook yakuniy holatni qo'yadi.
			uc.log.Warn(ctx, "room.EndLesson: yozuvni to'xtatib bo'lmadi",
				logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}

	if uc.livekit.Enabled() {
		delCtx, delCancel := lkCtx(ctx)
		err := uc.livekit.DeleteRoom(delCtx, roomName(l.ID))
		delCancel()
		if err != nil {
			uc.log.Warn(ctx, "room.EndLesson: delete room failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}
	// Stale "room-ready" flagni tozalaymiz (aks holda qayta ulanishda EnsureRoom o'tkazib yuboriladi).
	if uc.cache != nil {
		_ = uc.cache.Del(ctx, roomReadyKey(l.ID))
	}
	// Ko'tarilgan qo'llar dars bilan birga tugaydi. Tozalanmasa, xuddi shu dars
	// qayta ochilganda (masalan takroriy mashg'ulot) eski navbat qayta paydo bo'lardi.
	if uc.hands != nil {
		if err := uc.hands.Clear(ctx, l.ID); err != nil {
			uc.log.Warn(ctx, "room.EndLesson: qo'l holatini tozalab bo'lmadi",
				logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}

	uc.log.Info(ctx, "room.EndLesson: ended", logger.String("lesson_id", l.ID))
	return nil
}

// isBanned — ishtirokchi shu darsdan chiqarilganmi.
//
// Redis yetib bo'lmasa `false` qaytadi (fail-open): ban — moderatsiya qulayligi,
// autentifikatsiya emas. Redis uzilganda butun darsga kirishni to'sib qo'yish
// zarari chiqarilgan bitta kishining qaytib kirishidan ancha katta.
func (uc *useCase) isBanned(ctx context.Context, lessonID, identity string) bool {
	if uc.cache == nil || identity == "" {
		return false
	}
	v, err := uc.cache.Get(ctx, banKey(lessonID, identity))
	return err == nil && v != ""
}

// roomName dars ID sidan barqaror LiveKit xona nomini hosil qiladi (shared.RoomName).
func roomName(lessonID string) string {
	return shared.RoomName(lessonID)
}

func (uc *useCase) ListParticipants(ctx context.Context, mentorID, lessonID string) ([]entity.RoomParticipant, error) {
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	items, err := uc.livekit.ListParticipantViews(lctx, roomName(lessonID))
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("room.ListParticipants: %w", err))
	}
	return items, nil
}

// isHostIdentity — LiveKit'dagi host identity'si mentorID bilan AYNAN bir xil
// (HostToken: `AccessToken(roomName, mentorID, hostName, true)`), shuning uchun
// moderatsiya amallarining nishoni mentorID bo'lsa — ustoz o'ziga qarshi amal
// bajarayotgan bo'ladi. MuteAll bu qoidani allaqachon qo'llaydi (host'ni o'tkazib
// yuboradi); qolgan uch amalda guard yo'q edi. Eng xatarlisi allow-speak: u
// CanPublishSources'ni [CAMERA, MICROPHONE] ga tushiradi va ustozning EKRAN
// ULASHISHINI o'chirib qo'yadi (qayta ulanmaguncha tiklanmaydi).
func isHostIdentity(mentorID, identity string) bool {
	return identity != "" && identity == mentorID
}

func (uc *useCase) MuteParticipant(ctx context.Context, mentorID, lessonID, identity string, audioOnly bool) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host'ni moderatsiya API'si orqali mute qilmaymiz — MuteAll bilan izchil.
	// O'z mikrofonini o'chirish klient SDK'sining ishi (lokal mute), server-mute emas.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot mute the host: use the client SDK to mute your own microphone")
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.MuteParticipant(lctx, roomName(lessonID), identity, audioOnly); err != nil {
		return apperr.Internal(fmt.Errorf("room.MuteParticipant: %w", err))
	}
	uc.log.Info(ctx, "room: participant muted", logger.String("lesson_id", lessonID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) MuteAll(ctx context.Context, mentorID, lessonID string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	rn := roomName(lessonID)
	listCtx, listCancel := lkCtx(ctx)
	parts, err := uc.livekit.ListParticipantViews(listCtx, rn)
	listCancel()
	if err != nil {
		return apperr.Internal(fmt.Errorf("room.MuteAll list: %w", err))
	}
	// PARALLEL mute. Ketma-ket bajarilganda 100 kishilik darsda ~100 ta HTTP
	// chaqiruvi navbatga tushardi: har biri 10 s timeout bilan, ya'ni eng yomon
	// holatda so'rov `WriteTimeout` dan oshib ketardi va ustoz "mute all" ning
	// natijasini umuman ko'rmasdi.
	//
	// Konkurensiya cheklangan: cheksiz goroutine LiveKit'ni o'zimiz DDoS qilardi.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(muteAllConcurrency)
	for _, p := range parts {
		if p.Identity == mentorID {
			continue // host'ni mute qilmaymiz
		}
		identity := p.Identity
		g.Go(func() error {
			muteCtx, muteCancel := lkCtx(gctx)
			defer muteCancel()
			// Bitta ishtirokchidagi xato qolganlarini to'xtatmaydi: "hammani
			// mute qil" amali qisman bajarilgani umuman bajarilmaganidan yaxshi.
			if err := uc.livekit.MuteParticipant(muteCtx, rn, identity, true); err != nil {
				uc.log.Warn(ctx, "room.MuteAll: ishtirokchini mute qilib bo'lmadi",
					logger.String("identity", identity), logger.SafeString("err", err.Error()))
			}
			return nil
		})
	}
	_ = g.Wait()

	uc.log.Info(ctx, "room: mute all", logger.String("lesson_id", lessonID), logger.Int("count", len(parts)))
	return nil
}

func (uc *useCase) RemoveParticipant(ctx context.Context, mentorID, lessonID, identity string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host o'zini xonadan chiqarib yuborsa dars "live" holatida host'siz qoladi
	// (ishtirokchilar boshsiz xonada) — darsni tugatish uchun alohida endpoint bor.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot remove the host: use POST /lessons/:id/end to finish the lesson")
	}
	// AVVAL ban, KEYIN uzish. Tartib muhim: teskarisi bo'lsa, uzilish bilan ban
	// yozilishi orasidagi bir necha millisekundda ishtirokchi qayta ulanib
	// ulgurishi mumkin (klient SDK'si uzilishda darhol qayta urinadi).
	if uc.cache != nil {
		if err := uc.cache.Set(ctx, banKey(lessonID, identity), "1", banTTL); err != nil {
			uc.log.Error(ctx, "room.RemoveParticipant: ban yozib bo'lmadi — chiqarilgan qaytib kirishi mumkin",
				logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		}
	}

	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.RemoveParticipant(lctx, roomName(lessonID), identity); err != nil {
		return apperr.Internal(fmt.Errorf("room.RemoveParticipant: %w", err))
	}
	uc.log.Info(ctx, "room: participant removed (banned)", logger.String("lesson_id", lessonID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) SetSpeakPermission(ctx context.Context, mentorID, lessonID, identity string, canPublish bool) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host'ning publish huquqlari FAQAT host token'idan keladi (screen-share bilan).
	// Bu yerdan o'tsa student to'plamiga (kamera+mikrofon) tushib, ekran ulashish
	// o'chib qolardi — mobil ilovaning asosiy funksiyasi.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot change the host's publish permissions: host rights come from the host token")
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.SetParticipantPublish(lctx, roomName(lessonID), identity, canPublish); err != nil {
		return apperr.Internal(fmt.Errorf("room.SetSpeakPermission: %w", err))
	}
	// Ruxsat BERILGANDA qo'l avtomatik tushadi: o'quvchi so'ragan narsani oldi,
	// navbatda turishining ma'nosi qolmadi. Aks holda ustoz har safar ikki amal
	// bajarishga majbur bo'lardi va navbat ro'yxati amalda tozalanmasdi.
	// Ruxsat OLIB TASHLANGANDA tegilmaydi — o'quvchi yana so'rashi mumkin.
	if canPublish && uc.hands != nil {
		if err := uc.hands.LowerHand(ctx, mentorID, lessonID, identity); err != nil {
			uc.log.Warn(ctx, "room.SetSpeakPermission: qo'lni tushirib bo'lmadi",
				logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		}
	}

	uc.log.Info(ctx, "room: speak permission set", logger.String("lesson_id", lessonID), logger.String("identity", identity), logger.Int("can_publish", boolToInt(canPublish)))
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
