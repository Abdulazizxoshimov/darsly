package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type RecordingRepository interface {
	Create(ctx context.Context, rec *entity.Recording) error
	GetByID(ctx context.Context, id string) (*entity.Recording, error)
	GetByEgressID(ctx context.Context, egressID string) (*entity.Recording, error)
	ListByLesson(ctx context.Context, lessonID string) ([]*entity.Recording, error)
	UpdateStatus(ctx context.Context, id, status string) error
	MarkReady(ctx context.Context, egressID, objectKey string, durationSec int, sizeBytes int64, endedAt time.Time) error
	MarkFailed(ctx context.Context, egressID string, endedAt time.Time) error

	// ── Retention (30 kunlik saqlash) — `worker.RetentionWorker` ishlatadi ──

	// ListExpired — saqlash muddati o'tgan `ready` yozuvlar
	// (tugash vaqti `endedBefore` dan oldin).
	//
	// ⚠️ Telegram arxivi O'CHIQ bo'lgan o'rnatishlar uchun: bu yerdan qaytgan
	// yozuv butunlay o'chiriladi (`expired`). Telegram yoqilgan bo'lsa ishchi
	// buning O'RNIGA [ListArchivable] ni ishlatadi.
	ListExpired(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ListArchivable — muddati o'tgan va Telegramda TASDIQLANGAN yozuvlar.
	//
	// ⭐ `telegram_sent_at IS NOT NULL` sharti SQL'da: server nusxasini
	// o'chirish faqat Telegramda nusxa borligi tasdiqlangandan keyin mumkin
	// (PRODUCT.md — «video hech qachon yo'qolmaydi»). Shartni usecase'ga
	// chiqarish "ro'yxatni oldik, filtrlashni unutdik" degan bitta xatoda
	// yozuvni butunlay yo'qotish xavfini tug'dirardi.
	//
	// Kesh nusxalari (`cached_until IS NOT NULL`) chiqmaydi — ular
	// [ListCacheExpired] orqali o'z muddati bo'yicha tozalanadi.
	ListArchivable(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ListExpiredUnconfirmed — muddati o'tgan, LEKIN Telegramga tushmagan
	// yozuvlar. Ular O'CHIRILMAYDI; mentor ogohlantiriladi.
	ListExpiredUnconfirmed(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ClaimExpire atomik ravishda yozuvni `expired` deb belgilaydi (faqat hamon
	// `ready` bo'lsa true). Ko'p instansda bitta faylni ikki marta o'chirishning
	// oldini oladi — `ClaimReminder` bilan bir xil naqsh.
	//
	// DIQQAT: Telegram arxivi bilan bu yo'l endi ishlatilmaydi — o'rniga
	// [RecordingRepository.ClaimArchive]. Metod eski (Telegramsiz) o'rnatishlar
	// va testlar uchun qoldi.
	ClaimExpire(ctx context.Context, id string, deletedAt time.Time) (bool, error)
	// ClaimArchive — `ready → archived` atomik o'tish (server nusxasi
	// o'chirilmoqda, Telegramdagi qoladi).
	//
	// `telegram_sent_at IS NOT NULL` sharti BU YERDA HAM takrorlanadi:
	// ro'yxat olingandan keyin va o'chirishgacha bo'lgan oynada holat
	// o'zgargan bo'lishi mumkin (masalan qo'lda tiklangan). Ikki joyda
	// tekshirish arzon, ma'lumot yo'qotish esa qaytarib bo'lmaydigan.
	ClaimArchive(ctx context.Context, id string, deletedAt time.Time) (bool, error)
	// ListExpiringUnwarned — muddati yaqinlashgan, lekin ogohlantirish hali
	// yuborilmagan `ready` yozuvlar.
	ListExpiringUnwarned(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ClaimRetentionWarning atomik ravishda "ogohlantirildi" deb belgilaydi
	// (faqat hali belgilanmagan bo'lsa true) — dublikat bildirishnoma bo'lmaydi.
	ClaimRetentionWarning(ctx context.Context, id string) (bool, error)

	// ── Telegram arxivi — `worker.TelegramUploadWorker` ishlatadi ───────────

	// EnqueueTelegram — yozuvni Telegramga yuborish navbatiga qo'yadi
	// (`HandleEgress` da, yozuv `ready` bo'lgan zahoti).
	EnqueueTelegram(ctx context.Context, egressID string) error
	// ClaimTelegramUpload — navbatdan BITTA yozuvni atomik oladi
	// (`FOR UPDATE SKIP LOCKED`), urinish hisobini oshiradi va keyingi urinish
	// vaqtini uzoqqa suradi.
	//
	// Nega hisob DARHOL oshadi: yuklash soatlab davom etishi mumkin va shu
	// paytda jarayon o'lsa (deploy, OOM) yozuv navbatda "band" bo'lib qolardi.
	// Keyingi urinish vaqti oldindan surilgani uchun u o'zi qaytadi.
	//
	// Navbat bo'sh bo'lsa `nil, nil`.
	ClaimTelegramUpload(ctx context.Context, now time.Time, maxAttempts int) (*entity.Recording, error)
	// MarkTelegramSent — muvaffaqiyat: `telegram_sent_at` to'ldiriladi va
	// AYNAN shundan keyin server nusxasini o'chirish mumkin bo'ladi.
	MarkTelegramSent(ctx context.Context, id string, chatID, messageID int64, fileID string) error
	// MarkTelegramFailed — xato + keyingi urinish vaqti. `nextAttempt` nil →
	// urinishlar tugadi (mentorga bildirishnoma ketadi, fayl SAQLANADI).
	MarkTelegramFailed(ctx context.Context, id, errMsg string, nextAttempt *time.Time) error

	// ── Telegramdan qaytarib olish (restore) ────────────────────────────────

	// ClaimRestore — `archived → restoring` atomik o'tish.
	//
	// Idempotentlik shu yerda: ikki marta bosilgan "tiklash" tugmasidan
	// ikkinchisi `false` oladi va ikkinchi yuklab olish BOSHLANMAYDI
	// (aks holda ikkita 1 GB yuklash parallel ketardi).
	ClaimRestore(ctx context.Context, id string) (bool, error)
	// FinishRestore — fayl MinIO'ga qaytdi: `restoring → ready` + kesh muddati.
	FinishRestore(ctx context.Context, id, objectKey string, cachedUntil time.Time) error
	// FailRestore — tiklash yiqildi: `restoring → archived` (Telegramdagi
	// nusxa joyida, mentor qayta urinib ko'radi).
	FailRestore(ctx context.Context, id, errMsg string) error
	// ListByStatus — berilgan holatdagi yozuvlar (eng eskisidan).
	//
	// `worker.RestoreWorker` `restoring` yozuvlarni shu orqali topadi. Nega
	// navbat jadvali emas: holatning O'ZI navbat. Jarayon tiklash o'rtasida
	// o'lsa yozuv `restoring` da qoladi va keyingi skanda O'ZI qayta olinadi —
	// alohida navbatda esa u yerda "band" bo'lib qotib qolardi.
	ListByStatus(ctx context.Context, status string, limit uint64) ([]*entity.Recording, error)
	// ListCacheExpired — kesh muddati tugagan TIKLANGAN nusxalar
	// (`cached_until < now`). Ular MinIO'dan qayta o'chiriladi.
	ListCacheExpired(ctx context.Context, now time.Time, limit uint64) ([]*entity.Recording, error)
	// ClaimCacheEvict — `ready(kesh) → archived` atomik qaytish.
	ClaimCacheEvict(ctx context.Context, id string) (bool, error)

	// Qayta kodlash navbati (CRF) — `worker.TranscodeWorker` ishlatadi.
	EnqueueTranscode(ctx context.Context, egressID string) error
	ClaimTranscode(ctx context.Context) (*entity.Recording, error)
	FinishTranscode(ctx context.Context, id string, newSize, originalSize int64) error
	FailTranscode(ctx context.Context, id string) error
	RequeueStaleTranscodes(ctx context.Context, olderThan time.Time) (int64, error)
}
