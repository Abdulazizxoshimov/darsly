package worker

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	tguc "github.com/zoom/darsly/internal/usecase/telegram"
)

// ⭐ NEGA LONG-POLLING, WEBHOOK EMAS
//
// Webhook tezroq va "to'g'riroq" ko'rinadi, lekin bizning muhitda uch narsani
// talab qiladi:
//  1. ochiq HTTPS endpoint va Telegram'ning IP diapazonidan kelgan so'rovni
//     tekshirish (`secret_token`) — yana bitta autentifikatsiya yuzasi;
//  2. Caddy'da yangi route va sertifikat bog'liqligi — deploy paytida
//     bot jimgina o'lik qolishi mumkin bo'lgan yana bir joy;
//  3. Local Bot API Server bilan webhook manzili ichki tarmoqqa ko'rsatiladi,
//     ya'ni tashqi Telegram undan foydalana olmaydi — konfiguratsiya ikki
//     rejimda ikki xil bo'lardi.
//
// Long-polling esa CHIQUVCHI ulanish: hech qanday port ochilmaydi, sertifikat
// kerak emas va u ikkala rejimda (rasmiy API va local server) bir xil
// ishlaydi. Yuk ham arzimas — kuniga o'nlab yangilanish.
//
// YAGONA cheklov: `getUpdates` ni bir vaqtda FAQAT bitta iste'molchi
// chaqirishi mumkin (ikkinchisi birinchisining yangilanishlarini o'g'irlaydi).
// Shuning uchun ishchi Redis'dagi liderlik qulfi ostida ishlaydi.

// botLockKey — long-polling liderlik qulfi (ko'p-instans uchun).
const botLockKey = "tg:bot:leader"

// botOffsetKey — oxirgi qayta ishlangan `update_id`.
const botOffsetKey = "tg:bot:offset"

const (
	// botPollTimeout — Telegram javobni shuncha soniya ushlab turadi.
	botPollTimeout = 30
	// botLockTTL — qulf muddati. Poll timeout'idan uzun bo'lishi SHART, aks
	// holda uzun so'rov davomida qulf bo'shab, ikkinchi instans ham polling
	// boshlab yuborardi.
	botLockTTL = 90 * time.Second
	// botLockRenew — qulfni yangilash oralig'i.
	botLockRenew = 30 * time.Second
)

// Callback-data prefikslari.
//
// Qisqa, chunki Telegram `callback_data` ni 64 BAYT bilan cheklaydi:
// `s|<uuid-36>|<chat_id-14>` = 53 bayt — sig'adi. To'liq so'zlar bilan
// ("share_recording") sig'masdi va serverda holat saqlashga to'g'ri kelardi.
const (
	cbShare   = "s" // s|<recording_id>|<chat_id> — videoni guruhga yuborish
	cbSkip    = "x" // x|<recording_id>          — yubormayman
	cbChatYes = "c" // c|<recording_id>|<chat_id> — chat transkriptini ham
	cbChatNo  = "n" // n|<recording_id>          — chat kerak emas
)

// TranscriptSource — chat transkriptini yasaydigan manba.
//
// Interfeys, chunki transkript formati chat domenining ishi va u alohida
// rivojlanadi; bu ishchi faqat "baytlar + fayl nomi" ni biladi.
type TranscriptSource interface {
	// Transcript — dars chat tarixini fayl sifatida qaytaradi.
	// Xabar bo'lmasa `nil, "", nil` (yuborishga narsa yo'q).
	Transcript(ctx context.Context, lesson *entity.Lesson) ([]byte, string, error)
}

// TelegramBotWorker — bot yangilanishlarini qayta ishlaydi.
type TelegramBotWorker struct {
	bot        tg.Client
	uc         tguc.UseCase
	recRepo    repository.RecordingRepository
	lessonRepo repository.LessonRepository
	cache      redis.Cache
	transcript TranscriptSource
	log        logger.Logger
}

func NewTelegramBotWorker(
	bot tg.Client,
	uc tguc.UseCase,
	recRepo repository.RecordingRepository,
	lessonRepo repository.LessonRepository,
	cache redis.Cache,
	transcript TranscriptSource,
	log logger.Logger,
) *TelegramBotWorker {
	return &TelegramBotWorker{
		bot: bot, uc: uc, recRepo: recRepo, lessonRepo: lessonRepo,
		cache: cache, transcript: transcript, log: log,
	}
}

func (w *TelegramBotWorker) Run(ctx context.Context) {
	if w.bot == nil || !w.bot.Enabled() {
		w.log.Info(ctx, "telegram bot worker disabled (TELEGRAM_BOT_TOKEN bo'sh)")
		return
	}
	w.log.Info(ctx, "telegram bot worker started (long-polling)",
		logger.String("bot", w.bot.BotUsername()))

	offset := w.loadOffset(ctx)
	lockID := strconv.FormatInt(time.Now().UnixNano(), 36)
	var lockedUntil time.Time

	for {
		if ctx.Err() != nil {
			return
		}
		// Liderlik: faqat bitta instans `getUpdates` chaqiradi.
		if !w.holdLock(ctx, lockID, &lockedUntil) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(botLockRenew):
			}
			continue
		}

		ups, err := w.bot.GetUpdates(ctx, offset, botPollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			wait := 5 * time.Second
			if after, ok := tg.IsRetryAfter(err); ok {
				wait = after
			}
			w.log.Warn(ctx, "telegram bot: getUpdates failed",
				logger.SafeString("err", err.Error()), logger.String("retry_in", wait.String()))
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
			continue
		}

		for _, u := range ups {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			// Bitta buzuq yangilanish butun tsiklni to'xtatmasin: har biri
			// alohida qayta ishlanadi va xato faqat log'ga tushadi. Aks holda
			// bitta noto'g'ri callback bot'ni butunlay to'xtatib qo'yardi.
			w.handle(ctx, u)
		}
		if len(ups) > 0 {
			w.saveOffset(ctx, offset)
		}
	}
}

// holdLock — liderlik qulfini oladi/yangilaydi.
//
// Redis yo'q bo'lsa `true` qaytaradi: bitta instansli o'rnatishda (hozirgi
// holat) Redis'siz ham bot ishlashi kerak. Ko'p-instansda Redis baribir bor.
//
// ⚠️ Uzaytirish (`Get` + `Set`) atomik EMAS — nazariy jihatdan ikki instans
// qisqa vaqt liderlikni bo'lishishi mumkin. Bu ATAYLAB soddalashtirilgan:
// eng yomon oqibat — bir yangilanish ikki marta qayta ishlanishi, va uning
// har ikkala natijasi ham idempotent (`UpsertChat`, `ClaimRestore`, kodni
// bir martalik iste'mol qilish). Lua bilan to'liq atomik qulf bu xavf
// darajasiga arzimaydi.
func (w *TelegramBotWorker) holdLock(ctx context.Context, id string, until *time.Time) bool {
	if w.cache == nil {
		return true
	}
	if time.Now().Before(*until) {
		return true // hali amal qiladi
	}
	ok, err := w.cache.AcquireLock(ctx, botLockKey, id, botLockTTL)
	if err != nil {
		w.log.Warn(ctx, "telegram bot: qulfni olib bo'lmadi — davom etamiz",
			logger.SafeString("err", err.Error()))
		return true
	}
	if !ok {
		// Qulf bizniki bo'lishi mumkin (oldingi tsikldan) — uzaytirishga urinamiz.
		if v, gerr := w.cache.Get(ctx, botLockKey); gerr == nil && v == id {
			_ = w.cache.Set(ctx, botLockKey, id, botLockTTL)
			*until = time.Now().Add(botLockRenew)
			return true
		}
		return false
	}
	*until = time.Now().Add(botLockRenew)
	return true
}

// loadOffset — oxirgi qayta ishlangan `update_id + 1`.
//
// Saqlanmagan bo'lsa BACKLOG TASHLANADI (`offset = -1` bilan bitta so'rov):
// bot bir necha kun o'chiq turgan bo'lsa Telegram to'plagan eski
// yangilanishlarni qaytaradi va bot eski darslar uchun "guruhga yuboraymi?"
// deb so'ray boshlardi — mentor uchun bu chalkashlik.
func (w *TelegramBotWorker) loadOffset(ctx context.Context) int64 {
	if w.cache != nil {
		if v, err := w.cache.Get(ctx, botOffsetKey); err == nil && v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				return n
			}
		}
	}
	ups, err := w.bot.GetUpdates(ctx, -1, 0)
	if err != nil || len(ups) == 0 {
		return 0
	}
	next := ups[len(ups)-1].UpdateID + 1
	w.saveOffset(ctx, next)
	return next
}

func (w *TelegramBotWorker) saveOffset(ctx context.Context, offset int64) {
	if w.cache == nil {
		return
	}
	// TTL yo'q emas, uzun: offset yo'qolsa backlog qayta o'qilardi.
	if err := w.cache.Set(ctx, botOffsetKey, strconv.FormatInt(offset, 10), 30*24*time.Hour); err != nil {
		w.log.Warn(ctx, "telegram bot: offset saqlanmadi", logger.SafeString("err", err.Error()))
	}
}

// ─── Yangilanishlarni qayta ishlash ──────────────────────────────────────────

func (w *TelegramBotWorker) handle(ctx context.Context, u tg.Update) {
	// ⭐ PANIKA USHLANADI. Bu yerga TASHQI, ishonchsiz ma'lumot keladi
	// (istalgan odam botga istalgan narsa yuborishi mumkin). Go'da gorutinadagi
	// panika BUTUN JARAYONNI o'ldiradi — ya'ni bitta buzuq xabar jonli darsni
	// ham uzib qo'yardi. Bot — platformaning eng chekka va eng kam muhim
	// qismi; u hech qachon o'zagini yiqita olmasligi kerak.
	defer func() {
		if r := recover(); r != nil {
			w.log.Error(ctx, "telegram bot: yangilanishda panika (yutildi)",
				logger.SafeString("panic", fmt.Sprint(r)))
		}
	}()

	// Har yangilanish uchun cheklangan muddat: transkript yuborish yoki
	// video ulashish uzoq bo'lishi mumkin, lekin cheksiz emas.
	hctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	switch {
	case u.MyChatMember != nil:
		w.handleChatMember(hctx, u.MyChatMember)
	case u.CallbackQuery != nil:
		w.handleCallback(hctx, u.CallbackQuery)
	case u.Message != nil:
		w.handleMessage(hctx, u.Message)
	}
}

// handleChatMember — bot guruhga qo'shildi yoki chiqarildi.
func (w *TelegramBotWorker) handleChatMember(ctx context.Context, ev *tg.ChatMemberUpdated) {
	if ev.Chat.Type == "private" {
		return // shaxsiy chat guruh emas
	}
	joined := ev.NewChatMember.IsMember()
	chat := entity.TelegramChat{
		ChatID: ev.Chat.ID,
		Title:  ev.Chat.Title,
		Type:   ev.Chat.Type,
	}
	if err := w.uc.HandleChatMembership(ctx, chat, ev.From.ID, joined); err != nil {
		w.log.Warn(ctx, "telegram bot: guruh holati yozilmadi",
			logger.String("chat_id", strconv.FormatInt(ev.Chat.ID, 10)),
			logger.SafeString("err", err.Error()))
		return
	}
	w.log.Info(ctx, "telegram bot: guruh holati yangilandi",
		logger.String("chat_id", strconv.FormatInt(ev.Chat.ID, 10)),
		logger.String("joined", strconv.FormatBool(joined)))

	// Qo'shgan odam bog'langan mentor bo'lsa — tasdiq. Bu muhim UX detali:
	// aks holda mentor botni guruhga qo'shib, "ishladimi?" deb bilmay qolardi.
	if joined {
		if u, err := w.uc.MentorByTelegramID(ctx, ev.From.ID); err == nil && u != nil {
			w.send(ctx, ev.From.ID, fmt.Sprintf(
				"«%s» guruhi qo'shildi. Endi dars tugagach yozuvni shu guruhga yuborishni taklif qilaman.",
				ev.Chat.Title), nil)
		} else {
			w.send(ctx, ev.From.ID,
				"Guruh qo'shildi, lekin Telegram akkauntingiz Darsly hisobiga bog'lanmagan. "+
					"Ilovada «Telegram bilan bog'lash» ni oching va kodni menga /start <kod> ko'rinishida yuboring.", nil)
		}
	}
}

// handleMessage — shaxsiy chatdagi buyruqlar.
func (w *TelegramBotWorker) handleMessage(ctx context.Context, m *tg.Message) {
	if m.Chat.Type != "private" || m.From == nil {
		// Guruhdagi xabarlarga javob bermaymiz: bot u yerda faqat video
		// tashlaydi va suhbatga aralashishi shovqin bo'lardi.
		return
	}
	text := strings.TrimSpace(m.Text)
	switch {
	case strings.HasPrefix(text, "/start"):
		code := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
		if code == "" {
			w.send(ctx, m.Chat.ID,
				"Salom! Men Darsly dars yozuvlarini saqlaydigan botman.\n\n"+
					"Boshlash uchun: ilovada «Telegram bilan bog'lash» ni oching va chiqqan kodni menga shunday yuboring:\n"+
					"/start ABCD2345", nil)
			return
		}
		u, err := w.uc.RedeemCode(ctx, code, m.From.ID, m.From.Username)
		if err != nil {
			w.send(ctx, m.Chat.ID, "Kod yaroqsiz yoki muddati o'tgan. Ilovadan yangi kod oling.", nil)
			return
		}
		w.send(ctx, m.Chat.ID, fmt.Sprintf(
			"Tayyor, %s! Hisobingiz bog'landi.\n\n"+
				"Endi: meni o'quvchilaringiz guruhiga admin qilib qo'shing — dars tugagach yozuvni o'sha guruhga yuborishni taklif qilaman.",
			u.FullName), nil)
	case text == "/chats":
		w.replyChats(ctx, m)
	case text == "/help":
		w.send(ctx, m.Chat.ID,
			"Buyruqlar:\n"+
				"/start <kod> — hisobni bog'lash\n"+
				"/chats — men a'zo bo'lgan guruhlaringiz\n\n"+
				"Dars tugagach yozuvni avtomatik arxivlayman va sizdan qaysi guruhga yuborishni so'rayman.", nil)
	}
}

func (w *TelegramBotWorker) replyChats(ctx context.Context, m *tg.Message) {
	u, err := w.uc.MentorByTelegramID(ctx, m.From.ID)
	if err != nil || u == nil {
		w.send(ctx, m.Chat.ID, "Avval hisobingizni bog'lang: /start <kod>", nil)
		return
	}
	chats, err := w.uc.ChatsForMentor(ctx, u.ID)
	if err != nil || len(chats) == 0 {
		w.send(ctx, m.Chat.ID, "Hozircha guruh yo'q. Meni guruhga admin qilib qo'shing.", nil)
		return
	}
	var b strings.Builder
	b.WriteString("Guruhlaringiz:\n")
	for _, c := range chats {
		b.WriteString("· " + c.Title + "\n")
	}
	w.send(ctx, m.Chat.ID, b.String(), nil)
}

// handleCallback — inline tugma bosildi.
func (w *TelegramBotWorker) handleCallback(ctx context.Context, cb *tg.CallbackQuery) {
	parts := strings.Split(cb.Data, "|")
	if len(parts) == 0 {
		return
	}
	// ⭐ Tugmani bosgan odamni HAR DOIM aniqlaymiz. `callback_data` ni
	// o'zgartirish mumkin, `from.id` ni esa yo'q — u Telegram tomonidan
	// beriladi. Butun ruxsat modeli shunga tayanadi.
	mentor, err := w.uc.MentorByTelegramID(ctx, cb.From.ID)
	if err != nil || mentor == nil {
		w.answer(ctx, cb.ID, "Hisobingiz bog'lanmagan")
		return
	}

	switch parts[0] {
	case cbSkip:
		w.answer(ctx, cb.ID, "Yaxshi, yubormadim")
		w.editDone(ctx, cb, "Yozuv faqat arxivda saqlandi.")
	case cbChatNo:
		w.answer(ctx, cb.ID, "Yaxshi")
		w.editDone(ctx, cb, "Video yuborildi. Chat tarixi yuborilmadi.")
	case cbShare:
		if len(parts) != 3 {
			w.answer(ctx, cb.ID, "Noto'g'ri so'rov")
			return
		}
		w.shareVideo(ctx, cb, mentor, parts[1], parts[2])
	case cbChatYes:
		if len(parts) != 3 {
			w.answer(ctx, cb.ID, "Noto'g'ri so'rov")
			return
		}
		w.shareTranscript(ctx, cb, mentor, parts[1], parts[2])
	}
}

// shareVideo — arxivdagi videoni mentor tanlagan guruhga yuboradi.
//
// ⭐ Fayl QAYTA YUKLANMAYDI: `file_id` bilan yuboriladi. Telegram bir marta
// yuklangan faylni ID bo'yicha istalgan chatga bepul va bir zumda ko'chiradi.
// Qayta yuklash 1 GB trafik va o'nlab daqiqa bo'lardi — mentor tugmani bosib,
// hech narsa bo'lmayotgandek kutib turardi.
func (w *TelegramBotWorker) shareVideo(ctx context.Context, cb *tg.CallbackQuery, mentor *entity.User, recID, chatIDStr string) {
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		w.answer(ctx, cb.ID, "Noto'g'ri guruh")
		return
	}
	if _, err := w.uc.AuthorizeChat(ctx, mentor.ID, chatID); err != nil {
		w.answer(ctx, cb.ID, "Bu guruhga yuborish huquqi yo'q")
		return
	}
	rec, l, err := w.ownedRecording(ctx, mentor.ID, recID)
	if err != nil {
		w.answer(ctx, cb.ID, "Yozuv topilmadi")
		return
	}
	if rec.TelegramFileID == nil || *rec.TelegramFileID == "" {
		w.answer(ctx, cb.ID, "Yozuv hali arxivlanmagan")
		return
	}

	w.answer(ctx, cb.ID, "Yuborilmoqda…")
	// Mentorning O'Z o'quvchilar guruhiga ulashish — mentor nomi ortiqcha (bo'sh).
	if _, err := w.bot.SendVideoByFileID(ctx, chatID, *rec.TelegramFileID, archiveCaption(rec, l, "")); err != nil {
		w.log.Warn(ctx, "telegram bot: guruhga yuborilmadi",
			logger.String("recording_id", recID), logger.SafeString("err", err.Error()))
		w.editDone(ctx, cb, "Yuborib bo'lmadi: "+truncateErr(err.Error(), 200))
		return
	}
	// Ikkinchi qadam: chat transkripti ham kerakmi.
	kb := &tg.InlineKeyboard{InlineKeyboard: [][]tg.InlineButton{{
		{Text: "Ha, chat ham", CallbackData: cbChatYes + "|" + recID + "|" + chatIDStr},
		{Text: "Yo'q", CallbackData: cbChatNo + "|" + recID},
	}}}
	w.editKeyboard(ctx, cb, "Video guruhga yuborildi.\n\nChat tarixi ham yuborilsinmi?", kb)
}

// shareTranscript — dars chat tarixini fayl sifatida guruhga yuboradi.
//
// FAQAT OMMAVIY xabarlar: `TranscriptSource` shaxsiy (private) xabarlarni
// qaytarmaydi. Aks holda o'quvchining mentorga yozgan shaxsiy savoli butun
// guruhga tarqalardi — bu maxfiylik buzilishi bo'lardi.
func (w *TelegramBotWorker) shareTranscript(ctx context.Context, cb *tg.CallbackQuery, mentor *entity.User, recID, chatIDStr string) {
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		w.answer(ctx, cb.ID, "Noto'g'ri guruh")
		return
	}
	if _, err := w.uc.AuthorizeChat(ctx, mentor.ID, chatID); err != nil {
		w.answer(ctx, cb.ID, "Bu guruhga yuborish huquqi yo'q")
		return
	}
	_, l, err := w.ownedRecording(ctx, mentor.ID, recID)
	if err != nil || l == nil {
		w.answer(ctx, cb.ID, "Dars topilmadi")
		return
	}
	if w.transcript == nil {
		w.answer(ctx, cb.ID, "Chat tarixi mavjud emas")
		return
	}

	w.answer(ctx, cb.ID, "Tayyorlanmoqda…")
	body, name, err := w.transcript.Transcript(ctx, l)
	if err != nil {
		w.log.Warn(ctx, "telegram bot: transkript yasalmadi", logger.SafeString("err", err.Error()))
		w.editDone(ctx, cb, "Chat tarixini tayyorlab bo'lmadi.")
		return
	}
	if len(body) == 0 {
		w.editDone(ctx, cb, "Video yuborildi. Bu darsda chat xabari bo'lmagan.")
		return
	}
	if _, err := w.bot.SendDocument(ctx, chatID, name, bytes.NewReader(body), int64(len(body)),
		"«"+l.Title+"» — chat tarixi"); err != nil {
		w.log.Warn(ctx, "telegram bot: transkript yuborilmadi", logger.SafeString("err", err.Error()))
		w.editDone(ctx, cb, "Chat tarixini yuborib bo'lmadi.")
		return
	}
	w.editDone(ctx, cb, "Video va chat tarixi guruhga yuborildi.")
}

// ownedRecording — yozuv + dars, EGALIK tekshiruvi bilan.
//
// Telegram tomonida ham egalik tekshiriladi (HTTP tomonida qilinganidek):
// `callback_data` dagi recording ID ni istalgan UUID bilan almashtirish
// mumkin va tekshiruvsiz begona darsning yozuvi tarqalib ketardi.
func (w *TelegramBotWorker) ownedRecording(ctx context.Context, mentorID, recID string) (*entity.Recording, *entity.Lesson, error) {
	rec, err := w.recRepo.GetByID(ctx, recID)
	if err != nil {
		return nil, nil, err
	}
	l, err := w.lessonRepo.GetByID(ctx, rec.LessonID)
	if err != nil {
		return nil, nil, err
	}
	if l.MentorID != mentorID {
		return nil, nil, apperr.Forbidden("not your recording")
	}
	return rec, l, nil
}

// ─── TelegramPrompter: yuklash ishchisi chaqiradi ────────────────────────────

// PromptShare — arxivlash tugagach mentorga guruh tanlash so'rovi.
//
// Guruh yo'q bo'lsa hech narsa yuborilmaydi: "guruhingiz yo'q" degan xabar
// har darsdan keyin takrorlanib, spam bo'lardi. Mentor botni guruhga
// qo'shganda esa tasdiq xabarini oladi (`handleChatMember`).
func (w *TelegramBotWorker) PromptShare(ctx context.Context, rec *entity.Recording, l *entity.Lesson) error {
	if w.bot == nil || !w.bot.Enabled() || l == nil {
		return nil
	}
	tgID, linked, err := w.uc.TelegramIDOf(ctx, l.MentorID)
	if err != nil || !linked {
		// Mentor Telegramni bog'lamagan — so'rash mumkin emas. Bu xato emas:
		// video baribir arxiv guruhida.
		return nil
	}
	chats, err := w.uc.ChatsForMentor(ctx, l.MentorID)
	if err != nil {
		return err
	}
	if len(chats) == 0 {
		return nil
	}

	rows := make([][]tg.InlineButton, 0, len(chats)+1)
	for _, c := range chats {
		rows = append(rows, []tg.InlineButton{{
			Text:         c.Title,
			CallbackData: cbShare + "|" + rec.ID + "|" + strconv.FormatInt(c.ChatID, 10),
		}})
	}
	rows = append(rows, []tg.InlineButton{{Text: "Yubormayman", CallbackData: cbSkip + "|" + rec.ID}})

	text := fmt.Sprintf("«%s» darsi tugadi va yozuv arxivga saqlandi.\n\nO'quvchilar guruhiga ham yuboraymi?", l.Title)
	_, err = w.bot.SendMessage(ctx, tgID, text, &tg.InlineKeyboard{InlineKeyboard: rows})
	return err
}

// ─── Kichik yordamchilar ─────────────────────────────────────────────────────

func (w *TelegramBotWorker) send(ctx context.Context, chatID int64, text string, kb *tg.InlineKeyboard) {
	if _, err := w.bot.SendMessage(ctx, chatID, text, kb); err != nil {
		w.log.Warn(ctx, "telegram bot: xabar yuborilmadi", logger.SafeString("err", err.Error()))
	}
}

func (w *TelegramBotWorker) answer(ctx context.Context, id, text string) {
	if err := w.bot.AnswerCallbackQuery(ctx, id, text); err != nil {
		w.log.Warn(ctx, "telegram bot: callback javobi ketmadi", logger.SafeString("err", err.Error()))
	}
}

// editDone — tugmalarni olib tashlab, yakuniy matnni qo'yadi.
//
// Tugmalar OLIB TASHLANISHI muhim: aks holda mentor o'sha xabarga qaytib
// yana bosaverardi va guruhga dublikat video ketardi.
func (w *TelegramBotWorker) editDone(ctx context.Context, cb *tg.CallbackQuery, text string) {
	w.editKeyboard(ctx, cb, text, nil)
}

func (w *TelegramBotWorker) editKeyboard(ctx context.Context, cb *tg.CallbackQuery, text string, kb *tg.InlineKeyboard) {
	if cb.Message == nil {
		return
	}
	if err := w.bot.EditMessageText(ctx, cb.Message.Chat.ID, cb.Message.MessageID, text, kb); err != nil {
		w.log.Warn(ctx, "telegram bot: xabarni tahrirlab bo'lmadi", logger.SafeString("err", err.Error()))
	}
}

// ─── Chat transkripti (default implementatsiya) ──────────────────────────────

// ChatTranscript — `ChatRepository` dan oddiy matnli transkript yasaydi.
//
// Format TXT: Telegram uni ilovaning o'zida ochib ko'rsatadi (HTML esa
// yuklab olishni talab qiladi), va u har qanday qurilmada o'qiladi.
type ChatTranscript struct {
	repo repository.ChatRepository
	// Limit — nechta xabar olinadi. Cheklov bor, chunki 4 soatlik darsda
	// minglab xabar bo'lishi mumkin va ularning hammasi Telegram fayl
	// chegarasiga urilardi.
	Limit int
}

func NewChatTranscript(repo repository.ChatRepository) *ChatTranscript {
	return &ChatTranscript{repo: repo, Limit: 5000}
}

func (t *ChatTranscript) Transcript(ctx context.Context, l *entity.Lesson) ([]byte, string, error) {
	if t.repo == nil || l == nil {
		return nil, "", nil
	}
	limit := t.Limit
	if limit <= 0 {
		limit = 5000
	}
	// viewerIdentity BO'SH — bu ataylab: repozitoriy shunda FAQAT ommaviy
	// xabarlarni qaytaradi. Shaxsiy xabarlar guruhga hech qachon tushmasin.
	msgs, err := t.repo.ListByLesson(ctx, l.ID, "", nil, limit)
	if err != nil {
		return nil, "", err
	}
	if len(msgs) == 0 {
		return nil, "", nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s — chat tarixi\n", l.Title)
	fmt.Fprintf(&b, "%s\n\n", lessonWhen(l).Format("2006-01-02 15:04"))
	// Repozitoriy eng yangisidan beradi (created_at DESC) — o'qish uchun
	// teskarisiga aylantiramiz.
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		line := fmt.Sprintf("[%s] %s: %s", m.CreatedAt.Format("15:04:05"), m.SenderName, m.Body)
		if m.File != nil {
			line += fmt.Sprintf(" (fayl: %s)", m.File.Name)
		}
		b.WriteString(line + "\n")
	}
	name := "chat-" + lessonWhen(l).Format("2006-01-02") + ".txt"
	return []byte(b.String()), name, nil
}

// lessonWhen — dars qachon bo'lgani. Uch manba, aniqligi bo'yicha:
// haqiqatan boshlangani → rejalashtirilgani → yaratilgani. Faqat
// `CreatedAt` ga tayanish noto'g'ri sana berardi (dars bir hafta oldin
// yaratilib, kecha o'tkazilgan bo'lishi mumkin).
func lessonWhen(l *entity.Lesson) time.Time {
	if l.StartedAt != nil {
		return *l.StartedAt
	}
	if l.ScheduledAt != nil {
		return *l.ScheduledAt
	}
	return l.CreatedAt
}
