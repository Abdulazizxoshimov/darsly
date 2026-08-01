package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/zoom/darsly/internal/entity"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// ─── FakeTelegram — Bot API klientining soxtasi ──────────────────────────────

// FakeTelegram — `telegram.Client` ning testdagi o'rnini bosuvchi.
//
// Xatolarni SOZLASH mumkin (`SendVideoErr`, `GetFileErr`): yuklash ishchisining
// eng muhim yo'llari — 429 (rate limit), doimiy xato va qayta urinish — aynan
// shu orqali tekshiriladi.
type FakeTelegram struct {
	mu sync.Mutex

	Disabled  bool
	ArchiveID int64
	MaxUpload int64
	Username  string
	LocalMode bool

	// Sent — arxivga yuborilgan videolar (chat ID → nechta).
	Sent map[int64]int
	// Forwarded — `file_id` bilan qayta yuborilganlar.
	Forwarded []string
	// Documents — yuborilgan hujjatlar (fayl nomlari).
	Documents []string
	// Messages — yuborilgan matnli xabarlar.
	Messages []string

	// SendVideoErr — `SendVideo` shu xatoni qaytaradi (nil → muvaffaqiyat).
	SendVideoErr error
	// GetFileErr / DownloadErr — tiklash yo'lini sinash uchun.
	GetFileErr  error
	DownloadErr error
	// DownloadBody — tiklashda diskka yoziladigan mazmun.
	DownloadBody string

	// NextFileID — muvaffaqiyatli yuborishda qaytariladigan `file_id`.
	// Bo'sh bo'lsa avtomatik yasaladi. Ataylab bo'sh qoldirib, «file_id
	// kelmadi» holatini ham sinash mumkin (`EmptyFileID`).
	NextFileID  string
	EmptyFileID bool
}

func NewFakeTelegram() *FakeTelegram {
	return &FakeTelegram{
		ArchiveID:    -1001,
		MaxUpload:    1900 * 1024 * 1024,
		Username:     "darsly_test_bot",
		LocalMode:    true,
		Sent:         map[int64]int{},
		DownloadBody: "restored-video-bytes",
	}
}

func (f *FakeTelegram) Enabled() bool              { return !f.Disabled }
func (f *FakeTelegram) Local() bool                { return f.LocalMode }
func (f *FakeTelegram) ArchiveChatID() int64       { return f.ArchiveID }
func (f *FakeTelegram) MaxUploadBytes() int64      { return f.MaxUpload }
func (f *FakeTelegram) BotUsername() string        { return f.Username }
func (f *FakeTelegram) Init(context.Context) error { return nil }

func (f *FakeTelegram) SendMessage(_ context.Context, _ int64, text string, _ *tg.InlineKeyboard) (*tg.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Messages = append(f.Messages, text)
	return &tg.Message{MessageID: int64(len(f.Messages))}, nil
}

func (f *FakeTelegram) EditMessageText(context.Context, int64, int64, string, *tg.InlineKeyboard) error {
	return nil
}
func (f *FakeTelegram) AnswerCallbackQuery(context.Context, string, string) error { return nil }

func (f *FakeTelegram) SendVideo(_ context.Context, chatID int64, _, _ string, _ int) (*tg.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.SendVideoErr != nil {
		return nil, f.SendVideoErr
	}
	f.Sent[chatID]++
	msg := &tg.Message{MessageID: int64(100 + f.Sent[chatID]), Chat: tg.Chat{ID: chatID}}
	if !f.EmptyFileID {
		id := f.NextFileID
		if id == "" {
			id = fmt.Sprintf("file-%d-%d", chatID, f.Sent[chatID])
		}
		msg.Video = &tg.Video{FileID: id}
	}
	return msg, nil
}

func (f *FakeTelegram) SendVideoByFileID(_ context.Context, chatID int64, fileID, _ string) (*tg.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Forwarded = append(f.Forwarded, fileID)
	return &tg.Message{MessageID: 1, Chat: tg.Chat{ID: chatID}}, nil
}

func (f *FakeTelegram) SendDocument(_ context.Context, _ int64, filename string, _ io.Reader, _ int64, _ string) (*tg.Message, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Documents = append(f.Documents, filename)
	return &tg.Message{MessageID: 1}, nil
}

func (f *FakeTelegram) GetFile(_ context.Context, fileID string) (*tg.File, error) {
	if f.GetFileErr != nil {
		return nil, f.GetFileErr
	}
	return &tg.File{FileID: fileID, FilePath: "videos/" + fileID + ".mp4", FileSize: 1024}, nil
}

// DownloadFile — haqiqiy fayl yozadi: `recording.RunRestore` uni `os.Stat` va
// `os.Open` bilan o'qiydi, ya'ni mavjud bo'lmagan fayl bilan test yo'lni
// oxirigacha bosib o'tmasdi.
func (f *FakeTelegram) DownloadFile(_ context.Context, _ *tg.File, dst string) error {
	if f.DownloadErr != nil {
		return f.DownloadErr
	}
	return os.WriteFile(dst, []byte(f.DownloadBody), 0o600)
}

func (f *FakeTelegram) GetUpdates(context.Context, int64, int) ([]tg.Update, error) { return nil, nil }

// ─── FakeTelegramRepo ────────────────────────────────────────────────────────

// FakeTelegramRepo — `repository.TelegramRepository` in-memory implementatsiyasi.
type FakeTelegramRepo struct {
	mu sync.Mutex
	// users — userID → foydalanuvchi (faqat telegram maydonlari muhim).
	users map[string]*entity.User
	chats map[int64]*entity.TelegramChat
}

func NewFakeTelegramRepo() *FakeTelegramRepo {
	return &FakeTelegramRepo{
		users: map[string]*entity.User{},
		chats: map[int64]*entity.TelegramChat{},
	}
}

// AddUser — testga foydalanuvchi qo'shadi (bog'lanmagan holatda).
func (r *FakeTelegramRepo) AddUser(id, fullName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[id] = &entity.User{ID: id, FullName: fullName, Role: "mentor", IsActive: true}
}

func (r *FakeTelegramRepo) LinkUser(_ context.Context, userID string, telegramUserID int64, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[userID]
	if !ok {
		return apperr.NotFound("user")
	}
	// Postgres bilan bir xil: bu Telegram akkaunti boshqada bo'lsa uziladi
	// (UNIQUE indeks bilan to'qnashmaslik uchun).
	for _, other := range r.users {
		if other.ID != userID && other.TelegramUserID != nil && *other.TelegramUserID == telegramUserID {
			other.TelegramUserID = nil
			other.TelegramUsername = nil
			other.TelegramLinkedAt = nil
		}
	}
	now := time.Now().UTC()
	tgID := telegramUserID
	u.TelegramUserID = &tgID
	if username != "" {
		un := username
		u.TelegramUsername = &un
	}
	u.TelegramLinkedAt = &now
	return nil
}

func (r *FakeTelegramRepo) UnlinkUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[userID]; ok {
		u.TelegramUserID = nil
		u.TelegramUsername = nil
		u.TelegramLinkedAt = nil
	}
	return nil
}

func (r *FakeTelegramRepo) GetUserByTelegramID(_ context.Context, telegramUserID int64) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.TelegramUserID != nil && *u.TelegramUserID == telegramUserID {
			cp := *u
			return &cp, nil
		}
	}
	return nil, apperr.NotFound("user")
}

func (r *FakeTelegramRepo) GetLink(_ context.Context, userID string) (*entity.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[userID]
	if !ok {
		return nil, apperr.NotFound("user")
	}
	cp := *u
	return &cp, nil
}

func (r *FakeTelegramRepo) UpsertChat(_ context.Context, chat *entity.TelegramChat) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *chat
	cp.IsActive = true
	if existing, ok := r.chats[chat.ChatID]; ok {
		// COALESCE(EXCLUDED.mentor_id, mavjud) — postgres bilan bir xil.
		if cp.MentorID == nil {
			cp.MentorID = existing.MentorID
		}
		cp.AddedAt = existing.AddedAt
	} else {
		cp.AddedAt = time.Now().UTC()
	}
	cp.UpdatedAt = time.Now().UTC()
	r.chats[chat.ChatID] = &cp
	return nil
}

func (r *FakeTelegramRepo) DeactivateChat(_ context.Context, chatID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.chats[chatID]; ok {
		c.IsActive = false
	}
	return nil
}

func (r *FakeTelegramRepo) ListChatsByMentor(_ context.Context, mentorID string) ([]*entity.TelegramChat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*entity.TelegramChat
	for _, c := range r.chats {
		if c.IsActive && c.MentorID != nil && *c.MentorID == mentorID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *FakeTelegramRepo) GetChat(_ context.Context, chatID int64) (*entity.TelegramChat, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.chats[chatID]
	if !ok {
		return nil, apperr.NotFound("telegram chat")
	}
	cp := *c
	return &cp, nil
}
