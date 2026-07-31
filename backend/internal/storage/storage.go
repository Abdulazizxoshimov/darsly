package storage

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/zoom/darsly/internal/infrastructure/repository"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	"github.com/zoom/darsly/internal/pkg/postgres"
)

// Storage barcha repository implementatsiyalarini bir joyda saqlaydi.
// Yangi domen qo'shilganda shu yerga repository qo'shiladi.
type Storage struct {
	// pg — tranzaksiya ochish uchun (`WithTx`). Repozitoriylar undan
	// to'g'ridan-to'g'ri foydalanmaydi.
	pg *postgres.Postgres

	User         repository.UserRepository
	Auth         repository.AuthRepository
	Lesson       repository.LessonRepository
	WaitingRoom  repository.WaitingRoomRepository
	Recording    repository.RecordingRepository
	Notification repository.NotificationRepository
	Chat         repository.ChatRepository
	Poll         repository.PollRepository
	Blocklist    repository.BlocklistRepository
}

func New(pg *postgres.Postgres) *Storage {
	return &Storage{
		pg:           pg,
		User:         pgRepo.NewUserRepo(pg),
		Auth:         pgRepo.NewAuthRepo(pg),
		Lesson:       pgRepo.NewLessonRepo(pg),
		WaitingRoom:  pgRepo.NewWaitingRoomRepo(pg),
		Recording:    pgRepo.NewRecordingRepo(pg),
		Notification: pgRepo.NewNotificationRepo(pg),
		Chat:         pgRepo.NewChatRepo(pg),
		Poll:         pgRepo.NewPollRepo(pg),
		Blocklist:    pgRepo.NewBlocklistRepo(pg),
	}
}

// RunInTx — `users` va `auth` yozuvlarini BITTA tranzaksiyada bajaradi.
//
// Imzo ATAYLAB `*Storage` emas, aniq repozitoriy interfeyslari: usecase
// qatlami `storage` paketini bilmasligi kerak (qatlam yo'nalishi). Usecase o'z
// tomonida shu shakldagi kichik interfeys ta'riflaydi (`auth.TxRunner`) va
// `*Storage` uni qondiradi — loyihadagi `LiveKit`/`Recorder`/`Hands` bilan
// bir xil DIP naqshi.
func (s *Storage) RunInTx(
	ctx context.Context,
	fn func(users repository.UserRepository, auth repository.AuthRepository) error,
) error {
	return s.WithTx(ctx, func(tx *Storage) error {
		return fn(tx.User, tx.Auth)
	})
}

// WithTx — `fn` ichidagi barcha yozuvlarni BITTA tranzaksiyada bajaradi.
//
// # Nega kerak
//
// Ko'p-yozuvli oqimlar (masalan ro'yxatdan o'tish: `users` + `refresh_tokens`)
// ilgari KOMPENSATSIYA bilan qoplanardi: ikkinchi yozuv uzilsa, birinchisi
// qo'lda `DeleteHard` bilan o'chirilardi. Bu ishlaydi, lekin ikki zaif joyi bor:
//   - kompensatsiyaning O'ZI uzilishi mumkin (masalan DB shu payt yiqilgan) —
//     natijada yetim qator qoladi va email band bo'lib turadi;
//   - har yangi yozuv qo'shilganda kompensatsiyani ham yangilash kerak, aks
//     holda u jimgina to'liqsiz bo'lib qoladi.
//
// Tranzaksiyada bu sinf muammosi yo'q: rollback'ni DB kafolatlaydi.
//
// ⚠️ CHEKLOV: `fn` ichida TASHQI chaqiruv (Redis, LiveKit, HTTP) qilmang.
// Tranzaksiya ochiq turganda ulanish va qulflar band bo'ladi; sekin tashqi
// servis butun poolni to'ldirib qo'yishi mumkin. Tashqi ishni tranzaksiyadan
// OLDIN yoki KEYIN bajaring.
func (s *Storage) WithTx(ctx context.Context, fn func(s *Storage) error) error {
	return postgres.WithTx(ctx, s.pg.DB, func(tx pgx.Tx) error {
		// Faqat tranzaksiyani QO'LLAB-QUVVATLAYDIGAN repozitoriylar qayta
		// bog'lanadi. Qolganlari pool'da qoladi — ular bu oqimda ishlatilmaydi
		// va ularni ham ko'chirish keraksiz refaktoring bo'lardi.
		txStore := *s
		txStore.User = pgRepo.NewUserRepoTx(tx, s.pg.Builder)
		txStore.Auth = pgRepo.NewAuthRepoTx(tx, s.pg.Builder)
		return fn(&txStore)
	})
}
