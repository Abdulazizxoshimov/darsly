package shared_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/usecase/shared"
)

const lessonID = "11111111-1111-4111-8111-111111111111"

func TestRoomNameRoundTrip(t *testing.T) {
	// Prefiks ikki joyda mustaqil yozilgan bo'lsa, biri o'zgarganda ikkinchisi
	// jimgina mos kelmay qolardi va webhook orqali yozuv boshlash to'xtardi.
	got, ok := shared.LessonIDFromRoom(shared.RoomName(lessonID))
	require.True(t, ok)
	require.Equal(t, lessonID, got)
}

func TestLessonIDFromRoom_Rejects(t *testing.T) {
	for _, bad := range []string{"", "boshqa-xona", "lesson_", "lessonX" + lessonID, "LESSON_" + lessonID} {
		_, ok := shared.LessonIDFromRoom(bad)
		require.False(t, ok, "begona nom qabul qilinmasin: %q", bad)
	}
}
