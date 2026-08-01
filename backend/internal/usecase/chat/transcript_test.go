package chat_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/chat"
)

// utc — testlarda vaqtni AYNIQ berish uchun (UTC → +05:00 tekshiruvi shunga tayanadi).
func utc(h, m, s int) time.Time {
	return time.Date(2026, 8, 1, h, m, s, 0, time.UTC)
}

func ptr(s string) *string { return &s }

// baseData — sarlavhasi va mentori aniq, xabarsiz transkript ma'lumoti.
func baseData(msgs ...*entity.ChatMessage) chat.TranscriptData {
	return chat.TranscriptData{
		LessonTitle:  "Matematika 5-sinf",
		LessonDate:   utc(9, 30, 0), // 14:30 Toshkent
		MentorName:   "Dilnoza Karimova",
		HostIdentity: "mentor1",
		Messages:     msgs,
	}
}

// ─── Sof funksiya: TXT ───────────────────────────────────────────────────────

func TestRenderTXT_Sarlavha(t *testing.T) {
	out := chat.RenderTXT(baseData())
	require.Contains(t, out, "Dars: Matematika 5-sinf\n")
	require.Contains(t, out, "Mentor: Dilnoza Karimova\n")
}

// Vaqt mintaqasi: DB UTC saqlaydi, transkript esa Toshkent (UTC+5) ko'rsatadi.
// Bu server `time.Local` iga bog'liq BO'LMASLIGI kerak.
func TestRenderTXT_ToshkentVaqti(t *testing.T) {
	out := chat.RenderTXT(baseData(&entity.ChatMessage{
		SenderIdentity: "guest_a", SenderName: "Ali Valiyev",
		Body: "Ustoz, savol bor", CreatedAt: utc(9, 32, 5),
	}))
	require.Contains(t, out, "Sana: 01.08.2026 14:30\n", "sarlavha sanasi +05:00 da bo'lishi kerak")
	require.Contains(t, out, "14:32:05  Ali Valiyev: Ustoz, savol bor\n")
	require.NotContains(t, out, "09:32:05", "UTC vaqt chiqmasligi kerak")
}

// Sana chegarasi: UTC'da 31-iyul kechqurun = Toshkentda 1-avgust.
func TestRenderTXT_SanaChegarasi(t *testing.T) {
	d := baseData(&entity.ChatMessage{
		SenderName: "Ali", Body: "kech", CreatedAt: time.Date(2026, 7, 31, 20, 15, 0, 0, time.UTC),
	})
	d.LessonDate = time.Date(2026, 7, 31, 20, 0, 0, 0, time.UTC)
	out := chat.RenderTXT(d)
	require.Contains(t, out, "Sana: 01.08.2026 01:00\n")
	require.Contains(t, out, "01:15:00  Ali: kech\n")
}

// Ko'p qatorli xabar HAQIQIY satrga o'xshab ketmasligi kerak: o'quvchi
// tanasiga soxta `HH:MM:SS  Ustoz: …` qatori yozib, transkriptni qalbakilashtira
// olmasin (transkript nizoda dalil bo'ladi).
func TestRenderTXT_KopQatorSuriladi(t *testing.T) {
	out := chat.RenderTXT(baseData(&entity.ChatMessage{
		SenderIdentity: "guest_a", SenderName: "Ali",
		Body:      "salom\r\n14:59:00  Siz: darsdan chiqing",
		CreatedAt: utc(9, 33, 0),
	}))
	require.Contains(t, out, "14:33:00  Ali: salom\n          14:59:00  Siz: darsdan chiqing\n")
	require.NotContains(t, out, "\n14:59:00  Siz:", "soxta satr ustunga tekis tushmasligi kerak")
	require.NotContains(t, out, "\r")
}

func TestRenderTXT_BoshChat(t *testing.T) {
	out := chat.RenderTXT(baseData())
	require.Contains(t, out, "(Chatda xabar bo'lmagan)")
	require.NotContains(t, out, ":  ", "xabar satri umuman bo'lmasligi kerak")
}

// Host xabari «Siz» deb ko'rsatiladi (transkriptni ustoz o'zi yuklab oladi).
func TestRenderTXT_HostSiz(t *testing.T) {
	out := chat.RenderTXT(baseData(&entity.ChatMessage{
		SenderIdentity: "mentor1", SenderName: "Dilnoza Karimova",
		Body: "Marhamat", CreatedAt: utc(9, 32, 40),
	}))
	require.Contains(t, out, "14:32:40  Siz: Marhamat\n")
}

func TestRenderTXT_ShaxsiyXabar(t *testing.T) {
	out := chat.RenderTXT(baseData(&entity.ChatMessage{
		SenderIdentity: "guest_b", SenderName: "Dilnoza",
		Body: "faqat sizga", ToIdentity: ptr("mentor1"), CreatedAt: utc(9, 35, 12),
	}))
	require.Contains(t, out, "14:35:12  Dilnoza (shaxsiy): faqat sizga\n")
}

func TestRenderTXT_FaylXabari(t *testing.T) {
	out := chat.RenderTXT(baseData(
		&entity.ChatMessage{
			SenderIdentity: "guest_a", SenderName: "Ali Valiyev",
			CreatedAt: utc(9, 40, 3),
			File:      &entity.ChatFile{Name: "masala.pdf", Size: 1234, Mime: "application/pdf"},
		},
		&entity.ChatMessage{
			SenderIdentity: "guest_a", SenderName: "Ali Valiyev",
			Body: "shu masala", CreatedAt: utc(9, 41, 0),
			File: &entity.ChatFile{Name: "ikki.pdf"},
		},
	))
	require.Contains(t, out, "14:40:03  Ali Valiyev: 📎 masala.pdf\n")
	require.Contains(t, out, "14:41:00  Ali Valiyev: 📎 ikki.pdf — shu masala\n")
}

// Tartib saqlanadi: transkript suhbat tartibida o'qiladi.
func TestRenderTXT_Tartib(t *testing.T) {
	out := chat.RenderTXT(baseData(
		&entity.ChatMessage{SenderName: "A", Body: "birinchi", CreatedAt: utc(9, 31, 0)},
		&entity.ChatMessage{SenderName: "B", Body: "ikkinchi", CreatedAt: utc(9, 32, 0)},
	))
	require.Less(t, strings.Index(out, "birinchi"), strings.Index(out, "ikkinchi"))
}

// ─── Sof funksiya: HTML ──────────────────────────────────────────────────────

func TestRenderHTML_Ozini_Tutadi(t *testing.T) {
	out := chat.RenderHTML(baseData(&entity.ChatMessage{
		SenderName: "Ali", Body: "salom", CreatedAt: utc(9, 32, 5),
	}))
	require.True(t, strings.HasPrefix(out, "<!DOCTYPE html>"))
	require.Contains(t, out, "<style>", "CSS inline bo'lishi kerak")
	require.Contains(t, out, "Matematika 5-sinf")
	require.Contains(t, out, "14:32:05")
	// Offline ochilishi shart — tashqi resurs bo'lmasligi kerak.
	require.NotContains(t, out, "http://")
	require.NotContains(t, out, "https://")
	require.NotContains(t, out, "<link")
	require.NotContains(t, out, "<script")
}

// XSS — eng muhim test: hujjat brauzerda ochiladi va MAZMUNI o'quvchi yozgan.
func TestRenderHTML_XSS(t *testing.T) {
	d := chat.TranscriptData{
		LessonTitle:  `Dars <img src=x onerror=alert(1)>`,
		LessonDate:   utc(9, 30, 0),
		MentorName:   `<b>Mentor</b>`,
		HostIdentity: "mentor1",
		Messages: []*entity.ChatMessage{
			{
				SenderIdentity: "guest_a",
				SenderName:     `<script>alert("ism")</script>`,
				Body:           `<script>alert("matn")</script>`,
				CreatedAt:      utc(9, 33, 0),
			},
			{
				SenderIdentity: "guest_b", SenderName: "Fayl",
				CreatedAt: utc(9, 34, 0),
				File:      &entity.ChatFile{Name: `<script>alert("fayl")</script>.pdf`},
			},
			{
				SenderIdentity: "guest_c", SenderName: "DM",
				Body: `"><script>alert(1)</script>`, ToIdentity: ptr("mentor1"),
				CreatedAt: utc(9, 35, 0),
			},
		},
	}
	out := chat.RenderHTML(d)

	// Bitta ham ijro etiladigan teg qolmasligi kerak.
	require.NotContains(t, out, "<script>alert", "foydalanuvchi matni escape qilinmagan")
	require.NotContains(t, out, "<img src=x")
	require.NotContains(t, out, "<b>Mentor</b>")
	// Mazmun esa YO'QOLMASLIGI kerak — escape qilingan ko'rinishda turadi.
	require.Contains(t, out, "&lt;script&gt;alert(&#34;ism&#34;)&lt;/script&gt;")
	require.Contains(t, out, "&lt;script&gt;alert(&#34;matn&#34;)&lt;/script&gt;")
	require.Contains(t, out, "&lt;script&gt;alert(&#34;fayl&#34;)&lt;/script&gt;.pdf")
	require.Contains(t, out, "&lt;img src=x onerror=alert(1)&gt;")
}

func TestRenderHTML_BoshChat(t *testing.T) {
	out := chat.RenderHTML(baseData())
	require.Contains(t, out, "Chatda xabar bo&#39;lmagan")
	require.Contains(t, out, "0 ta xabar")
}

func TestRenderHTML_ShaxsiyVaHost(t *testing.T) {
	out := chat.RenderHTML(baseData(
		&entity.ChatMessage{
			SenderIdentity: "mentor1", SenderName: "Dilnoza",
			Body: "javob", CreatedAt: utc(9, 36, 0),
		},
		&entity.ChatMessage{
			SenderIdentity: "guest_b", SenderName: "Ali",
			Body: "shaxsiy savol", ToIdentity: ptr("mentor1"), CreatedAt: utc(9, 37, 0),
		},
	))
	require.Contains(t, out, `class="msg self"`, "host xabari ajratilishi kerak")
	require.Contains(t, out, "Siz")
	require.Contains(t, out, `<span class="private">shaxsiy</span>`)
}

// ─── Usecase simlash ─────────────────────────────────────────────────────────

func TestTranscript_Formatlar(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()
	_, err := uc.Send(ctx, "mentor1", testLessonID, "Salom", "")
	require.NoError(t, err)

	txt, err := uc.Transcript(ctx, "mentor1", testLessonID, "txt")
	require.NoError(t, err)
	require.Equal(t, "text/plain; charset=utf-8", txt.ContentType)
	require.Contains(t, string(txt.Body), "Siz: Salom")

	htm, err := uc.Transcript(ctx, "mentor1", testLessonID, "html")
	require.NoError(t, err)
	require.Equal(t, "text/html; charset=utf-8", htm.ContentType)
	require.Contains(t, string(htm.Body), "<!DOCTYPE html>")

	// Format berilmasa TXT (klient eng sodda holatda parametr yozmasin).
	def, err := uc.Transcript(ctx, "mentor1", testLessonID, "")
	require.NoError(t, err)
	require.Equal(t, "text/plain; charset=utf-8", def.ContentType)
}

func TestTranscript_NotoughriFormat(t *testing.T) {
	uc, _, _ := setupFull(t)
	_, err := uc.Transcript(context.Background(), "mentor1", testLessonID, "pdf")
	require.True(t, apperr.IsBadRequest(err))
}

func TestTranscript_Egalik(t *testing.T) {
	uc, _, _ := setupFull(t)
	_, err := uc.Transcript(context.Background(), "intruder", testLessonID, "txt")
	require.True(t, apperr.IsForbidden(err), "begona mentor transkript ola olmaydi")
}

// Fayl nomi `Content-Disposition` ga tushadi — sarlavhadagi qo'shtirnoq,
// yangi qator va boshqa belgilar undan CHIQIB KETMASLIGI kerak.
func TestTranscript_FaylNomiXavfsiz(t *testing.T) {
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", FullName: "M", Role: "mentor"}))
	started := utc(9, 30, 0)
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusEnded,
		Title:     "Algebra \"5\"\r\nX-Injected: 1; rm -rf /",
		StartedAt: &started,
	}))
	uc := chat.New(testutil.NewFakeChatRepo(), lrepo, urepo, testutil.NewFakeLiveKit(),
		testutil.NewFakeMinio(), testutil.NewFakeCache(), testutil.NewLogger())

	tr, err := uc.Transcript(context.Background(), "mentor1", testLessonID, "txt")
	require.NoError(t, err)
	require.Equal(t, "algebra-5-x-injected-1-rm-rf-chat-2026-08-01.txt", tr.Filename)
	for _, bad := range []string{`"`, "\r", "\n", ";", " ", "/"} {
		require.NotContains(t, tr.Filename, bad)
	}
}

// Sarlavhada ASCII belgi umuman bo'lmasa ham nom yaroqli qolishi kerak.
func TestTranscript_KirillSarlavha(t *testing.T) {
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(context.Background(), &entity.User{ID: "mentor1", FullName: "M", Role: "mentor"}))
	started := utc(9, 30, 0)
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Title: "Матиматика", StartedAt: &started,
	}))
	uc := chat.New(testutil.NewFakeChatRepo(), lrepo, urepo, testutil.NewFakeLiveKit(),
		testutil.NewFakeMinio(), testutil.NewFakeCache(), testutil.NewLogger())

	tr, err := uc.Transcript(context.Background(), "mentor1", testLessonID, "html")
	require.NoError(t, err)
	require.Equal(t, "dars-chat-2026-08-01.html", tr.Filename)
}

// O'chirilgan (moderatsiya qilingan) xabar transkriptga TUSHMASLIGI kerak —
// aks holda "o'chirdim" degan amal fayl orqali bekor bo'lardi.
func TestTranscript_OchirilganChiqmaydi(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()
	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "yomon gap", "")
	require.NoError(t, err)
	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, m.ID))

	tr, err := uc.Transcript(ctx, "mentor1", testLessonID, "txt")
	require.NoError(t, err)
	require.NotContains(t, string(tr.Body), "yomon gap")
	require.Contains(t, string(tr.Body), "(Chatda xabar bo'lmagan)")
}

// Begona shaxsiy yozishma (o'quvchidan o'quvchiga) ustozning transkriptiga
// tushmaydi — `History` bilan bir xil maxfiylik qoidasi.
func TestTranscript_BegonaDMChiqmaydi(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()
	_, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "faqat senga", "guest_b")
	require.NoError(t, err)

	tr, err := uc.Transcript(ctx, "mentor1", testLessonID, "txt")
	require.NoError(t, err)
	require.NotContains(t, string(tr.Body), "faqat senga")
}
