package user_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/user"
)

func newUC(t *testing.T) user.UseCase {
	t.Helper()
	repo := testutil.NewFakeUserRepo()
	return user.New(repo, hasher.New(4), testutil.NewFakeTokenMaker(), testutil.NewLogger())
}

// Yaroqsiz UUID DB'ga yetmasligi kerak: Postgres `22P02` xatosi 500 INTERNAL_ERROR
// bo'lib qaytardi va Sentry'ni soxta xatolar bilan to'ldirardi (`GET /users/abc`).
// Endi javob 404 — mavjud bo'lmagan (lekin shakli to'g'ri) UUID bilan AYNI xil.
func TestUser_InvalidUUID_Returns404NotInternal(t *testing.T) {
	uc := newUC(t)
	ctx := context.Background()

	bad := []string{"abc", "not-a-uuid", "123", "'; DROP TABLE users;--", "00000000-0000-0000-0000-00000000000"}

	assert404 := func(t *testing.T, err error, what string) {
		t.Helper()
		require.Error(t, err, what)
		var ae *apperr.AppError
		require.ErrorAs(t, err, &ae, "%s: apperr bo'lishi kerak (aks holda handler 500 qaytaradi)", what)
		require.Equal(t, http.StatusNotFound, ae.HTTPStatus, "%s: 404 bo'lishi kerak, 500 emas", what)
	}

	for _, id := range bad {
		t.Run(id, func(t *testing.T) {
			_, err := uc.GetByID(ctx, id)
			assert404(t, err, "GetByID")

			_, err = uc.Update(ctx, id, &entity.UpdateUserReq{})
			assert404(t, err, "Update")

			assert404(t, uc.ChangePassword(ctx, id, &entity.ChangePasswordReq{}), "ChangePassword")
			assert404(t, uc.Deactivate(ctx, id), "Deactivate")
			assert404(t, uc.Activate(ctx, id), "Activate")
			assert404(t, uc.Delete(ctx, id), "Delete")
			assert404(t, uc.ResetPassword(ctx, "caller", "admin", id, "parol12345"), "ResetPassword")
		})
	}
}

// Shakli to'g'ri, lekin mavjud bo'lmagan UUID ham xuddi shu 404 — hujumchi uchun
// "ID shakli to'g'rimi" degan oracle qolmasligi kerak.
func TestUser_ValidButMissingUUID_SameAsInvalid(t *testing.T) {
	uc := newUC(t)
	_, err := uc.GetByID(context.Background(), "11111111-2222-3333-4444-555555555555")
	require.Error(t, err)
	var ae *apperr.AppError
	require.ErrorAs(t, err, &ae)
	require.Equal(t, http.StatusNotFound, ae.HTTPStatus)
}
