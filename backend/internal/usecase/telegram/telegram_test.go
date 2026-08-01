package telegram_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/telegram"
)

const (
	mentorA = "11111111-1111-4111-8111-111111111111"
	mentorB = "22222222-2222-4222-8222-222222222222"
)

func setup(t *testing.T) (telegram.UseCase, *testutil.FakeTelegramRepo, *testutil.FakeTelegram) {
	t.Helper()
	repo := testutil.NewFakeTelegramRepo()
	repo.AddUser(mentorA, "Ali")
	repo.AddUser(mentorB, "Vali")
	bot := testutil.NewFakeTelegram()
	return telegram.New(repo, bot, testutil.NewFakeCache(), testutil.NewLogger()), repo, bot
}

// Bog'lash oqimining to'liq yo'li: kod olindi → botga yuborildi → bog'landi.
func TestLinkFlow(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()

	link, err := uc.StartLink(ctx, mentorA)
	require.NoError(t, err)
	require.NotEmpty(t, link.Code)
	require.Positive(t, link.ExpiresInS)
	require.Contains(t, link.DeepLink, link.Code, "deep-link kodni o'z ichiga olishi kerak")

	u, err := uc.RedeemCode(ctx, link.Code, 555, "ali_tg")
	require.NoError(t, err)
	require.Equal(t, mentorA, u.ID)

	st, err := uc.Status(ctx, mentorA)
	require.NoError(t, err)
	require.True(t, st.Linked)
	require.NotNil(t, st.Username)
	require.Equal(t, "ali_tg", *st.Username)
}

// ⭐ Kod BIR MARTALIK. Aks holda skrinshotga tushgan kodni ko'rgan boshqa odam
// mentorga bog'lanib, uning yozuvlarini o'z guruhiga yubora olardi.
func TestRedeemCode_SingleUse(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()

	link, err := uc.StartLink(ctx, mentorA)
	require.NoError(t, err)

	_, err = uc.RedeemCode(ctx, link.Code, 555, "ali_tg")
	require.NoError(t, err)

	_, err = uc.RedeemCode(ctx, link.Code, 999, "boshqa")
	require.Error(t, err, "kod ikkinchi marta ishlamasligi kerak")
	require.True(t, apperr.IsBadRequest(err))
}

// Noma'lum/muddati o'tgan kod bir xil javob beradi (classification oracle yo'q).
func TestRedeemCode_Invalid(t *testing.T) {
	uc, _, _ := setup(t)
	_, err := uc.RedeemCode(context.Background(), "YOQKOD12", 555, "x")
	require.True(t, apperr.IsBadRequest(err))
}

// Bitta Telegram akkaunti ikkinchi hisobga bog'lansa BIRINCHISIDAN uziladi
// (UNIQUE indeks bilan to'qnashmasin va ikki mentor bir akkauntni
// ulashmasin).
func TestRedeemCode_ReassignsTelegramAccount(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()

	l1, _ := uc.StartLink(ctx, mentorA)
	_, err := uc.RedeemCode(ctx, l1.Code, 555, "tg")
	require.NoError(t, err)

	l2, _ := uc.StartLink(ctx, mentorB)
	_, err = uc.RedeemCode(ctx, l2.Code, 555, "tg")
	require.NoError(t, err)

	stA, _ := uc.Status(ctx, mentorA)
	stB, _ := uc.Status(ctx, mentorB)
	require.False(t, stA.Linked, "eski egadan uzilishi kerak")
	require.True(t, stB.Linked)
}

// Uzish idempotent: bog'lanmagan holatda ham xato bermaydi.
func TestUnlink_Idempotent(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	require.NoError(t, uc.Unlink(ctx, mentorA))
	require.NoError(t, uc.Unlink(ctx, mentorA))
}

// Integratsiya o'chiq bo'lsa Status DB'ga bormasdan `enabled=false` beradi va
// bog'lash umuman boshlanmaydi.
func TestDisabledIntegration(t *testing.T) {
	repo := testutil.NewFakeTelegramRepo()
	bot := testutil.NewFakeTelegram()
	bot.Disabled = true
	uc := telegram.New(repo, bot, testutil.NewFakeCache(), testutil.NewLogger())
	ctx := context.Background()

	st, err := uc.Status(ctx, mentorA)
	require.NoError(t, err)
	require.False(t, st.Enabled)
	require.False(t, st.Linked)

	_, err = uc.StartLink(ctx, mentorA)
	require.True(t, apperr.IsBadRequest(err))
}

// Bot guruhga qo'shilganda guruh QO'SHGAN mentorga biriktiriladi.
func TestHandleChatMembership_AssignsOwner(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	link, _ := uc.StartLink(ctx, mentorA)
	_, err := uc.RedeemCode(ctx, link.Code, 555, "tg")
	require.NoError(t, err)

	err = uc.HandleChatMembership(ctx, entity.TelegramChat{
		ChatID: -1001, Title: "9-A sinf", Type: "supergroup",
	}, 555, true)
	require.NoError(t, err)

	chats, err := uc.ChatsForMentor(ctx, mentorA)
	require.NoError(t, err)
	require.Len(t, chats, 1)
	require.Equal(t, "9-A sinf", chats[0].Title)
}

// Bot chiqarilganda guruh ro'yxatdan tushadi (qator esa saqlanadi).
func TestHandleChatMembership_Removal(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	link, _ := uc.StartLink(ctx, mentorA)
	_, _ = uc.RedeemCode(ctx, link.Code, 555, "tg")

	chat := entity.TelegramChat{ChatID: -1001, Title: "9-A sinf", Type: "supergroup"}
	require.NoError(t, uc.HandleChatMembership(ctx, chat, 555, true))
	require.NoError(t, uc.HandleChatMembership(ctx, chat, 555, false))

	chats, err := uc.ChatsForMentor(ctx, mentorA)
	require.NoError(t, err)
	require.Empty(t, chats)
}

// ⭐ EGALIK: begona mentor boshqa birovning guruhiga yubora olmaydi.
//
// `callback_data` ni bosgan odam o'zgartira oladi, ya'ni bu tekshiruv
// bo'lmasa istalgan chat ID ga video yuborish mumkin bo'lardi.
func TestAuthorizeChat_ForeignMentorDenied(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	l1, _ := uc.StartLink(ctx, mentorA)
	_, _ = uc.RedeemCode(ctx, l1.Code, 555, "tg")
	require.NoError(t, uc.HandleChatMembership(ctx,
		entity.TelegramChat{ChatID: -1001, Title: "A guruh", Type: "supergroup"}, 555, true))

	_, err := uc.AuthorizeChat(ctx, mentorA, -1001)
	require.NoError(t, err, "egasi yubora olishi kerak")

	_, err = uc.AuthorizeChat(ctx, mentorB, -1001)
	require.True(t, apperr.IsForbidden(err), "begona mentor rad etilishi kerak")
}

// Bot chiqarilgan guruhga yuborishga urinish aniq xato beradi.
func TestAuthorizeChat_InactiveChat(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()
	l1, _ := uc.StartLink(ctx, mentorA)
	_, _ = uc.RedeemCode(ctx, l1.Code, 555, "tg")
	chat := entity.TelegramChat{ChatID: -1001, Title: "A guruh", Type: "supergroup"}
	require.NoError(t, uc.HandleChatMembership(ctx, chat, 555, true))
	require.NoError(t, uc.HandleChatMembership(ctx, chat, 555, false))

	_, err := uc.AuthorizeChat(ctx, mentorA, -1001)
	require.True(t, apperr.IsBadRequest(err))
}

// TelegramIDOf — teskari yo'nalish (bot mentorga o'zi murojaat qiladi).
func TestTelegramIDOf(t *testing.T) {
	uc, _, _ := setup(t)
	ctx := context.Background()

	_, linked, err := uc.TelegramIDOf(ctx, mentorA)
	require.NoError(t, err)
	require.False(t, linked, "bog'lanmagan mentor xato bermasligi kerak")

	link, _ := uc.StartLink(ctx, mentorA)
	_, _ = uc.RedeemCode(ctx, link.Code, 777, "tg")

	id, linked, err := uc.TelegramIDOf(ctx, mentorA)
	require.NoError(t, err)
	require.True(t, linked)
	require.Equal(t, int64(777), id)
}
