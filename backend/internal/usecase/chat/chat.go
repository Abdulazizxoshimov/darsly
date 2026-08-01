package chat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Tezlik cheklovi: bitta ishtirokchi 5 soniyada 5 ta xabar.
//
// Ustozga TEGMAYDI (u JWT bilan keladi va o'z darsining egasi) — ustozning
// tez-tez yozishi normal, buzg'unchilik esa xonadagi noma'lum ishtirokchidan
// keladi. Klient tomonda ham cheklov bor, lekin u faqat halol klientni to'xtatadi.
const (
	chatWindow = 5 * time.Second
	chatMax    = 5
)

type useCase struct {
	repo       repository.ChatRepository
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	livekit    LiveKit
	// storage — fayl ilovalari uchun (nil bo'lishi mumkin: MinIO sozlanmagan
	// muhitda chat matn sifatida ishlashda davom etadi).
	storage Storage
	cache   redis.Cache
	log     logger.Logger
}

func New(
	repo repository.ChatRepository,
	lessonRepo repository.LessonRepository,
	userRepo repository.UserRepository,
	lk LiveKit,
	storage Storage,
	cache redis.Cache,
	log logger.Logger,
) UseCase {
	return &useCase{
		repo: repo, lessonRepo: lessonRepo, userRepo: userRepo,
		livekit: lk, storage: storage, cache: cache, log: log,
	}
}

func chatRateKey(lessonID, identity string) string   { return "rl:chat:" + lessonID + ":" + identity }
func uploadRateKey(lessonID, identity string) string { return "rl:chatup:" + lessonID + ":" + identity }

// deliver — xabarni tarqatadi. Shaxsiy bo'lsa FAQAT ikki tomonga.
//
// Yetkazish xatosi amalni bekor qilmaydi: xabar DB'da saqlangan va tarixdan
// baribir keladi. Aksincha bo'lsa — saqlangan, lekin "yuborilmadi" deb xato
// qaytarilgan xabar foydalanuvchini ikki marta yozishga majbur qilardi.
func (uc *useCase) deliver(ctx context.Context, msg *entity.ChatMessage) {
	if uc.livekit == nil || !uc.livekit.Enabled() {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error(ctx, "chat: marshal failed", logger.SafeString("err", err.Error()))
		return
	}
	room := shared.RoomName(msg.LessonID)

	var derr error
	if msg.ToIdentity != nil {
		// Ikkala tomon ham oladi: qabul qiluvchi — xabarni, yuboruvchi — o'z
		// yozganini boshqa qurilmada/qayta ulanganda ko'rish uchun.
		derr = uc.livekit.SendDataTo(ctx, room, data, []string{*msg.ToIdentity, msg.SenderIdentity})
	} else {
		derr = uc.livekit.SendData(ctx, room, data)
	}
	if derr != nil {
		uc.log.Warn(ctx, "chat: broadcast failed",
			logger.String("lesson_id", msg.LessonID), logger.SafeString("err", derr.Error()))
	}
}

// build — xabar obyektini yasaydi va saqlaydi. file nil bo'lsa oddiy matn.
func (uc *useCase) build(ctx context.Context, lessonID, identity, name, body, to string, file *entity.ChatFile) (*entity.ChatMessage, error) {
	msg := &entity.ChatMessage{
		ID:             uuid.NewString(),
		LessonID:       lessonID,
		SenderIdentity: identity,
		SenderName:     name,
		Body:           body,
		CreatedAt:      time.Now().UTC(),
		File:           file,
	}
	if to != "" && to != identity { // o'ziga yozish ma'nosiz — ommaviy deb qaraymiz
		msg.ToIdentity = &to
	}
	if err := uc.repo.Create(ctx, msg); err != nil {
		uc.log.Error(ctx, "chat: db error",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, err
	}
	// Havola SAQLANGANDAN KEYIN imzolanadi: u DB'ga tushmaydi, faqat javob va
	// data-channel xabari uchun (`entity.ChatFile.URL` izohiga qara).
	uc.signFile(ctx, msg)
	uc.deliver(ctx, msg)
	return msg, nil
}

// signFile — xabardagi faylga vaqtinchalik presigned havola qo'yadi.
//
// Xato bo'lsa xabar FAYLSIZ HAVOLA bilan qaytadi (metama'lumot qoladi), lekin
// so'rov yiqilmaydi: bitta ilovaning havolasi olinmagani butun chat tarixini
// ko'rsatmaslikka sabab bo'lmasligi kerak.
func (uc *useCase) signFile(ctx context.Context, msg *entity.ChatMessage) {
	if msg == nil || msg.File == nil || msg.File.Key == "" || uc.storage == nil {
		return
	}
	url, err := uc.storage.PresignedURL(ctx, msg.File.Key, ChatFileURLTTL)
	if err != nil {
		uc.log.Warn(ctx, "chat: presign failed",
			logger.String("message_id", msg.ID), logger.SafeString("err", err.Error()))
		return
	}
	msg.File.URL = url
	msg.File.ExpiresInS = int(ChatFileURLTTL.Seconds())
}

func (uc *useCase) signAll(ctx context.Context, msgs []*entity.ChatMessage) []*entity.ChatMessage {
	for _, m := range msgs {
		uc.signFile(ctx, m)
	}
	return msgs
}

// upload — ikkala yo'l uchun umumiy fayl yuklash oqimi.
//
// Tartib MUHIM: avval tekshiruv (arzon, xotirada), keyin MinIO'ga yozish,
// oxirida DB. DB yozuvi yiqilsa MinIO'da yetim obyekt qoladi — bu ongli
// tanlov: teskarisi (avval DB) esa "xabar bor, fayl yo'q" holatini berardi,
// ya'ni foydalanuvchi ko'radigan buzilish. Yetim obyekt esa ko'rinmaydi va
// retention ishi bilan tozalanadi.
func (uc *useCase) upload(ctx context.Context, lessonID, identity, name string, f FileUpload, body, to string) (*entity.ChatMessage, error) {
	if uc.storage == nil {
		return nil, apperr.BadRequest("file sharing is not available")
	}
	vf, err := validateFile(f.Name, f.Size, f.Reader)
	if err != nil {
		return nil, err
	}
	if uc.cache != nil {
		n, err := uc.cache.Incr(ctx, uploadRateKey(lessonID, identity), uploadWindow)
		if err == nil && n > uploadMax {
			return nil, apperr.TooManyRequests("too many uploads — slow down")
		}
	}

	key := "chat/" + lessonID + "/" + uuid.NewString() + vf.ext
	if _, err := uc.storage.Upload(ctx, key, vf.mime, vf.body, vf.size); err != nil {
		uc.log.Error(ctx, "chat.upload: storage error",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(err)
	}

	msg, err := uc.build(ctx, lessonID, identity, name, body, to, &entity.ChatFile{
		Name: vf.name, Size: vf.size, Mime: vf.mime, Key: key,
	})
	if err != nil {
		return nil, err
	}
	uc.log.Info(ctx, "chat file uploaded",
		logger.String("lesson_id", lessonID), logger.String("message_id", msg.ID))
	return msg, nil
}

// ─── Host yo'li (JWT + egalik) ────────────────────────────────────────────────

func (uc *useCase) Send(ctx context.Context, mentorID, lessonID, body, to string) (*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	msg, err := uc.build(ctx, lessonID, mentorID, uc.hostName(ctx, mentorID), body, to, nil)
	if err != nil {
		return nil, err
	}
	uc.log.Info(ctx, "chat message sent (host)", logger.String("lesson_id", lessonID))
	return msg, nil
}

func (uc *useCase) Upload(ctx context.Context, mentorID, lessonID string, f FileUpload, body, to string) (*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	return uc.upload(ctx, lessonID, mentorID, uc.hostName(ctx, mentorID), f, body, to)
}

// hostName — ustozning ko'rsatiladigan ismi (topilmasa "Host").
func (uc *useCase) hostName(ctx context.Context, mentorID string) string {
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		return m.FullName
	}
	return "Host"
}

func (uc *useCase) History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	// Host identity'si = mentorID (`room.HostToken` shunday beradi), ya'ni ustoz
	// o'zining shaxsiy yozishmalarini ko'radi, boshqalarnikini emas. Ustoz
	// darsning egasi bo'lsa-da, o'quvchilarning bir-biriga yozgani unga tegishli
	// emas — bu maxfiylik qarori, texnik cheklov emas.
	items, err := uc.repo.ListByLesson(ctx, lessonID, mentorID, before, limit)
	if err != nil {
		return nil, err
	}
	return uc.signAll(ctx, items), nil
}

// Transcript — chat eksporti (ish №21). Interfeys izohida sabab.
//
// Bu yerda faqat SIMLASH bor: egalik, ma'lumot yig'ish va format tanlash.
// Butun formatlash mantig'i `transcript.go` dagi sof funksiyalarda —
// shuning uchun "vaqt mintaqasi to'g'rimi", "XSS escape qilinganmi" degan
// savollar DB'siz va HTTP'siz sinaladi.
func (uc *useCase) Transcript(ctx context.Context, mentorID, lessonID, format string) (*entity.ChatTranscript, error) {
	if format == "" {
		format = FormatTXT
	}
	if format != FormatTXT && format != FormatHTML {
		return nil, apperr.BadRequest("format must be txt or html")
	}
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}
	// Ko'rinuvchanlik `History` bilan bir xil (mentorID = host identity).
	msgs, err := uc.repo.ListAllByLesson(ctx, lessonID, mentorID, 0)
	if err != nil {
		return nil, err
	}

	data := TranscriptData{
		LessonTitle:  l.Title,
		LessonDate:   lessonDate(l),
		MentorName:   uc.hostName(ctx, mentorID),
		HostIdentity: mentorID,
		Messages:     msgs,
	}
	body, ctype := RenderTXT(data), "text/plain; charset=utf-8"
	if format == FormatHTML {
		body, ctype = RenderHTML(data), "text/html; charset=utf-8"
	}
	uc.log.Info(ctx, "chat transcript exported",
		logger.String("lesson_id", lessonID), logger.String("format", format))
	return &entity.ChatTranscript{
		Filename:    safeFilename(l.Title, data.LessonDate, format),
		ContentType: ctype,
		Body:        []byte(body),
	}, nil
}

// lessonDate — transkript sarlavhasidagi «Sana».
//
// Tanlov tartibi: haqiqatan boshlangan vaqt → rejalashtirilgan vaqt →
// yaratilgan vaqt. Birinchisi eng to'g'ri, lekin dars jonli bo'lmagan bo'lsa
// (bekor qilingan, faqat chat yozilgan) u bo'sh — shunda ham sarlavhada
// "01.01.0001" chiqmasligi kerak.
func lessonDate(l *entity.Lesson) time.Time {
	if l.StartedAt != nil {
		return *l.StartedAt
	}
	if l.ScheduledAt != nil {
		return *l.ScheduledAt
	}
	return l.CreatedAt
}

// Delete — moderatsiya (№6). Interfeys izohida sabab.
func (uc *useCase) Delete(ctx context.Context, mentorID, lessonID, messageID string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	if err := shared.ValidateID(messageID, "chat message"); err != nil {
		return err
	}
	// Atomik: `WHERE deleted_at IS NULL` + RETURNING (repo izohiga qara).
	msg, err := uc.repo.SoftDelete(ctx, lessonID, messageID, mentorID)
	if err != nil {
		return err
	}
	uc.broadcastDeleted(ctx, msg, mentorID)
	uc.log.Info(ctx, "chat message deleted (moderation)",
		logger.String("lesson_id", lessonID), logger.String("message_id", messageID))
	return nil
}

// broadcastDeleted — jonli xonadagi klientlarga xabarni olib tashlashni aytadi.
//
// Manzil xabarning turiga qarab: ommaviy xabar butun xonaga, shaxsiysi esa
// FAQAT ikki tomonga. Aks holda o'chirish hodisasi shaxsiy yozishmaning
// MAVJUDLIGINI butun xonaga oshkor qilardi (mazmunini emas, lekin fakti ham
// maxfiy ma'lumot).
func (uc *useCase) broadcastDeleted(ctx context.Context, msg *entity.ChatMessage, deletedBy string) {
	if uc.livekit == nil || !uc.livekit.Enabled() || msg == nil {
		return
	}
	data, err := json.Marshal(entity.NewChatDeletedEvent(msg.ID, msg.LessonID, deletedBy))
	if err != nil {
		uc.log.Error(ctx, "chat: marshal delete event failed", logger.SafeString("err", err.Error()))
		return
	}
	room := shared.RoomName(msg.LessonID)
	if msg.ToIdentity != nil {
		err = uc.livekit.SendDataTo(ctx, room, data, []string{*msg.ToIdentity, msg.SenderIdentity})
	} else {
		err = uc.livekit.SendData(ctx, room, data)
	}
	if err != nil {
		// Tarqatish yiqilsa ham o'chirish AMALGA OSHGAN: xabar tarixdan chiqdi
		// va sahifa yangilanganda hech kimda ko'rinmaydi.
		uc.log.Warn(ctx, "chat: delete broadcast failed",
			logger.String("lesson_id", msg.LessonID), logger.SafeString("err", err.Error()))
	}
}

// ─── Xona yo'li (LiveKit room-token) ──────────────────────────────────────────

func (uc *useCase) SendFromRoom(ctx context.Context, lessonID, identity, name, body, to string) (*entity.ChatMessage, error) {
	// Dars jonli va yozuvchi chiqarilmagan bo'lishi shart. Avval bu ikkalasi ham
	// tekshirilmasdi: chiqarilgan buzg'unchi eski tokeni bilan chatga yozishda
	// davom eta olardi (so'rovnomada bu tekshiruv bor edi, chatda yo'q), yakunlangan
	// darsga ham xabar tushaverardi.
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, lessonID, identity); err != nil {
		return nil, err
	}
	if uc.cache != nil {
		n, err := uc.cache.Incr(ctx, chatRateKey(lessonID, identity), chatWindow)
		if err == nil && n > chatMax {
			return nil, apperr.TooManyRequests("too many messages — slow down")
		}
	}
	if name == "" {
		name = identity
	}
	msg, err := uc.build(ctx, lessonID, identity, name, body, to, nil)
	if err != nil {
		return nil, err
	}
	uc.log.Info(ctx, "chat message sent (participant)", logger.String("lesson_id", lessonID))
	return msg, nil
}

func (uc *useCase) UploadFromRoom(ctx context.Context, lessonID, identity, name string, f FileUpload, body, to string) (*entity.ChatMessage, error) {
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, lessonID, identity); err != nil {
		return nil, err
	}
	if name == "" {
		name = identity
	}
	return uc.upload(ctx, lessonID, identity, name, f, body, to)
}

func (uc *useCase) HistoryForRoom(ctx context.Context, lessonID, identity string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	// Chiqarilgan ishtirokchi tarixni ham o'qiy olmasligi kerak — aks holda u
	// darsni "chetdan kuzatib" turaverardi.
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, lessonID, identity); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	items, err := uc.repo.ListByLesson(ctx, lessonID, identity, before, limit)
	if err != nil {
		return nil, err
	}
	return uc.signAll(ctx, items), nil
}
