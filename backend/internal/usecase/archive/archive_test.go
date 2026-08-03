package archive_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/archive"
)

const (
	lessonID = "11111111-1111-4111-8111-111111111111"
	mentorID = "mentor1"
)

// startedAt — dars boshlanish vaqti; `offset_sec` shunga nisbatan hisoblanadi.
var startedAt = time.Date(2026, 8, 1, 9, 30, 0, 0, time.UTC)

type env struct {
	uc      archive.UseCase
	lessons *testutil.FakeLessonRepo
	chat    *testutil.FakeChatRepo
	recs    *testutil.FakeRecordingRepo
	minio   *testutil.FakeMinio
}

// newEnv — yakunlangan, boshlanish vaqti ma'lum dars.
func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		lessons: testutil.NewFakeLessonRepo(),
		chat:    testutil.NewFakeChatRepo(),
		recs:    testutil.NewFakeRecordingRepo(),
		minio:   testutil.NewFakeMinio(),
	}
	st := startedAt
	require.NoError(t, e.lessons.Create(context.Background(), &entity.Lesson{
		ID: lessonID, MentorID: mentorID, Title: "Algebra",
		Status: entity.LessonStatusEnded, StartedAt: &st,
	}))
	// Retention 30 kun — `expires_at` hisobini tekshirish uchun.
	e.uc = archive.New(e.lessons, e.chat, e.recs, e.minio, 30*24*time.Hour, false, testutil.NewLogger())
	return e
}

// msg — chat xabari qo'shadi (dars boshlanishidan `after` keyin).
func (e *env) msg(t *testing.T, id, identity, name, body string, after time.Duration, to *string, file *entity.ChatFile) {
	t.Helper()
	require.NoError(t, e.chat.Create(context.Background(), &entity.ChatMessage{
		ID: id, LessonID: lessonID, SenderIdentity: identity, SenderName: name,
		Body: body, ToIdentity: to, File: file, CreatedAt: startedAt.Add(after),
	}))
}

func (e *env) rec(t *testing.T, id, status, key string, ended time.Duration) *entity.Recording {
	t.Helper()
	end := startedAt.Add(ended)
	r := &entity.Recording{
		ID: id, LessonID: lessonID, EgressID: "eg-" + id, ObjectKey: key,
		Status: status, DurationSec: 3600, SizeBytes: 123456,
		StartedAt: startedAt, EndedAt: &end, CreatedAt: startedAt,
	}
	require.NoError(t, e.recs.Create(context.Background(), r))
	return r
}

// recAt — yozuv boshlanishi darsdan `delay` keyin va boshidan `trimmed` soniya
// kesilgan. Vaqt shkalasi testlari uchun (`PlaybackZero`).
func (e *env) recAt(t *testing.T, id string, delay time.Duration, trimmed int) {
	t.Helper()
	end := startedAt.Add(time.Hour)
	require.NoError(t, e.recs.Create(context.Background(), &entity.Recording{
		ID: id, LessonID: lessonID, EgressID: "eg-" + id, ObjectKey: "rec/" + id + ".mp4",
		Status: entity.RecordingStatusReady, DurationSec: 3600, SizeBytes: 123456,
		StartedAt: startedAt.Add(delay), EndedAt: &end, CreatedAt: startedAt,
		ContentOffsetSec: trimmed,
	}))
}

// ─── Egalik ──────────────────────────────────────────────────────────────────

func TestGet_Egalik(t *testing.T) {
	e := newEnv(t)
	_, err := e.uc.Get(context.Background(), "intruder", lessonID)
	require.True(t, apperr.IsForbidden(err), "begona mentor arxivni ko'ra olmaydi")
}

func TestGet_YoqDars(t *testing.T) {
	e := newEnv(t)
	_, err := e.uc.Get(context.Background(), mentorID, "22222222-2222-4222-8222-222222222222")
	require.True(t, apperr.IsNotFound(err))
}

// Yaroqsiz UUID DB'ga YETMASLIGI kerak (aks holda 22P02 → soxta 500 + Sentry
// shovqini). Javob mavjud bo'lmagan UUID bilan bir xil — `shared.ValidateID`
// izohidagi sabab (classification oracle qolmasin).
func TestGet_YaroqsizID(t *testing.T) {
	e := newEnv(t)
	_, err := e.uc.Get(context.Background(), mentorID, "not-a-uuid")
	require.True(t, apperr.IsNotFound(err))
}

// ─── Bo'sh holatlar ──────────────────────────────────────────────────────────

// Yozuvsiz dars — `recording: null`, lekin XATO EMAS.
func TestGet_YozuvsizDars(t *testing.T) {
	e := newEnv(t)
	e.msg(t, "m1", "guest_a", "Ali", "salom", time.Minute, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Nil(t, a.Recording)
	require.Len(t, a.Chat, 1)
	require.Equal(t, "Algebra", a.Lesson.Title)
}

// Chatsiz dars — bo'sh massiv (nil emas: klient `.map` qiladi).
func TestGet_ChatsizDars(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r1", entity.RecordingStatusReady, "rec/r1.mp4", time.Hour)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.NotNil(t, a.Chat)
	require.Empty(t, a.Chat)
	require.NotNil(t, a.Materials)
	require.Empty(t, a.Materials)
	require.NotNil(t, a.Recording)
}

// ─── offset_sec ──────────────────────────────────────────────────────────────

func TestGet_OffsetHisobi(t *testing.T) {
	e := newEnv(t)
	e.msg(t, "m1", "guest_a", "Ali", "boshida", 0, nil, nil)
	e.msg(t, "m2", "guest_a", "Ali", "ikki daqiqa", 125*time.Second, nil, nil)
	e.msg(t, "m3", "guest_a", "Ali", "bir soat", time.Hour, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Len(t, a.Chat, 3)
	require.Equal(t, 0, a.Chat[0].OffsetSec)
	require.Equal(t, 125, a.Chat[1].OffsetSec)
	require.Equal(t, 3600, a.Chat[2].OffsetSec)
	// Tartib — eskidan yangiga (o'qish tartibi).
	require.Equal(t, "boshida", a.Chat[0].Body)
	require.Equal(t, "bir soat", a.Chat[2].Body)
}

// Dars boshlanishidan OLDIN yozilgan xabar (kutish xonasi) — 0, manfiy emas.
func TestGet_OffsetManfiyEmas(t *testing.T) {
	e := newEnv(t)
	e.msg(t, "m1", "guest_a", "Ali", "erta keldim", -5*time.Minute, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 0, a.Chat[0].OffsetSec)
}

// `started_at` bo'lmasa yozuv boshlanishiga tushamiz.
func TestGet_OffsetYozuvdanHisoblanadi(t *testing.T) {
	e := newEnv(t)
	l, err := e.lessons.GetByID(context.Background(), lessonID)
	require.NoError(t, err)
	l.StartedAt = nil
	require.NoError(t, e.lessons.Update(context.Background(), l))
	e.rec(t, "r1", entity.RecordingStatusReady, "rec/r1.mp4", time.Hour)
	e.msg(t, "m1", "guest_a", "Ali", "salom", 90*time.Second, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 90, a.Chat[0].OffsetSec)
}

// ⭐ Yozuv DARSDAN KEYINROQ boshlangan: nol nuqta yozuvniki bo'lishi kerak.
//
// Egress birinchi trek e'lon qilinganda ishga tushadi, ya'ni darsdan bir necha
// soniya/daqiqa keyin. Sakrash dars boshidan hisoblansa xabar videoning
// noto'g'ri joyiga olib borardi.
func TestGet_OffsetYozuvBoshidanHisoblanadi(t *testing.T) {
	e := newEnv(t)
	e.recAt(t, "r1", 40*time.Second, 0) // egress 40 s kechikdi
	e.msg(t, "m1", "guest_a", "Ali", "salom", 100*time.Second, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 60, a.Chat[0].OffsetSec,
		"100 s (dars boshidan) − 40 s (egress kechikishi) = videoda 60 s")
}

// ⭐ TRANSKOD BOSHINI KESGAN: sakrash shu qadar surilishi kerak.
//
// Aynan shu holat 2026-08-03 da qo'shilgan «boshidagi qora va jim qismni
// kesish» tufayli paydo bo'ldi. Busiz butun chat kesilgan miqdorga siljib,
// har bir xabar noto'g'ri lahzaga olib borardi.
func TestGet_OffsetKesilganBoshniHisobgaOladi(t *testing.T) {
	e := newEnv(t)
	e.recAt(t, "r1", 0, 13) // transkod boshidan 13 s kesdi
	e.msg(t, "m1", "guest_a", "Ali", "kesishdan oldin", 5*time.Second, nil, nil)
	e.msg(t, "m2", "guest_a", "Ali", "kesishdan keyin", 73*time.Second, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 0, a.Chat[0].OffsetSec,
		"kesilgan qismdagi xabar 0 ga qisiladi (manfiy bo'lmaydi)")
	require.Equal(t, 60, a.Chat[1].OffsetSec, "73 s − 13 s kesildi = videoda 60 s")
}

// Kechikish VA kesish birga — ikkalasi ham qo'shilib hisoblanadi.
func TestGet_OffsetKechikishVaKesishBirga(t *testing.T) {
	e := newEnv(t)
	e.recAt(t, "r1", 20*time.Second, 13)
	e.msg(t, "m1", "guest_a", "Ali", "savol", 93*time.Second, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 60, a.Chat[0].OffsetSec, "93 − 20 − 13 = 60")
}

// Na dars, na yozuv boshlanish vaqtini bilmasa — 0 (noto'g'ri sakrashdan ko'ra
// sakramaslik yaxshi).
func TestGet_OffsetTayanchsiz(t *testing.T) {
	e := newEnv(t)
	l, err := e.lessons.GetByID(context.Background(), lessonID)
	require.NoError(t, err)
	l.StartedAt = nil
	require.NoError(t, e.lessons.Update(context.Background(), l))
	e.msg(t, "m1", "guest_a", "Ali", "salom", time.Hour, nil, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, 0, a.Chat[0].OffsetSec)
}

// ─── Materiallar ─────────────────────────────────────────────────────────────

func TestGet_MateriallarYigiladi(t *testing.T) {
	e := newEnv(t)
	e.msg(t, "m1", "guest_a", "Ali", "matn", time.Minute, nil, nil)
	e.msg(t, "m2", "guest_a", "Ali", "", 2*time.Minute, nil, &entity.ChatFile{
		Name: "masala.pdf", Size: 12345, Mime: "application/pdf", Key: "chat/x/masala.pdf",
	})
	e.msg(t, "m3", mentorID, "Dilnoza", "javob", 3*time.Minute, nil, &entity.ChatFile{
		Name: "javob.png", Size: 999, Mime: "image/png", Key: "chat/x/javob.png",
	})

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Len(t, a.Materials, 2, "faqat fayl xabarlari materialga aylanadi")
	require.Equal(t, "masala.pdf", a.Materials[0].Name)
	require.Equal(t, int64(12345), a.Materials[0].Size)
	require.Equal(t, "application/pdf", a.Materials[0].Mime)
	require.Contains(t, a.Materials[0].URL, "chat/x/masala.pdf", "presigned havola bo'lishi kerak")
	require.Equal(t, "javob.png", a.Materials[1].Name)

	// Chatdagi fayl ham imzolangan.
	require.NotNil(t, a.Chat[1].File)
	require.NotEmpty(t, a.Chat[1].File.URL)
	require.Equal(t, 3600, a.Chat[1].File.ExpiresInS)
}

// MinIO obyekt kaliti JSON'ga CHIQMASLIGI kerak: u bucket ichidagi nomlash
// sxemasini oshkor qilardi (`entity.ChatFile.Key` izohi).
func TestGet_ObyektKalitiChiqmaydi(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r1", entity.RecordingStatusReady, "rec/2026/secret-key.mp4", time.Hour)
	e.msg(t, "m1", "guest_a", "Ali", "", time.Minute, nil, &entity.ChatFile{
		Name: "a.pdf", Key: "chat/lesson/secret-file-key.pdf",
	})

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	raw, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(raw), `"key"`)
	require.NotContains(t, string(raw), `"object_key"`)
	// Presigned havolada yo'l ko'rinadi (bu muqarrar), lekin ALOHIDA maydon sifatida emas.
	require.NotContains(t, string(raw), `"egress_id"`)

	// `to_identity` va `file` bo'sh bo'lganda ham maydon TURADI (omitempty emas) —
	// klient `msg.file === null` deb tekshiradi, `undefined` deb emas.
	require.Contains(t, string(raw), `"to_identity":null`)
}

// ─── Yozuv holatlari ─────────────────────────────────────────────────────────

func TestGet_ReadyYozuvHavolasi(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r1", entity.RecordingStatusReady, "rec/r1.mp4", time.Hour)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, a.Recording.Status)
	require.NotNil(t, a.Recording.URL)
	require.Contains(t, *a.Recording.URL, "rec/r1.mp4")
	require.Equal(t, 3600, a.Recording.DurationSec)
	require.Equal(t, int64(123456), a.Recording.SizeBytes)
	// expires_at = ended_at + 30 kun.
	require.NotNil(t, a.Recording.ExpiresAt)
	require.Equal(t, startedAt.Add(time.Hour).Add(30*24*time.Hour), *a.Recording.ExpiresAt)
}

// `expired` — status SHUNDAYLIGICHA qoladi (Telegramdan qaytarib olish oqimi
// shunga tayanadi), havola esa `null` (fayl MinIO'da yo'q).
func TestGet_ExpiredYozuvHavolasiz(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r1", entity.RecordingStatusExpired, "rec/r1.mp4", time.Hour)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusExpired, a.Recording.Status)
	require.Nil(t, a.Recording.URL)
	require.Nil(t, a.Recording.ExpiresAt, "o'chgan yozuvda muddat ma'nosiz")
}

func TestGet_ProcessingYozuvHavolasiz(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r1", entity.RecordingStatusProcessing, "rec/r1.mp4", time.Hour)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusProcessing, a.Recording.Status)
	require.Nil(t, a.Recording.URL, "tayyor bo'lmagan yozuvga havola berilmaydi")
}

// Bir nechta yozuvdan `ready` yutadi (arxivda bitta pleyer bor).
func TestGet_ReadyYozuvUstuvor(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r-failed", entity.RecordingStatusFailed, "", 10*time.Minute)
	e.rec(t, "r-ready", entity.RecordingStatusReady, "rec/ok.mp4", 20*time.Minute)
	e.rec(t, "r-expired", entity.RecordingStatusExpired, "rec/old.mp4", 30*time.Minute)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, "r-ready", a.Recording.ID)
}

// `ready` bo'lmasa `expired` `failed` dan ustun (tiklash imkoni bor).
func TestGet_ExpiredFailedanUstun(t *testing.T) {
	e := newEnv(t)
	e.rec(t, "r-failed", entity.RecordingStatusFailed, "", 10*time.Minute)
	e.rec(t, "r-expired", entity.RecordingStatusExpired, "rec/old.mp4", 5*time.Minute)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Equal(t, "r-expired", a.Recording.ID)
}

// ─── Maxfiylik ───────────────────────────────────────────────────────────────

// Begona shaxsiy yozishma (o'quvchidan o'quvchiga) arxivga TUSHMAYDI —
// `chat.History` bilan bir xil qoida.
func TestGet_BegonaDMChiqmaydi(t *testing.T) {
	e := newEnv(t)
	other := "guest_b"
	e.msg(t, "m1", "guest_a", "Ali", "faqat senga", time.Minute, &other, nil)
	e.msg(t, "m2", "guest_a", "Ali", "ommaviy", 2*time.Minute, nil, nil)
	mentor := mentorID
	e.msg(t, "m3", "guest_c", "Vali", "ustozga shaxsiy", 3*time.Minute, &mentor, nil)

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Len(t, a.Chat, 2)
	require.Equal(t, "ommaviy", a.Chat[0].Body)
	require.Equal(t, "ustozga shaxsiy", a.Chat[1].Body)
	require.Equal(t, mentorID, *a.Chat[1].ToIdentity)
}

// MinIO sozlanmagan muhitda ham arxiv ISHLAYDI (havolasiz).
func TestGet_SaqlagichsizMuhit(t *testing.T) {
	e := newEnv(t)
	e.uc = archive.New(e.lessons, e.chat, e.recs, nil, 30*24*time.Hour, false, testutil.NewLogger())
	e.rec(t, "r1", entity.RecordingStatusReady, "rec/r1.mp4", time.Hour)
	e.msg(t, "m1", "guest_a", "Ali", "", time.Minute, nil, &entity.ChatFile{
		Name: "a.pdf", Key: "chat/x/a.pdf",
	})

	a, err := e.uc.Get(context.Background(), mentorID, lessonID)
	require.NoError(t, err)
	require.Nil(t, a.Recording.URL)
	require.Equal(t, "a.pdf", a.Materials[0].Name)
	require.Empty(t, a.Materials[0].URL)
}
