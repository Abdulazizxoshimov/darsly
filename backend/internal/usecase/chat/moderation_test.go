package chat_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// №6 — moderatsiya. Dars egasi xabarni o'chira oladi.
func TestDelete_EgasiOchiraOladi(t *testing.T) {
	uc, _, crepo := setupFull(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "haqorat", "")
	require.NoError(t, err)

	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, m.ID))
	require.Equal(t, "mentor1", crepo.DeletedBy(m.ID), "o'chirgan kishi moderatsiya izida qolishi kerak")
}

func TestDelete_BegonaMentor403(t *testing.T) {
	uc, _, crepo := setupFull(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "xabar", "")
	require.NoError(t, err)

	err = uc.Delete(ctx, "intruder", testLessonID, m.ID)
	require.True(t, apperr.IsForbidden(err), "begona mentor o'chira olmasligi kerak, olindi: %v", err)
	require.Empty(t, crepo.DeletedBy(m.ID), "xabar o'chirilmagan bo'lishi kerak")
}

// O'chirilgan xabar TARIXDA KO'RINMAYDI — na o'quvchida, na ustozda.
// Qaror izohi: `migrations/000020_chat_moderation.up.sql`.
func TestDelete_TarixdanYoqoladi(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()

	bad, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "haqorat", "")
	require.NoError(t, err)
	_, err = uc.SendFromRoom(ctx, testLessonID, "guest_b", "Vali", "oddiy savol", "")
	require.NoError(t, err)

	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, bad.ID))

	// O'quvchi ko'rmaydi.
	hist, err := uc.HistoryForRoom(ctx, testLessonID, "guest_c", nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, "oddiy savol", hist[0].Body)

	// Ustoz ham ko'rmaydi (qabrtosh qoldirilmaydi — buzg'unchiga e'tibor bermaslik).
	hist, err = uc.History(ctx, "mentor1", testLessonID, nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.Equal(t, "oddiy savol", hist[0].Body)
}

// Jonli xonadagi klientlar `chat_deleted` hodisasini oladi — pufak darhol yo'qoladi.
func TestDelete_XonagaHodisaYuboriladi(t *testing.T) {
	uc, lk, _ := setupFull(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "haqorat", "")
	require.NoError(t, err)
	before := len(lk.Sent)

	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, m.ID))
	require.Len(t, lk.Sent, before+1, "o'chirish hodisasi tarqatilishi kerak")

	var ev map[string]any
	require.NoError(t, json.Unmarshal(lk.Sent[len(lk.Sent)-1], &ev))
	require.Equal(t, "chat_deleted", ev["kind"])
	require.Equal(t, m.ID, ev["id"])
	// Mazmun YUBORILMAYDI: o'chirishning butun maqsadi uni tarqatmaslik edi.
	require.NotContains(t, ev, "body")
}

// Shaxsiy xabarning o'chirilgani butun xonaga oshkor bo'lmasligi kerak —
// yozishmaning FAKTI ham maxfiy ma'lumot.
func TestDelete_ShaxsiyXabarHodisasiManzilli(t *testing.T) {
	uc, lk, _ := setupFull(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "shaxsiy haqorat", "mentor1")
	require.NoError(t, err)
	beforeAll, beforeTo := lk.Calls["SendData"], lk.Calls["SendDataTo"]

	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, m.ID))
	require.Equal(t, beforeAll, lk.Calls["SendData"], "xonaga tarqatilmasligi kerak")
	require.Equal(t, beforeTo+1, lk.Calls["SendDataTo"])
	require.ElementsMatch(t, []string{"mentor1", "guest_a"}, lk.SentTo[len(lk.SentTo)-1])
}

// Takroriy o'chirish (ikki marta bosish / ikki qurilma) ikkinchi marta hodisa
// tarqatmasligi kerak — atomiklik `WHERE deleted_at IS NULL` bilan.
func TestDelete_Takroriy404(t *testing.T) {
	uc, lk, _ := setupFull(t)
	ctx := context.Background()

	m, err := uc.SendFromRoom(ctx, testLessonID, "guest_a", "Ali", "x", "")
	require.NoError(t, err)
	require.NoError(t, uc.Delete(ctx, "mentor1", testLessonID, m.ID))
	after := len(lk.Sent)

	err = uc.Delete(ctx, "mentor1", testLessonID, m.ID)
	require.True(t, apperr.IsNotFound(err), "kutilgan 404, olindi: %v", err)
	require.Len(t, lk.Sent, after, "ikkinchi urinishda hodisa tarqatilmasligi kerak")
}

func TestDelete_YaroqsizID(t *testing.T) {
	uc, _, _ := setupFull(t)
	// Yaroqsiz UUID Postgres'ga yetib 500 generatoriga aylanmasin.
	err := uc.Delete(context.Background(), "mentor1", testLessonID, "abc")
	require.True(t, apperr.IsNotFound(err), "kutilgan 404, olindi: %v", err)
}
