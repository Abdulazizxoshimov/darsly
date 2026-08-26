package worker

// B-1 — Telegram bot IDOR gate (ownedRecording) + callback-data parsing.
//
// `telegram_bot.go` da 0 ta test bor edi. `ownedRecording` — mentor FAQAT o'z
// yozuvini Telegram guruhiga ulasha olishini ta'minlaydigan yagona to'siq.
// `callback_data` ("s|<recID>|<chatID>") ni bosgan odam istalgancha
// o'zgartira oladi (Telegram uni imzolamaydi), shuning uchun begona recording
// ID bilan boshqa mentorning yozuvi guruhga sizib chiqmasligi SHART.

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	tguc "github.com/zoom/darsly/internal/usecase/telegram"
)

const (
	// Ikki mentor — biri "o'z" yozuvining egasi, ikkinchisi begona.
	botMentorA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" // yozuv egasi
	botMentorB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb" // begona
	botLessonA = "11111111-1111-4111-8111-111111111111"
	botRecA    = "22222222-2222-4222-8222-222222222222"
	botTgA     = int64(1001) // mentorA telegram akkaunt id
	botTgB     = int64(2002) // mentorB telegram akkaunt id
	botChatB   = int64(-500) // mentorB ning O'Z guruhi
)

func strp(s string) *string { return &s }

// ownedRecTestWorker — ownedRecording'ni to'g'ridan-to'g'ri sinash uchun eng
// kichik worker: faqat rec/lesson repolar kerak.
func ownedRecTestWorker(t *testing.T) (*TelegramBotWorker, *testutil.FakeRecordingRepo, *testutil.FakeLessonRepo) {
	t.Helper()
	ctx := context.Background()
	recRepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: botLessonA, MentorID: botMentorA, Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	require.NoError(t, recRepo.Create(ctx, &entity.Recording{
		ID: botRecA, LessonID: botLessonA, Status: entity.RecordingStatusReady,
		TelegramFileID: strp("file-abc"),
	}))
	w := NewTelegramBotWorker(nil, nil, recRepo, lrepo, nil, nil, testutil.NewLogger())
	return w, recRepo, lrepo
}

// ⭐ Asosiy IDOR testi: yozuv egasi o'tadi, begona mentor RAD etiladi.
// Bug: egalik tekshiruvi bo'lmasa boshqa mentorning recording ID'si bilan
// uning darsining yozuvi begona guruhga ulashilib ketardi.
func TestOwnedRecording_EnforcesOwnership(t *testing.T) {
	w, _, _ := ownedRecTestWorker(t)
	ctx := context.Background()

	t.Run("o'z yozuvi o'tadi", func(t *testing.T) {
		rec, l, err := w.ownedRecording(ctx, botMentorA, botRecA)
		require.NoError(t, err)
		require.Equal(t, botRecA, rec.ID)
		require.Equal(t, botLessonA, l.ID)
	})

	t.Run("begona mentor RAD etiladi (exfiltration yo'q)", func(t *testing.T) {
		rec, l, err := w.ownedRecording(ctx, botMentorB, botRecA)
		require.Error(t, err)
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
		require.Nil(t, rec, "begona yozuv qaytarilmasin")
		require.Nil(t, l)
	})
}

// Yo'q yozuv / yo'q dars — xato, panik emas (callback ID ixtiyoriy satr).
func TestOwnedRecording_NotFound(t *testing.T) {
	w, _, _ := ownedRecTestWorker(t)
	_, _, err := w.ownedRecording(context.Background(), botMentorA, "deadbeef-dead-4dead-8dead-deaddeaddead")
	require.Error(t, err, "mavjud bo'lmagan recording ID xato berishi kerak")
}

// ─── handleCallback orqali to'liq zanjir (callback-data parsing + IDOR) ──────

// botCallbackEnv — MentorByTelegramID/AuthorizeChat bilan real telegram usecase
// va fake bot: forged callback'ning oqibatini uchdan-uchgacha ko'ramiz.
type botCallbackEnv struct {
	w   *TelegramBotWorker
	bot *testutil.FakeTelegram
}

func newBotCallbackEnv(t *testing.T) *botCallbackEnv {
	t.Helper()
	ctx := context.Background()

	tgRepo := testutil.NewFakeTelegramRepo()
	tgRepo.AddUser(botMentorA, "Ali (egasi)")
	tgRepo.AddUser(botMentorB, "Vali (begona)")
	require.NoError(t, tgRepo.LinkUser(ctx, botMentorA, botTgA, "ali"))
	require.NoError(t, tgRepo.LinkUser(ctx, botMentorB, botTgB, "vali"))
	// mentorB ning O'Z guruhi — u shu guruhga yuborishga haqli.
	mb := botMentorB
	require.NoError(t, tgRepo.UpsertChat(ctx, &entity.TelegramChat{
		ChatID: botChatB, Title: "Vali guruhi", Type: "supergroup", MentorID: &mb,
	}))

	recRepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: botLessonA, MentorID: botMentorA, Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	require.NoError(t, recRepo.Create(ctx, &entity.Recording{
		ID: botRecA, LessonID: botLessonA, Status: entity.RecordingStatusReady,
		TelegramFileID: strp("file-secret"),
	}))

	bot := testutil.NewFakeTelegram()
	uc := tguc.New(tgRepo, bot, testutil.NewFakeCache(), testutil.NewLogger())
	w := NewTelegramBotWorker(bot, uc, recRepo, lrepo, testutil.NewFakeCache(), nil, testutil.NewLogger())
	return &botCallbackEnv{w: w, bot: bot}
}

func shareCb(from int64, recID string, chatID int64) *tg.CallbackQuery {
	return &tg.CallbackQuery{
		ID:      "cbq1",
		From:    tg.User{ID: from},
		Data:    "s|" + recID + "|" + strconv.FormatInt(chatID, 10),
		Message: &tg.Message{MessageID: 7, Chat: tg.Chat{ID: from}},
	}
}

// ⭐ Cross-mentor exfiltration: mentorB o'z guruhiga mentorA ning yozuvini
// yuborishga urinadi. Video FORWARD qilinmasligi kerak.
func TestHandleCallback_RejectsForeignRecording(t *testing.T) {
	e := newBotCallbackEnv(t)
	e.w.handleCallback(context.Background(), shareCb(botTgB, botRecA, botChatB))

	require.Empty(t, e.bot.Forwarded,
		"begona mentorning yozuvi guruhga ulashilmasligi kerak (IDOR)")
}

// Nazorat: egasi (mentorA) o'z yozuvini o'z guruhiga yuborsa FORWARD bo'ladi —
// aks holda "hammani bloklash" degan noto'g'ri fix ham yashil ko'rinardi.
func TestHandleCallback_AllowsOwnRecording(t *testing.T) {
	e := newBotCallbackEnv(t)
	ctx := context.Background()
	// mentorA ning o'z guruhi bo'lsin.
	// (mentorA ni chatA egasi qilib qo'shamiz — env'da faqat mentorB guruhi bor,
	//  shuning uchun bu yerda AuthorizeChat mentorA uchun ishlashi kerak.)
	e = newBotCallbackEnvOwnerChat(t)
	e.w.handleCallback(ctx, shareCb(botTgA, botRecA, botChatA))

	require.Equal(t, []string{"file-secret"}, e.bot.Forwarded,
		"egasi o'z yozuvini o'z guruhiga yubora olishi kerak")
}

const botChatA = int64(-400)

// newBotCallbackEnvOwnerChat — pozitiv nazorat uchun: mentorA ning O'Z guruhi.
func newBotCallbackEnvOwnerChat(t *testing.T) *botCallbackEnv {
	t.Helper()
	ctx := context.Background()

	tgRepo := testutil.NewFakeTelegramRepo()
	tgRepo.AddUser(botMentorA, "Ali (egasi)")
	require.NoError(t, tgRepo.LinkUser(ctx, botMentorA, botTgA, "ali"))
	ma := botMentorA
	require.NoError(t, tgRepo.UpsertChat(ctx, &entity.TelegramChat{
		ChatID: botChatA, Title: "Ali guruhi", Type: "supergroup", MentorID: &ma,
	}))

	recRepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: botLessonA, MentorID: botMentorA, Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	require.NoError(t, recRepo.Create(ctx, &entity.Recording{
		ID: botRecA, LessonID: botLessonA, Status: entity.RecordingStatusReady,
		TelegramFileID: strp("file-secret"),
	}))

	bot := testutil.NewFakeTelegram()
	uc := tguc.New(tgRepo, bot, testutil.NewFakeCache(), testutil.NewLogger())
	w := NewTelegramBotWorker(bot, uc, recRepo, lrepo, testutil.NewFakeCache(), nil, testutil.NewLogger())
	return &botCallbackEnv{w: w, bot: bot}
}

// Callback-data parsing: 3 qismdan kam bo'lsa amal bajarilmaydi (noto'g'ri
// so'rov) — buzuq payload video yubormasligi kerak.
func TestHandleCallback_MalformedShareDataIsNoop(t *testing.T) {
	e := newBotCallbackEnvOwnerChat(t)
	ctx := context.Background()

	// chat_id qismisiz "s|<recID>" — len(parts) != 3.
	e.w.handleCallback(ctx, &tg.CallbackQuery{
		ID: "cbq2", From: tg.User{ID: botTgA}, Data: "s|" + botRecA,
		Message: &tg.Message{MessageID: 1, Chat: tg.Chat{ID: botTgA}},
	})
	require.Empty(t, e.bot.Forwarded, "buzuq callback-data video yubormasligi kerak")
}

// Bog'lanmagan Telegram akkaunt hech narsa ulasha olmaydi (from.id noma'lum).
func TestHandleCallback_UnlinkedUserRejected(t *testing.T) {
	e := newBotCallbackEnvOwnerChat(t)
	// botTgB bu env'da bog'lanmagan.
	e.w.handleCallback(context.Background(), shareCb(botTgB, botRecA, botChatA))
	require.Empty(t, e.bot.Forwarded, "bog'lanmagan akkaunt ulasha olmasligi kerak")
}
