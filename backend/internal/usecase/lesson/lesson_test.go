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

func newLessonUC(repo *testutil.FakeLessonRepo) lesson.UseCase {
	return lesson.New(repo, hasher.New(4), testutil.NewLogger())
}

func ptr[T any](v T) *T { return &v }

func TestCreate_DefaultsAndPasscode(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := newLessonUC(repo)

	l, err := uc.Create(context.Background(), "mentor1", &entity.CreateLessonReq{
		Title:    "Matematika",
		Passcode: ptr("1234"),
	})
	require.NoError(t, err)
	require.Equal(t, "mentor1", l.MentorID)
	require.Equal(t, entity.LessonStatusScheduled, l.Status)
	require.Equal(t, 60, l.DurationMin, "default davomiylik 60")
	require.NotEmpty(t, l.JoinSlug)
	require.True(t, l.HasPasscode)
	require.NotNil(t, l.PasscodeHash)
	require.NotEqual(t, "1234", *l.PasscodeHash, "passcode hash qilingan bo'lishi kerak")
}

func TestCreate_SlugsAreUnique(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := newLessonUC(repo)

	seen := map[string]bool{}
	for range 30 {
		l, err := uc.Create(context.Background(), "m", &entity.CreateLessonReq{Title: "X"})
		require.NoError(t, err)
		require.False(t, seen[l.JoinSlug], "join_slug noyob bo'lishi kerak: %s", l.JoinSlug)
		seen[l.JoinSlug] = true
	}
}

func TestGetByID_Ownership(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := newLessonUC(repo)
	l, _ := uc.Create(context.Background(), "owner", &entity.CreateLessonReq{Title: "X"})

	got, err := uc.GetByID(context.Background(), "owner", l.ID)
	require.NoError(t, err)
	require.Equal(t, l.ID, got.ID)

	_, err = uc.GetByID(context.Background(), "intruder", l.ID)
	require.Error(t, err)
	require.True(t, apperr.IsForbidden(err), "boshqa mentor GET → 403 (IDOR bloklandi)")
}

func TestUpdate_OwnershipAndPasscodeRemoval(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := newLessonUC(repo)
	l, _ := uc.Create(context.Background(), "owner", &entity.CreateLessonReq{Title: "X", Passcode: ptr("1234")})

	_, err := uc.Update(context.Background(), "intruder", l.ID, &entity.UpdateLessonReq{Title: ptr("Hacked")})
	require.True(t, apperr.IsForbidden(err), "boshqa mentor Update → 403")

	upd, err := uc.Update(context.Background(), "owner", l.ID, &entity.UpdateLessonReq{RemovePasscode: true})
	require.NoError(t, err)
	require.False(t, upd.HasPasscode, "parol olib tashlanishi kerak")
	require.Nil(t, upd.PasscodeHash)
}

func TestDelete_Ownership(t *testing.T) {
	repo := testutil.NewFakeLessonRepo()
	uc := newLessonUC(repo)
	l, _ := uc.Create(context.Background(), "owner", &entity.CreateLessonReq{Title: "X"})

	require.True(t, apperr.IsForbidden(uc.Delete(context.Background(), "intruder", l.ID)))
	require.NoError(t, uc.Delete(context.Background(), "owner", l.ID))
	_, err := uc.GetByID(context.Background(), "owner", l.ID)
	require.Error(t, err, "o'chirilgach topilmasligi kerak")
}

func TestCreate_RecordingDefaultsOn(t *testing.T) {
	// ⭐ "Yozib olish default yoniq" — bu SERVER qoidasi, klient odobi emas.
	//
	// Avval `is_recording_enabled` oddiy `bool` edi va berilmasa Go'ning nol
	// qiymati `false` bo'lardi: mobil va web `true` yuborgani uchun ular
	// to'g'ri ishlardi, API'ga to'g'ridan-to'g'ri murojaat qilgan har qanday
	// narsa esa jimgina yozuvsiz dars yaratardi.
	uc := newLessonUC(testutil.NewFakeLessonRepo())

	l, err := uc.Create(context.Background(), "mentor1", &entity.CreateLessonReq{Title: "Bayroqsiz"})
	require.NoError(t, err)
	require.True(t, l.IsRecordingEnabled, "berilmagan bayroq → YOQILGAN")
}

func TestCreate_RecordingCanBeDisabledExplicitly(t *testing.T) {
	// Default yoniq, lekin MAJBURIY EMAS: oshkora `false` hurmat qilinadi.
	uc := newLessonUC(testutil.NewFakeLessonRepo())

	l, err := uc.Create(context.Background(), "mentor1", &entity.CreateLessonReq{
		Title: "Yozuvsiz", IsRecordingEnabled: ptr(false),
	})
	require.NoError(t, err)
	require.False(t, l.IsRecordingEnabled)
}
