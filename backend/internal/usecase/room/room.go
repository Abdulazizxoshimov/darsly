package room

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	// blocklist — mentor darajasidagi doimiy qora ro'yxat (№4). nil bo'lishi
	// mumkin (eski testlar); har chaqiruvdan oldin tekshiriladi.
	blocklist repository.BlocklistRepository
	// ensureG — bir xona uchun bir vaqtda faqat bitta EnsureRoom (thundering herd:
	// 500 talaba bir vaqtda kirsa 500 ta SFU CreateRoom o'rniga bitta chaqiruv).
	ensureG singleflight.Group
}

func New(lessonRepo repository.LessonRepository, userRepo repository.UserRepository, lk LiveKit, cache redis.Cache, log logger.Logger, recorder Recorder, hands Hands, blocklist repository.BlocklistRepository) UseCase {
	return &useCase{lessonRepo: lessonRepo, userRepo: userRepo, livekit: lk, cache: cache, log: log, recorder: recorder, hands: hands, blocklist: blocklist}
}

func roomReadyKey(lessonID string) string { return "room:ready:" + lessonID }

// Ban reyestri `usecase/shared` ga ko'chirildi: uni endi faqat token berish emas,
// xonaga tegishli BARCHA usecase'lar (roomstate, chat, poll) va
// `participant_joined` webhook'i ham tekshiradi. Sabablari `shared/ban.go` da.

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
		WSURL:    uc.livekit.ClientWSURL(shared.RequestHost(ctx)),
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
		WSURL:    uc.livekit.ClientWSURL(shared.RequestHost(ctx)),
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

	uc.teardown(ctx, l.ID)
	uc.log.Info(ctx, "room.EndLesson: ended", logger.String("lesson_id", l.ID))
	return nil
}

// teardown — dars yakunlangandan KEYINGI tozalash: yozuvni to'xtatish, LiveKit
// xonasini o'chirish, kesh bayroqlari va ko'tarilgan qo'llar.
//
// Ajratilganining sababi: darsni yakunlashning ikki yo'li bor — mentor tugmasi
// ([useCase.EndLesson]) va avtomatik yakun ([useCase.SweepAutoEnd]: 4 soatlik
// limit / bo'sh xona). Ikkalasida ham AYNAN shu tozalash bajarilishi shart.
// Nusxa ko'chirilganda avto-yakun yo'lida masalan `StopActiveForLesson` unutilib,
// yozuv abadiy "recording" holatida qolishi mumkin edi.
//
// Xato qaytarmaydi: bu bosqichdagi har bir amal "eng yaxshi harakat" —
// dars DB'da allaqachon yakunlangan va uni orqaga qaytarish noto'g'ri bo'lardi.
func (uc *useCase) teardown(ctx context.Context, lessonID string) {
	// Yozuvni xona o'chirilishidan OLDIN to'xtatamiz. Tartib muhim: xona
	// o'chirilsa egress o'zi ham tugaydi, lekin u holda DB'dagi yozuv webhook
	// kelguncha "recording" bo'lib turadi va ustoz yozuvlar ro'yxatida
	// "hali yozilmoqda" degan yolg'onni ko'radi.
	if uc.recorder != nil {
		if err := uc.recorder.StopActiveForLesson(ctx, lessonID); err != nil {
			// Darsni yakunlashni bloklamaydi — xona o'chirilishi baribir
			// egress'ni tugatadi va webhook yakuniy holatni qo'yadi.
			uc.log.Warn(ctx, "room.teardown: yozuvni to'xtatib bo'lmadi",
				logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		}
	}

	if uc.livekit.Enabled() {
		delCtx, delCancel := lkCtx(ctx)
		err := uc.livekit.DeleteRoom(delCtx, roomName(lessonID))
		delCancel()
		if err != nil {
			uc.log.Warn(ctx, "room.teardown: delete room failed", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		}
	}
	// Stale "room-ready" flagni tozalaymiz (aks holda qayta ulanishda EnsureRoom o'tkazib yuboriladi).
	if uc.cache != nil {
		_ = uc.cache.Del(ctx, roomReadyKey(lessonID))
		// Bo'shlik hisoblagichi ham keraksiz — dars tugadi.
		_ = uc.cache.Del(ctx, roomEmptyKey(lessonID))
	}
	// "Jonli" keshini ham tozalaymiz — busiz chat/reaksiya/ovoz endpointlari
	// yakunlangan darsni TTL tugagunicha jonli deb qabul qilaverardi
	// (`shared.GuardRoomAction`).
	shared.InvalidateLive(ctx, uc.cache, lessonID)
	// Ko'tarilgan qo'llar dars bilan birga tugaydi. Tozalanmasa, xuddi shu dars
	// qayta ochilganda (masalan takroriy mashg'ulot) eski navbat qayta paydo bo'lardi.
	if uc.hands != nil {
		if err := uc.hands.Clear(ctx, lessonID); err != nil {
			uc.log.Warn(ctx, "room.teardown: qo'l holatini tozalab bo'lmadi",
				logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		}
	}
}

// isBanned — ishtirokchi shu darsdan chiqarilganmi (`shared.IsBanned` ustidan).
func (uc *useCase) isBanned(ctx context.Context, lessonID, identity string) bool {
	return shared.IsBanned(ctx, uc.cache, lessonID, identity)
}

// EnforceJoin — `participant_joined` webhook'i uchun: chiqarilgan ishtirokchi
// xonaga qaytib kirgan bo'lsa uni darhol uzadi.
//
// # Nega token tekshiruvining o'zi yetmaydi
//
// `ParticipantToken` ban'ni tekshiradi, lekin u faqat YANGI token so'ralganda
// ishlaydi. Chiqarilgan ishtirokchining tokeni esa qo'lida qolgan va TTL
// tugagunicha yaroqli — u backendga umuman murojaat qilmasdan to'g'ridan-to'g'ri
// `room.connect(wsUrl, eskiToken)` qila oladi. Ya'ni "chiqarish" tugmasi
// serverdan o'tmaydigan yo'l orqali chetlab o'tilardi.
//
// Bu webhook esa aynan SFU tomonidan, har bir ulanishda chaqiriladi — token
// qayerdan kelganidan qat'i nazar. Qisqa ishtirokchi TTL'i bilan birga
// (`livekit.participantTokenTTLMax`) ban endi haqiqiy kuchga ega.
//
// Xato qaytmaydi: webhook non-200 olsa LiveKit hodisani qayta yuboradi, bu esa
// foyda bermaydi (ulanish allaqachon sodir bo'lgan). Muvaffaqiyatsizlik
// jurnalga yoziladi.
func (uc *useCase) EnforceJoin(ctx context.Context, rn, identity, displayName string) {
	if identity == "" || !uc.livekit.Enabled() {
		return
	}
	lessonID, ok := shared.LessonIDFromRoom(rn)
	if !ok {
		return // bizniki bo'lmagan xona
	}
	if !shared.IsBanned(ctx, uc.cache, lessonID, identity) &&
		!uc.isMentorBlockedByLesson(ctx, lessonID, identity, displayName) {
		return
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.RemoveParticipant(lctx, rn, identity); err != nil {
		uc.log.Error(ctx, "room.EnforceJoin: chiqarilgan ishtirokchini uzib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.String("identity", identity),
			logger.SafeString("err", err.Error()))
		return
	}
	uc.log.Info(ctx, "room.EnforceJoin: chiqarilgan ishtirokchi qaytib kirdi — uzildi",
		logger.String("lesson_id", lessonID), logger.String("identity", identity))
}

// isMentorBlockedByLesson — ishtirokchi shu dars mentorining DOIMIY qora
// ro'yxatidami (№4). Ism bo'yicha tekshiriladi (`repository.BlocklistRepository`
// izohiga qarang); host hech qachon bloklanmaydi (mentorID == identity).
//
// Fail-open (xatoda false): bu moderatsiya qulayligi, autentifikatsiya emas —
// DB uzilganda butun sinfni darsdan to'sish zarari bloklangan bitta odamning
// kirishidan katta (shared.IsBanned bilan bir xil falsafa).
func (uc *useCase) isMentorBlockedByLesson(ctx context.Context, lessonID, identity, displayName string) bool {
	if uc.blocklist == nil || displayName == "" {
		return false
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return false
	}
	if isHostIdentity(l.MentorID, identity) {
		return false
	}
	blocked, err := uc.blocklist.IsBlocked(ctx, l.MentorID, displayName)
	if err != nil {
		uc.log.Warn(ctx, "room: blocklist tekshirib bo'lmadi (fail-open)",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return false
	}
	return blocked
}

// entryMuteKey — "kirishdagi mute allaqachon qo'llangan" belgisi (Redis).
// TTL BanTTL bilan bir xil mantiqda: dars tugagach belgi ham keraksiz.
func entryMuteKey(lessonID, identity string) string {
	return "room:entrymuted:" + lessonID + ":" + identity
}

// EnforceAudioPolicy — `track_published` (audio) webhook'ida dars ovoz
// sozlamalarini SERVER tomonda qo'llaydi (Zoom modeli, №11):
//
//   - allow_self_unmute=false → HAR audio publish qayta mute qilinadi
//     ("unmute taqiqlangan" xulqi). Mentor bayroqni PATCH bilan jonli o'zgartiradi.
//   - mute_on_entry=true → ishtirokchining BIRINCHI audio publish'i mute qilinadi
//     (SetNX belgisi bilan bir marta): "kirganda mute", keyin o'zi ochsa — ochiq
//     qoladi (allow_self_unmute=true bo'lsa).
//
// Nega webhook, token emas: "kirganda mute" tokenga yozib bo'lmaydi (grant faqat
// publish HUQUQINI boshqaradi, trekning mute holatini emas), klientga ishonib
// bo'lmaydi (curl/konsol chetlab o'tadi). `participant_joined` da esa hali trek
// yo'q — mute qiladigan narsaning o'zi bo'lmaydi. Shuning uchun aynan
// `track_published`.
//
// Ma'lum cheklov (hujjatlashtirilgan): klient SDK'si mute'da trekni unpublish
// qilmasa, keyingi self-unmute yangi webhook bermaydi. Buni "Mute All"
// (allow_self_unmute=false bilan) yopadi — u joriy treklarni darhol mute qiladi,
// yangi publish'lar esa shu yerda tutiladi.
//
// Xato qaytarmaydi (EnforceJoin bilan bir xil): webhook'ga non-200 qaytarish
// hodisani qayta-qayta yubortiradi, foydasi yo'q; muvaffaqiyatsizlik jurnalda.
func (uc *useCase) EnforceAudioPolicy(ctx context.Context, rn, identity string) {
	if identity == "" || !uc.livekit.Enabled() {
		return
	}
	lessonID, ok := shared.LessonIDFromRoom(rn)
	if !ok {
		return // bizniki bo'lmagan xona
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return
	}
	// Host'ga ovoz siyosati qo'llanmaydi — ustoz har doim erkin gapiradi.
	if isHostIdentity(l.MentorID, identity) {
		return
	}

	mute := false
	switch {
	case !l.AllowSelfUnmute:
		mute = true
	case l.MuteOnEntry:
		// Faqat BIRINCHI publish (SetNX true qaytarsa). Keyingi publish'lar
		// (masalan mobil SDK unmute'da qayta publish qilsa) tegilmaydi —
		// aks holda "kirganda mute" amalda "hech qachon gapira olmaydi"ga aylanardi.
		if uc.cache != nil {
			first, err := uc.cache.SetNX(ctx, entryMuteKey(lessonID, identity), "1", shared.BanTTL)
			mute = err == nil && first
		}
	}
	if !mute {
		return
	}

	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.MuteParticipant(lctx, rn, identity, true); err != nil {
		uc.log.Warn(ctx, "room.EnforceAudioPolicy: mute qilib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.String("identity", identity),
			logger.SafeString("err", err.Error()))
		return
	}
	uc.log.Info(ctx, "room.EnforceAudioPolicy: audio mute qilindi",
		logger.String("lesson_id", lessonID), logger.String("identity", identity),
		logger.Int("allow_self_unmute", boolToInt(l.AllowSelfUnmute)))
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

func (uc *useCase) MuteAll(ctx context.Context, mentorID, lessonID string, allowSelfUnmute *bool) error {
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return err
	}
	// Zoom'dagi "Mute All" dialogidagi checkbox: bayroq AVVAL yangilanadi, keyin
	// mute — aks holda mute bilan bayroq orasida publish qilgan o'quvchi eski
	// (ruxsat beruvchi) siyosatga tushib qolardi.
	if allowSelfUnmute != nil && l.AllowSelfUnmute != *allowSelfUnmute {
		l.AllowSelfUnmute = *allowSelfUnmute
		if err := uc.lessonRepo.Update(ctx, l); err != nil {
			return err
		}
	}
	rn := roomName(lessonID)
	listCtx, listCancel := lkCtx(ctx)
	parts, lkErr := uc.livekit.ListParticipantViews(listCtx, rn)
	listCancel()
	if lkErr != nil {
		return apperr.Internal(fmt.Errorf("room.MuteAll list: %w", lkErr))
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

	// Siyosatni SERVER tarqatadi.
	//
	// Avval `kind:"policy"` xabarini faqat WEB ustoz klienti yuborardi. Mobil
	// ustoz hech narsa yubormasdi, ya'ni telefondan o'tilgan darsda o'quvchining
	// mikrofon tugmasi "yoqish mumkin" deb qolardi va bosilganda server uni
	// jimgina qayta mute qilardi — foydalanuvchi uchun "tugma ishlamadi".
	// Endi manba bitta: siyosat qayerda saqlansa (server), o'zgarish ham
	// o'sha yerdan e'lon qilinadi va HAR QANDAY klient bir xil xabar oladi.
	uc.broadcastPolicy(ctx, lessonID, l)

	uc.log.Info(ctx, "room: mute all", logger.String("lesson_id", lessonID), logger.Int("count", len(parts)))
	return nil
}

// broadcastPolicy — dars ovoz siyosatini xonaga e'lon qiladi.
//
// Xabar shakli klient kutayotgani bilan aynan mos
// (`frontend/src/livekit/messaging.js` → `kind:"policy"`).
//
// Tarqatish muvaffaqiyatsiz bo'lsa amal BEKOR QILINMAYDI: siyosat DB'da
// saqlangan va kech ulangan klient uni `GET /rooms/:lessonID/state` orqali
// baribir oladi (`entity.RoomState` izohiga qara).
func (uc *useCase) broadcastPolicy(ctx context.Context, lessonID string, l *entity.Lesson) {
	if !uc.livekit.Enabled() {
		return
	}
	data, err := json.Marshal(map[string]any{
		"kind":              "policy",
		"mute_on_entry":     l.MuteOnEntry,
		"allow_self_unmute": l.AllowSelfUnmute,
	})
	if err != nil {
		return
	}
	sendCtx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.SendData(sendCtx, roomName(lessonID), data); err != nil {
		uc.log.Warn(ctx, "room.broadcastPolicy: tarqatib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
	}
}

func (uc *useCase) RemoveParticipant(ctx context.Context, mentorID, lessonID, identity, scope string) error {
	switch scope {
	case "", entity.BanScopeLesson, entity.BanScopeMentor:
		// "" = default (lesson) — eski klientlar body yubormaydi.
	default:
		return apperr.BadRequest("scope must be \"lesson\" or \"mentor\"")
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host o'zini xonadan chiqarib yuborsa dars "live" holatida host'siz qoladi
	// (ishtirokchilar boshsiz xonada) — darsni tugatish uchun alohida endpoint bor.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot remove the host: use POST /lessons/:id/end to finish the lesson")
	}
	// DOIMIY ban (scope=mentor): ismni UZISHDAN OLDIN o'qiymiz — chiqarib
	// yuborilgan ishtirokchi ro'yxatda qolmaydi va ismi keyin topilmasdi.
	if scope == entity.BanScopeMentor {
		if err := uc.blockPermanently(ctx, mentorID, lessonID, identity); err != nil {
			// Doimiy ban yozilmasa amalni TO'XTATAMIZ: ustoz "doimiy" deb bosdi,
			// biz esa jimgina faqat bir darslik qilib qo'ysak — bu yolg'on UI.
			return err
		}
	}
	// AVVAL ban, KEYIN uzish. Tartib muhim: teskarisi bo'lsa, uzilish bilan ban
	// yozilishi orasidagi bir necha millisekundda ishtirokchi qayta ulanib
	// ulgurishi mumkin (klient SDK'si uzilishda darhol qayta urinadi).
	if err := shared.Ban(ctx, uc.cache, lessonID, identity); err != nil {
		uc.log.Error(ctx, "room.RemoveParticipant: ban yozib bo'lmadi — chiqarilgan qaytib kirishi mumkin",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
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

// blockPermanently — ishtirokchini mentorning doimiy qora ro'yxatiga qo'shadi
// (scope=mentor). Ism xonadagi joriy ro'yxatdan olinadi; ishtirokchi allaqachon
// chiqib ketgan bo'lsa ism bo'sh qoladi (faqat identity/audit) — bu holda
// ism-mosligi ishlamasligi ongli cheklov (`migrations/000015` izohi).
func (uc *useCase) blockPermanently(ctx context.Context, mentorID, lessonID, identity string) error {
	if uc.blocklist == nil {
		return apperr.Internal(fmt.Errorf("room.blockPermanently: blocklist repository is not configured"))
	}
	displayName := ""
	lctx, cancel := lkCtx(ctx)
	parts, err := uc.livekit.ListParticipantViews(lctx, roomName(lessonID))
	cancel()
	if err != nil {
		uc.log.Warn(ctx, "room.blockPermanently: ishtirokchi ismini o'qib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
	} else {
		for _, p := range parts {
			if p.Identity == identity {
				displayName = p.Name
				break
			}
		}
	}
	e := &entity.BlocklistEntry{
		ID:          uuid.NewString(),
		MentorID:    mentorID,
		Identity:    identity,
		DisplayName: displayName,
	}
	if err := uc.blocklist.Add(ctx, e); err != nil {
		uc.log.Error(ctx, "room.blockPermanently: qora ro'yxatga yozib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return apperr.Internal(fmt.Errorf("room.blockPermanently: %w", err))
	}
	uc.log.Info(ctx, "room: participant blocked permanently",
		logger.String("mentor_id", mentorID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) ListBlocklist(ctx context.Context, mentorID string) ([]*entity.BlocklistEntry, error) {
	if uc.blocklist == nil {
		return nil, apperr.Internal(fmt.Errorf("room.ListBlocklist: blocklist repository is not configured"))
	}
	items, err := uc.blocklist.ListByMentor(ctx, mentorID)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("room.ListBlocklist: %w", err))
	}
	return items, nil
}

func (uc *useCase) Unblock(ctx context.Context, mentorID, entryID string) error {
	if uc.blocklist == nil {
		return apperr.Internal(fmt.Errorf("room.Unblock: blocklist repository is not configured"))
	}
	if err := shared.ValidateID(entryID, "blocklist entry"); err != nil {
		return err
	}
	if err := uc.blocklist.Delete(ctx, mentorID, entryID); err != nil {
		return err
	}
	uc.log.Info(ctx, "room: blocklist entry removed", logger.String("mentor_id", mentorID), logger.String("entry_id", entryID))
	return nil
}
