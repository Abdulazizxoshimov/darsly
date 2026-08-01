package telegram

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// UseCase — Telegram integratsiyasining biznes qoidalari.
//
// Ikki chaqiruvchisi bor va ular butunlay boshqa yo'ldan keladi:
//   - HTTP (mentor ilovada «Telegram bilan bog'lash» ni bosadi);
//   - bot long-polling ishchisi (mentor Telegramda tugma bosadi).
//
// Ikkalasi ham SHU interfeysdan o'tadi, ya'ni ruxsat qoidalari bitta joyda.
// Bot tomonini alohida yozish vasvasasi bor edi — o'shanda «kim nimani
// yubora oladi» tekshiruvi ikki nusxada bo'lardi va ulardan biri albatta
// eskirardi.
type UseCase interface {
	// Enabled — integratsiya sozlanganmi (TELEGRAM_BOT_TOKEN bor).
	//
	// Klient shu bayroqqa qarab UI'ni ko'rsatadi: sozlanmagan serverda
	// «Telegram bilan bog'lash» tugmasi umuman chiqmasin.
	Enabled() bool

	// StartLink — bir martalik bog'lash kodini yaratadi (mentor ilovada).
	//
	// Kod Redis'da qisqa TTL bilan yashaydi: u aslida bir martalik parol va
	// DB'da abadiy qolishi kerak emas. Redis yo'q bo'lsa xato — bu holda
	// bog'lashni "amalga oshdi" deb ko'rsatish yolg'on bo'lardi.
	StartLink(ctx context.Context, userID string) (*entity.TelegramLink, error)
	// Status — bog'lanish holati + mentorning guruhlari (ilova ko'rsatadi).
	Status(ctx context.Context, userID string) (*entity.TelegramLinkStatus, error)
	// Unlink — bog'lanishni uzadi. Idempotent (bog'lanmagan bo'lsa ham OK).
	Unlink(ctx context.Context, userID string) error

	// ── Bot tomoni ──────────────────────────────────────────────────────────

	// RedeemCode — mentor botga `/start <kod>` yubordi.
	//
	// Kod bir MARTALIK: muvaffaqiyatli almashinuvdan keyin Redis'dan
	// o'chiriladi. Aks holda kod skrinshotga tushib, boshqa odam o'sha
	// mentorning yozuvlarini o'z guruhiga yubora olardi.
	RedeemCode(ctx context.Context, code string, telegramUserID int64, username string) (*entity.User, error)
	// HandleChatMembership — bot guruhga qo'shildi/chiqarildi
	// (`my_chat_member`). Bu bot qaysi guruhlarda ekanini bilishning YAGONA
	// yo'li — Bot API'da "mening guruhlarim" metodi yo'q.
	HandleChatMembership(ctx context.Context, chat entity.TelegramChat, byTelegramUserID int64, joined bool) error
	// MentorByTelegramID — tugma bosgan odam kim (bog'lanmagan bo'lsa NotFound).
	MentorByTelegramID(ctx context.Context, telegramUserID int64) (*entity.User, error)
	// TelegramIDOf — TESKARI yo'nalish: mentorning Telegram akkaunt ID si.
	//
	// Bot mentorga o'zi murojaat qilganda kerak («dars tugadi, guruhga
	// yuboraymi?»). Bog'lanmagan bo'lsa `0, false, nil` — bu XATO EMAS:
	// mentor Telegramni ulashga majbur emas va bunday holatda bot jim turadi.
	TelegramIDOf(ctx context.Context, mentorID string) (int64, bool, error)
	// ChatsForMentor — mentorning faol guruhlari (bot inline tugmalar yasaydi).
	ChatsForMentor(ctx context.Context, mentorID string) ([]*entity.TelegramChat, error)
	// AuthorizeChat — mentor shu guruhga yubora oladimi.
	//
	// Bot tugmasi bosilganda MAJBURIY: `callback_data` klient tomonidan
	// o'zgartirilishi mumkin bo'lgan satr (Telegram uni imzolamaydi), ya'ni
	// hech qanday tekshiruvsiz har kim istalgan chat ID ni yubora olardi.
	AuthorizeChat(ctx context.Context, mentorID string, chatID int64) (*entity.TelegramChat, error)
}
