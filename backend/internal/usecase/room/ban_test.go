package room_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/chat"
	"github.com/zoom/darsly/internal/usecase/poll"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/roomstate"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// C-2 — "chiqarib yuborish" AMALDA ishlashi.
//
// Auditda topilgan holat: `RemoveParticipant` LiveKit'da faqat joriy ulanishni
// uzardi va ban FAQAT yangi token so'ralganda tekshirilardi. Chiqarilgan
// buzg'unchi esa qo'lidagi eski token bilan to'g'ridan-to'g'ri SFU'ga qaytib
// ulanardi (backendga umuman murojaat qilmasdan) va chat/qo'l/reaksiya/ovoz
// endpointlari uni bemalol qabul qilaverardi — ya'ni ustozning moderatsiya
// tugmalari bor edi, kuchi yo'q edi.
//
// Quyidagi testlar aynan shu zanjirni qoplaydi.

// banSetup — jonli dars + chiqarilgan bitta ishtirokchi.
func banSetup(t *testing.T) (
	uc room.UseCase,
	lrepo *testutil.FakeLessonRepo,
	lk *testutil.FakeLiveKit,
	cache *testutil.FakeCache,
) {
	uc, lrepo, lk, cache, _ = banSetupWithBlocklist(t)
	return uc, lrepo, lk, cache
}

// banSetupWithBlocklist — banSetup + qora ro'yxat fake'iga kirish (№4 testlari).
func banSetupWithBlocklist(t *testing.T) (
	uc room.UseCase,
	lrepo *testutil.FakeLessonRepo,
	lk *testutil.FakeLiveKit,
	cache *testutil.FakeCache,
	blk *testutil.FakeBlocklistRepo,
) {
	t.Helper()
	ctx := context.Background()
	lrepo = testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(ctx, &entity.User{ID: "mentor1", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive,
	}))
	lk = testutil.NewFakeLiveKit()
	cache = testutil.NewFakeCache()
	blk = testutil.NewFakeBlocklistRepo()
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	uc = room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), nil, hands, blk)
	return uc, lrepo, lk, cache, blk
}

func TestRemoveParticipant_BansAndDisconnects(t *testing.T) {
	uc, _, lk, cache := banSetup(t)
	ctx := context.Background()

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", ""))

	require.Equal(t, []string{"buzgunchi"}, lk.RemovedList(), "AYNAN o'sha ishtirokchi uzilishi kerak")
	require.True(t, shared.IsBanned(ctx, cache, testLessonID, "buzgunchi"), "ban yozilishi kerak")
	require.False(t, shared.IsBanned(ctx, cache, testLessonID, "boshqa"), "boshqalar banlanmasin")
}

// ⭐ Asosiy C-2 testi: eski token bilan qaytib kelgan buzg'unchi
// `participant_joined` webhook'ida darhol uziladi.
func TestEnforceJoin_KicksBannedReconnect(t *testing.T) {
	uc, _, lk, _ := banSetup(t)
	ctx := context.Background()
	rn := shared.RoomName(testLessonID)

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", ""))
	require.Len(t, lk.RemovedList(), 1)

	// Buzg'unchi backendga murojaat qilmasdan, eski tokeni bilan qaytib ulandi.
	uc.EnforceJoin(ctx, rn, "buzgunchi", "Buzg'unchi")

	require.Equal(t, []string{"buzgunchi", "buzgunchi"}, lk.RemovedList(),
		"qaytib kirgan chiqarilgan ishtirokchi yana uzilishi kerak")
}

func TestEnforceJoin_AllowsNormalParticipants(t *testing.T) {
	uc, _, lk, _ := banSetup(t)
	ctx := context.Background()

	uc.EnforceJoin(ctx, shared.RoomName(testLessonID), "halol-oquvchi", "Halol")

	require.Empty(t, lk.RemovedList(), "banlanmagan ishtirokchi uzilmasligi kerak")
}

func TestEnforceJoin_IgnoresForeignRoomAndEmptyIdentity(t *testing.T) {
	uc, _, lk, _ := banSetup(t)
	ctx := context.Background()

	uc.EnforceJoin(ctx, "begona-xona", "kimdir", "Kimdir")
	uc.EnforceJoin(ctx, shared.RoomName(testLessonID), "", "")

	require.Empty(t, lk.RemovedList())
}

// Ban XONA ICHIDAGI barcha ochiq endpointlarda qo'llanishi kerak — bittasi
// qolib ketsa buzg'unchi o'sha yo'l bilan darsni buzishda davom etadi.
func TestBannedParticipant_BlockedOnEveryOpenPath(t *testing.T) {
	uc, lrepo, lk, cache := banSetup(t)
	ctx := context.Background()
	log := testutil.NewLogger()

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", ""))

	stateUC := roomstate.New(lrepo, lk, cache, nil, log)
	chatUC := chat.New(testutil.NewFakeChatRepo(), lrepo, testutil.NewFakeUserRepo(), lk, testutil.NewFakeMinio(), cache, log)
	pollUC := poll.New(testutil.NewFakePollRepo(), lrepo, lk, cache, log)

	p, err := pollUC.Create(ctx, "mentor1", testLessonID, "Savol?", []string{"Ha", "Yo'q"}, "")
	require.NoError(t, err)

	t.Run("qo'l ko'tarish", func(t *testing.T) {
		err := stateUC.SetHand(ctx, testLessonID, "buzgunchi", "Buzg'unchi", true)
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
	})
	t.Run("reaksiya", func(t *testing.T) {
		err := stateUC.Reaction(ctx, testLessonID, "buzgunchi", "Buzg'unchi", "🎉")
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
	})
	t.Run("chat yozish", func(t *testing.T) {
		_, err := chatUC.SendFromRoom(ctx, testLessonID, "buzgunchi", "Buzg'unchi", "salom", "")
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
	})
	t.Run("chat tarixini o'qish", func(t *testing.T) {
		_, err := chatUC.HistoryForRoom(ctx, testLessonID, "buzgunchi", nil, 20)
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
	})
	t.Run("ovoz berish", func(t *testing.T) {
		err := pollUC.Vote(ctx, p.ID, "buzgunchi", shared.RoomName(testLessonID), 0)
		require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
	})

	// Nazorat: halol o'quvchi uchun aynan shu yo'llar OCHIQ qolishi kerak —
	// aks holda test "hammasini bloklash" degan noto'g'ri fixni ham yashil ko'rsatardi.
	t.Run("halol o'quvchi ta'sirlanmaydi", func(t *testing.T) {
		require.NoError(t, stateUC.SetHand(ctx, testLessonID, "halol", "Halol", true))
		_, err := chatUC.SendFromRoom(ctx, testLessonID, "halol", "Halol", "salom", "")
		require.NoError(t, err)
		require.NoError(t, pollUC.Vote(ctx, p.ID, "halol", shared.RoomName(testLessonID), 0))
	})
}

// Yakunlangan darsda ochiq endpointlar yopilishi kerak: xona o'chirilgan, lekin
// endpointlar ochiq qolib chat yozilishda davom etardi.
func TestEndedLesson_ClosesOpenPaths(t *testing.T) {
	uc, lrepo, lk, cache := banSetup(t)
	ctx := context.Background()
	log := testutil.NewLogger()
	stateUC := roomstate.New(lrepo, lk, cache, nil, log)
	chatUC := chat.New(testutil.NewFakeChatRepo(), lrepo, testutil.NewFakeUserRepo(), lk, testutil.NewFakeMinio(), cache, log)

	// Dars jonli — amallar ishlaydi (va "jonli" keshiga tushadi).
	require.NoError(t, stateUC.SetHand(ctx, testLessonID, "oquvchi", "O'quvchi", true))

	require.NoError(t, uc.EndLesson(ctx, "mentor1", testLessonID))

	// EndLesson keshni tozalashi SHART — aks holda yakunlangan dars TTL
	// tugagunicha jonli deb qabul qilinardi.
	err := stateUC.SetHand(ctx, testLessonID, "oquvchi", "O'quvchi", true)
	require.True(t, isBadRequest(err), "yakunlangan darsga qo'l ko'tarilmasin, olindi: %v", err)

	_, err = chatUC.SendFromRoom(ctx, testLessonID, "oquvchi", "O'quvchi", "salom", "")
	require.True(t, isBadRequest(err), "yakunlangan darsga chat yozilmasin, olindi: %v", err)
}

// isBadRequest — `apperr` da IsBadRequest predikati yo'q (faqat NotFound /
// Conflict / Validation / Forbidden bor), shuning uchun kod bu yerda tekshiriladi.
func isBadRequest(err error) bool {
	ae := apperr.As(err)
	return ae != nil && ae.Code == apperr.CodeBadRequest
}

// ─── №4 — ban tanlovi: bir darslik / mentor bo'yicha doimiy ─────────────────

// scope=mentor: kick doimiy qora ro'yxatga yozadi (ism xonadagi ro'yxatdan
// olinadi) VA odatdagi dars-ban + uzish ham qo'llanadi.
func TestRemoveParticipant_MentorScope_AddsBlocklist(t *testing.T) {
	uc, _, lk, cache, blk := banSetupWithBlocklist(t)
	ctx := context.Background()
	lk.Participants = []entity.RoomParticipant{
		{Identity: "buzgunchi", Name: "Bezori Aka"},
		{Identity: "halol", Name: "Halol O'quvchi"},
	}

	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", entity.BanScopeMentor))

	// Doimiy ro'yxatda ism bo'yicha (katta-kichik harf farqsiz).
	blocked, err := blk.IsBlocked(ctx, "mentor1", "bezori aka")
	require.NoError(t, err)
	require.True(t, blocked, "scope=mentor doimiy qora ro'yxatga yozishi kerak")
	// Dars-ban ham qo'llanadi (joriy token oynasini yopish uchun).
	require.True(t, shared.IsBanned(ctx, cache, testLessonID, "buzgunchi"))
	require.Equal(t, []string{"buzgunchi"}, lk.RemovedList())

	// Ro'yxat mentorga ko'rinadi va o'chirilsa (unban) qayta kira oladi.
	items, err := uc.ListBlocklist(ctx, "mentor1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Bezori Aka", items[0].DisplayName)
	require.NoError(t, uc.Unblock(ctx, "mentor1", items[0].ID))
	blocked, _ = blk.IsBlocked(ctx, "mentor1", "Bezori Aka")
	require.False(t, blocked, "unban'dan keyin ism ro'yxatdan chiqishi kerak")
}

// Noto'g'ri scope — 400 (jimgina "lesson"ga tushib qolmasin: ustoz "doimiy"
// deb o'ylagan amal aslida bir darslik bo'lib qolardi).
func TestRemoveParticipant_InvalidScope(t *testing.T) {
	uc, _, lk, _, _ := banSetupWithBlocklist(t)
	err := uc.RemoveParticipant(context.Background(), "mentor1", testLessonID, "g", "forever")
	require.True(t, isBadRequest(err), "kutilgan 400, olindi: %v", err)
	require.Empty(t, lk.RemovedList(), "yaroqsiz scope'da hech kim uzilmasin")
}

// Doimiy ro'yxat yozilmasa amal TO'XTAYDI — ustozga "doimiy ban qo'yildi"
// degan yolg'on muvaffaqiyat ko'rsatilmaydi.
func TestRemoveParticipant_MentorScope_FailsClosed(t *testing.T) {
	uc, _, lk, _, blk := banSetupWithBlocklist(t)
	blk.FailNext = context.DeadlineExceeded
	err := uc.RemoveParticipant(context.Background(), "mentor1", testLessonID, "buzgunchi", entity.BanScopeMentor)
	require.Error(t, err)
	require.Empty(t, lk.RemovedList(), "ban yozilmagan bo'lsa uzish ham bo'lmasin (retry mumkin)")
}

// ⭐ Doimiy bloklangan ism mentorning BOSHQA darsiga ham kira olmaydi:
// `participant_joined` webhook'ida ism bo'yicha uziladi (№4).
func TestEnforceJoin_KicksMentorBlockedName(t *testing.T) {
	uc, lrepo, lk, _, blk := banSetupWithBlocklist(t)
	ctx := context.Background()
	lk.Participants = []entity.RoomParticipant{{Identity: "buzgunchi", Name: "Bezori"}}
	require.NoError(t, uc.RemoveParticipant(ctx, "mentor1", testLessonID, "buzgunchi", entity.BanScopeMentor))

	// Mentorning YANGI darsi — dars-ban bu yerda amal qilmaydi.
	other := "22222222-2222-4222-8222-222222222222"
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{ID: other, MentorID: "mentor1", Status: entity.LessonStatusLive}))

	// Xuddi shu ism yangi identity bilan qaytib keldi (yangi join = yangi guest id).
	uc.EnforceJoin(ctx, shared.RoomName(other), "guest_yangi", "bezori")
	require.Contains(t, lk.RemovedList(), "guest_yangi", "bloklangan ism boshqa darsda ham uzilishi kerak")

	// Boshqa ismli halol o'quvchi kiradi.
	uc.EnforceJoin(ctx, shared.RoomName(other), "guest_h", "Halol")
	require.NotContains(t, lk.RemovedList(), "guest_h")

	// Blocklist tekshiruvi FAIL bo'lsa — fail-open (butun sinf to'silmaydi).
	blk.FailNext = context.DeadlineExceeded
	uc.EnforceJoin(ctx, shared.RoomName(other), "guest_f", "Kimdir")
	require.NotContains(t, lk.RemovedList(), "guest_f")
}

// ─── №11 — ovoz siyosati (Zoom modeli) ──────────────────────────────────────

// audioSetup — ovoz siyosati testlari uchun dars (bayroqlar parametrda).
func audioSetup(t *testing.T, muteOnEntry, allowSelfUnmute bool) (room.UseCase, *testutil.FakeLiveKit) {
	t.Helper()
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	urepo := testutil.NewFakeUserRepo()
	require.NoError(t, urepo.Create(ctx, &entity.User{ID: "mentor1", FullName: "Dilnoza", Role: "mentor"}))
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive,
		MuteOnEntry: muteOnEntry, AllowSelfUnmute: allowSelfUnmute,
	}))
	lk := testutil.NewFakeLiveKit()
	cache := testutil.NewFakeCache()
	hands := roomstate.New(lrepo, lk, cache, nil, testutil.NewLogger())
	return room.New(lrepo, urepo, lk, cache, testutil.NewLogger(), nil, hands, testutil.NewFakeBlocklistRepo()), lk
}

// mute_on_entry: BIRINCHI audio publish mute qilinadi, KEYINGISI tegilmaydi —
// "kirganda mute", "hech qachon gapira olmaydi" emas.
func TestEnforceAudioPolicy_MuteOnEntry_OnlyFirstPublish(t *testing.T) {
	uc, lk := audioSetup(t, true, true)
	ctx := context.Background()
	rn := shared.RoomName(testLessonID)

	uc.EnforceAudioPolicy(ctx, rn, "oquvchi")
	require.Equal(t, []string{"oquvchi"}, lk.MutedList(), "birinchi publish mute bilan boshlanadi")

	// O'quvchi keyin o'zi mikrofon yoqdi (qayta publish) — endi erkin.
	uc.EnforceAudioPolicy(ctx, rn, "oquvchi")
	require.Equal(t, []string{"oquvchi"}, lk.MutedList(), "ikkinchi publish mute qilinMAsligi kerak (allow_self_unmute=true)")
}

// allow_self_unmute=false: HAR audio publish qayta mute qilinadi (Zoom'ning
// "unmute taqiqlangan" xulqi).
func TestEnforceAudioPolicy_SelfUnmuteForbidden_EveryPublish(t *testing.T) {
	uc, lk := audioSetup(t, false, false)
	ctx := context.Background()
	rn := shared.RoomName(testLessonID)

	uc.EnforceAudioPolicy(ctx, rn, "oquvchi")
	uc.EnforceAudioPolicy(ctx, rn, "oquvchi")
	require.Equal(t, []string{"oquvchi", "oquvchi"}, lk.MutedList(), "har publish qayta mute qilinishi kerak")
}

// Ovoz siyosati HOST'ga qo'llanmaydi va o'chiq bayroqlar hech kimni mute qilmaydi.
func TestEnforceAudioPolicy_SkipsHostAndDisabledFlags(t *testing.T) {
	ctx := context.Background()

	t.Run("host erkin", func(t *testing.T) {
		uc, lk := audioSetup(t, true, false) // eng qattiq rejim ham
		uc.EnforceAudioPolicy(ctx, shared.RoomName(testLessonID), "mentor1")
		require.Empty(t, lk.MutedList(), "ustoz hech qachon avto-mute qilinmaydi")
	})

	t.Run("bayroqlar o'chiq — hech kim mute bo'lmaydi", func(t *testing.T) {
		uc, lk := audioSetup(t, false, true)
		uc.EnforceAudioPolicy(ctx, shared.RoomName(testLessonID), "oquvchi")
		require.Empty(t, lk.MutedList())
	})

	t.Run("begona xona / bo'sh identity e'tiborsiz", func(t *testing.T) {
		uc, lk := audioSetup(t, true, false)
		uc.EnforceAudioPolicy(ctx, "begona-xona", "kimdir")
		uc.EnforceAudioPolicy(ctx, shared.RoomName(testLessonID), "")
		require.Empty(t, lk.MutedList())
	})
}

// MuteAll allow_self_unmute bayrog'ini ham yangilaydi (Zoom dialogidagi checkbox).
func TestMuteAll_UpdatesAllowSelfUnmute(t *testing.T) {
	uc, lrepo, lk, _, _ := banSetupWithBlocklist(t)
	ctx := context.Background()
	// Boshlanish: allow_self_unmute default false (fake'da nol qiymat) — true qilamiz.
	lk.Participants = []entity.RoomParticipant{{Identity: "mentor1"}, {Identity: "u1"}, {Identity: "u2"}}

	f := false
	require.NoError(t, uc.MuteAll(ctx, "mentor1", testLessonID, &f))
	l, err := lrepo.GetByID(ctx, testLessonID)
	require.NoError(t, err)
	require.False(t, l.AllowSelfUnmute)
	require.Len(t, lk.MutedList(), 2, "host'dan tashqari hamma mute")

	tr := true
	require.NoError(t, uc.MuteAll(ctx, "mentor1", testLessonID, &tr))
	l, err = lrepo.GetByID(ctx, testLessonID)
	require.NoError(t, err)
	require.True(t, l.AllowSelfUnmute, "bayroq jonli yangilanishi kerak")

	// nil — bayroqqa tegilmaydi.
	require.NoError(t, uc.MuteAll(ctx, "mentor1", testLessonID, nil))
	l, _ = lrepo.GetByID(ctx, testLessonID)
	require.True(t, l.AllowSelfUnmute, "nil bayroqni o'zgartirmasligi kerak")
}
