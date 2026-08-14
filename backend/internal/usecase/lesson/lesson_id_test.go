package lesson_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/lesson"
)

// Yaroqsiz UUID Postgres'ga yetmasligi kerak: `GET /lessons/abc` avval
// `22P02 invalid input syntax for type uuid` → 500 INTERNAL_ERROR qaytarardi va
// har bir foydalanuvchi Sentry'ni soxta 500'lar bilan to'ldirib real nosozlikni
// ko'rinmas qila olardi. Kutilgan: 404 (mavjud bo'lmagan UUID bilan bir xil javob).
func TestLesson_InvalidUUID_NotFound(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := lesson.New(repo, hasher.New(4), testutil.NewLogger())
	ctx := context.Background()

	for _, id := range []string{"abc", "", "not-a-uuid", "' OR 1=1 --"} {
		t.Run("get/"+id, func(t *testing.T) {
			_, err := uc.GetByID(ctx, "mentor1", id)
			require.True(t, apperr.IsNotFound(err), "GetByID(%q) → 404 kutilgan, oldi: %v", id, err)
		})
		t.Run("update/"+id, func(t *testing.T) {
			title := "X"
			_, err := uc.Update(ctx, "mentor1", id, &entity.UpdateLessonReq{Title: &title})
			require.True(t, apperr.IsNotFound(err), "Update(%q) → 404 kutilgan, oldi: %v", id, err)
		})
		t.Run("delete/"+id, func(t *testing.T) {
			require.True(t, apperr.IsNotFound(uc.Delete(ctx, "mentor1", entity.RoleMentor, id)), "Delete(%q) → 404 kutilgan", id)
		})
	}
}
