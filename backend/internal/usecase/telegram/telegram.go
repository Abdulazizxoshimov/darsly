package telegram

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	"github.com/zoom/darsly/internal/pkg/audit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// linkTTL — bog'lash kodining amal qilish muddati.
//
// 15 daqiqa: mentor ilovadan Telegramga o'tib `/start` bosishi uchun bemalol
// yetadi, lekin kod skrinshotda qolib ketsa uzoq yashamaydi. Kod aslida bir
// martalik parol.
const linkTTL = 15 * time.Minute

// linkKey — Redis kaliti. Kod → userID.
func linkKey(code string) string { return "tg:link:" + code }

// codeAlphabet — chalkashadigan belgilarsiz (0/O, 1/I/l yo'q).
//
// Kod QO'LDA ko'chiriladi (ekranda ko'rib, Telegramda terib) — shuning uchun
// `O` va `0` ni ajratib bo'lmaydigan alifbo qo'llab-quvvatlash savollarini
// keltirib chiqarardi.
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const codeLength = 8

type useCase struct {
	repo repository.TelegramRepository
	bot  tg.Client
	// cache — bir martalik kodlar. DB emas: kod qisqa umrli va uni tozalash
	// uchun yana bir ishchi yozish keraksiz ish bo'lardi (TTL buni bepul qiladi).
	cache redis.Cache
	log   logger.Logger
}

func New(
	repo repository.TelegramRepository,
	bot tg.Client,
	cache redis.Cache,
	log logger.Logger,
) UseCase {
	return &useCase{repo: repo, bot: bot, cache: cache, log: log}
}

func (uc *useCase) Enabled() bool { return uc.bot != nil && uc.bot.Enabled() }

// ─── Bog'lash oqimi (ilova tomoni) ───────────────────────────────────────────

func (uc *useCase) StartLink(ctx context.Context, userID string) (*entity.TelegramLink, error) {
	if !uc.Enabled() {
		return nil, apperr.BadRequest("telegram integration is not configured")
	}
	if uc.cache == nil {
		return nil, apperr.Internal(fmt.Errorf("telegram: cache is not available"))
	}

	// Kod TASODIFIY va SetNX bilan yoziladi: takrorlanish ehtimoli juda kichik
	// (32^8), lekin nolga teng emas va ikki mentorga bir kod berilsa ikkinchisi
	// birinchining hisobiga bog'lanib qolardi — bu jiddiy huquq buzilishi.
	for attempt := 0; attempt < 5; attempt++ {
		code, err := randomCode()
		if err != nil {
			return nil, apperr.Internal(fmt.Errorf("telegram: code: %w", err))
		}
		ok, err := uc.cache.SetNX(ctx, linkKey(code), userID, linkTTL)
		if err != nil {
			return nil, apperr.Internal(fmt.Errorf("telegram: link store: %w", err))
		}
		if !ok {
			continue // band — yangisini urinamiz
		}
		link := &entity.TelegramLink{Code: code, ExpiresInS: int(linkTTL.Seconds())}
		if u := uc.bot.BotUsername(); u != "" {
			link.DeepLink = "https://t.me/" + u + "?start=" + code
		}
		return link, nil
	}
	return nil, apperr.Internal(fmt.Errorf("telegram: could not allocate link code"))
}

func (uc *useCase) Status(ctx context.Context, userID string) (*entity.TelegramLinkStatus, error) {
	st := &entity.TelegramLinkStatus{Enabled: uc.Enabled()}
	if !st.Enabled {
		// Integratsiya o'chiq — DB'ga bormaymiz ham. Klient uchun javob
		// aniq: "bu serverda Telegram yo'q".
		return st, nil
	}
	u, err := uc.repo.GetLink(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.TelegramUserID == nil {
		return st, nil
	}
	st.Linked = true
	st.Username = u.TelegramUsername
	st.LinkedAt = u.TelegramLinkedAt

	chats, err := uc.repo.ListChatsByMentor(ctx, userID)
	if err != nil {
		// Guruhlar ro'yxati bo'lmasa ham bog'lanish holati to'g'ri — asosiy
		// javobni xato bilan almashtirmaymiz.
		uc.log.Warn(ctx, "telegram.Status: guruhlar ro'yxati o'qilmadi",
			logger.SafeString("err", err.Error()))
		return st, nil
	}
	st.Chats = chats
	return st, nil
}

func (uc *useCase) Unlink(ctx context.Context, userID string) error {
	if err := uc.repo.UnlinkUser(ctx, userID); err != nil {
		return err
	}
	audit.Record(ctx, uc.log, "telegram.unlink", userID)
	return nil
}

// ─── Bot tomoni ──────────────────────────────────────────────────────────────

func (uc *useCase) RedeemCode(ctx context.Context, code string, telegramUserID int64, username string) (*entity.User, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, apperr.BadRequest("code is required")
	}
	if uc.cache == nil {
		return nil, apperr.Internal(fmt.Errorf("telegram: cache is not available"))
	}
	userID, err := uc.cache.Get(ctx, linkKey(code))
	if err != nil || userID == "" {
		// Muddati o'tgan, ishlatilgan yoki umuman bo'lmagan kod — bir xil
		// javob. Farqlash kod maydonini taxmin qilishga yordam berardi.
		return nil, apperr.BadRequest("kod yaroqsiz yoki muddati o'tgan")
	}
	// ⭐ Kod BIR MARTALIK: bog'lashdan OLDIN o'chiriladi.
	//
	// Tartib muhim. Teskarisi (avval bog'lab, keyin o'chirish) bo'lsa,
	// o'chirish uzilganda kod amal qilib qolardi va uni ko'rgan boshqa odam
	// o'sha mentorga bog'lanib, uning yozuvlarini o'z guruhiga yubora olardi.
	// Bu tartibda eng yomon holat — mentor kodni qayta so'rashi.
	if err := uc.cache.Del(ctx, linkKey(code)); err != nil {
		return nil, apperr.Internal(fmt.Errorf("telegram: code consume: %w", err))
	}

	if err := uc.repo.LinkUser(ctx, userID, telegramUserID, username); err != nil {
		return nil, err
	}
	u, err := uc.repo.GetLink(ctx, userID)
	if err != nil {
		return nil, err
	}
	audit.Record(ctx, uc.log, "telegram.link", userID,
		logger.String("telegram_username", username))
	return u, nil
}

func (uc *useCase) HandleChatMembership(ctx context.Context, chat entity.TelegramChat, byTelegramUserID int64, joined bool) error {
	if !joined {
		return uc.repo.DeactivateChat(ctx, chat.ChatID)
	}
	// Botni QO'SHGAN odam bog'langan mentor bo'lsa, guruh unga biriktiriladi.
	// Bog'lanmagan bo'lsa `mentor_id` NULL qoladi va UpsertChat mavjud egani
	// saqlaydi — guruh "egasiz" bo'lib qolmaydi.
	if u, err := uc.repo.GetUserByTelegramID(ctx, byTelegramUserID); err == nil && u != nil {
		chat.MentorID = &u.ID
	}
	return uc.repo.UpsertChat(ctx, &chat)
}

func (uc *useCase) MentorByTelegramID(ctx context.Context, telegramUserID int64) (*entity.User, error) {
	return uc.repo.GetUserByTelegramID(ctx, telegramUserID)
}

func (uc *useCase) TelegramIDOf(ctx context.Context, mentorID string) (int64, bool, error) {
	u, err := uc.repo.GetLink(ctx, mentorID)
	if err != nil {
		return 0, false, err
	}
	if u.TelegramUserID == nil {
		return 0, false, nil
	}
	return *u.TelegramUserID, true, nil
}

func (uc *useCase) ChatsForMentor(ctx context.Context, mentorID string) ([]*entity.TelegramChat, error) {
	return uc.repo.ListChatsByMentor(ctx, mentorID)
}

func (uc *useCase) AuthorizeChat(ctx context.Context, mentorID string, chatID int64) (*entity.TelegramChat, error) {
	chat, err := uc.repo.GetChat(ctx, chatID)
	if err != nil {
		return nil, err
	}
	// ⭐ Egalik tekshiruvi. `callback_data` — bosgan odam o'zgartira oladigan
	// satr (Telegram uni imzolamaydi va butunligini kafolatlamaydi). Tekshiruv
	// bo'lmasa har kim istalgan chat ID ni yozib, begona guruhga video
	// yuborishga majburlay olardi.
	if chat.MentorID == nil || *chat.MentorID != mentorID {
		return nil, apperr.Forbidden("bu guruhga yuborish huquqi yo'q")
	}
	if !chat.IsActive {
		return nil, apperr.BadRequest("bot bu guruhdan chiqarilgan")
	}
	return chat, nil
}

// randomCode — `crypto/rand` bilan (math/rand EMAS: kod ruxsat beruvchi sir).
func randomCode() (string, error) {
	b := make([]byte, codeLength)
	max := big.NewInt(int64(len(codeAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = codeAlphabet[n.Int64()]
	}
	return string(b), nil
}
