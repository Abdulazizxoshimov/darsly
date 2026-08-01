package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// TelegramRepository — Telegram integratsiyasining doimiy holati.
//
// ## Nega `users` ustunlari ham SHU YERDA
//
// `telegram_user_id` fizik jihatdan `users` jadvalida, lekin u AUTH domeniga
// tegishli emas: bog'lash oqimi, bekor qilish va "bu tugmani kim bosdi"
// qidiruvi butunlay Telegram domenining ishi. Ularni `UserRepository` ga
// qo'shish o'sha interfeysni begona mas'uliyat bilan shishirardi va ikki
// domen bitta faylda uchrashardi. Ustun `users` da — chunki bu foydalanuvchi
// atributi va alohida jadval keraksiz JOIN bo'lardi.
type TelegramRepository interface {
	// ── Mentor ↔ Telegram akkaunt bog'lanishi ───────────────────────────────

	// LinkUser — mentorni Telegram akkauntiga bog'laydi.
	//
	// ATOMIK va IDEMPOTENT: shu Telegram ID boshqa foydalanuvchida bo'lsa
	// avval undan uziladi. Sabab — mentor akkauntini almashtirganda
	// (masalan yangi hisob ochganda) UNIQUE indeks bilan to'qnashib, bog'lash
	// tushunarsiz 500 bilan yiqilardi.
	LinkUser(ctx context.Context, userID string, telegramUserID int64, username string) error
	// UnlinkUser — bog'lanishni uzadi (mentor ilovada "uzish"ni bosdi).
	UnlinkUser(ctx context.Context, userID string) error
	// GetUserByTelegramID — bot yangilanishida kelgan Telegram ID bo'yicha
	// mentorni topadi. Topilmasa `apperr.NotFound("user")`.
	GetUserByTelegramID(ctx context.Context, telegramUserID int64) (*entity.User, error)
	// GetLink — mentorning bog'lanish holati (username, sana). Bog'lanmagan
	// bo'lsa nil, nil.
	GetLink(ctx context.Context, userID string) (*entity.User, error)

	// ── Bot a'zo bo'lgan guruhlar ───────────────────────────────────────────

	// UpsertChat — `my_chat_member` hodisasidan kelgan guruhni yozadi/yangilaydi.
	//
	// `mentorID` nil bo'lsa MAVJUD egasi saqlanadi: guruh nomi o'zgarganda
	// (yoki bot admin qilinganda) hodisa boshqa odamdan kelishi mumkin va
	// egalikni o'sha odamga o'tkazib yuborish noto'g'ri bo'lardi.
	UpsertChat(ctx context.Context, chat *entity.TelegramChat) error
	// DeactivateChat — bot guruhdan chiqarildi. Qator o'chirilmaydi (tarix).
	DeactivateChat(ctx context.Context, chatID int64) error
	// ListChatsByMentor — mentor botni qo'shgan FAOL guruhlar.
	ListChatsByMentor(ctx context.Context, mentorID string) ([]*entity.TelegramChat, error)
	// GetChat — bitta guruh (mentor tanlagan chat haqiqatan mavjudligini va
	// unga tegishli ekanini tekshirish uchun).
	GetChat(ctx context.Context, chatID int64) (*entity.TelegramChat, error)
}
