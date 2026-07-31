package chat_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/usecase/chat"
)

// Haqiqiy sarlavhalar: `http.DetectContentType` mazmunni AYNAN shu baytlardan
// taniydi, ya'ni testlar sniff mantig'ini ham qamrab oladi.
var (
	pngBytes = append([]byte("\x89PNG\r\n\x1a\n"), bytes.Repeat([]byte{0}, 64)...)
	pdfBytes = append([]byte("%PDF-1.7\n"), bytes.Repeat([]byte{0}, 64)...)
	zipBytes = append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0}, 64)...)
)

func upload(name string, data []byte) chat.FileUpload {
	return chat.FileUpload{Name: name, Size: int64(len(data)), Reader: bytes.NewReader(data)}
}

func TestUpload_MuvaffaqiyatliVaTarixda(t *testing.T) {
	uc, lk, _ := setupFull(t)
	ctx := context.Background()

	m, err := uc.Upload(ctx, "mentor1", testLessonID, upload("darslik.pdf", pdfBytes), "Bugungi slayd", "")
	require.NoError(t, err)
	require.NotNil(t, m.File)
	require.Equal(t, "darslik.pdf", m.File.Name)
	require.Equal(t, "application/pdf", m.File.Mime)
	require.Equal(t, int64(len(pdfBytes)), m.File.Size)
	require.Equal(t, "Bugungi slayd", m.Body)
	// Presigned havola javobda bo'lishi kerak, kalit esa YO'Q (bucket sxemasi sir).
	require.NotEmpty(t, m.File.URL)
	require.Positive(t, m.File.ExpiresInS)
	require.GreaterOrEqual(t, lk.Calls["SendData"], 1, "fayl xabari ham xonaga tarqalishi kerak")

	// Tarixda ham fayl bilan va yangi (amaldagi) havola bilan qaytadi.
	hist, err := uc.History(ctx, "mentor1", testLessonID, nil, 0)
	require.NoError(t, err)
	require.Len(t, hist, 1)
	require.NotNil(t, hist[0].File)
	require.Equal(t, "darslik.pdf", hist[0].File.Name)
	require.NotEmpty(t, hist[0].File.URL, "tarixdagi fayl uchun ham presigned havola berilishi kerak")
}

// O'QUVCHI ham fayl yubora oladi (mahsulot qarori), lekin xuddi shu cheklovlar bilan.
func TestUploadFromRoom_OquvchiHamYuboradi(t *testing.T) {
	uc, _, _ := setupFull(t)
	m, err := uc.UploadFromRoom(context.Background(), testLessonID, "guest_a", "Ali",
		upload("uy_ishi.png", pngBytes), "", "")
	require.NoError(t, err)
	require.Equal(t, "guest_a", m.SenderIdentity)
	require.Equal(t, "image/png", m.File.Mime)
}

func TestUpload_HajmOshsa400(t *testing.T) {
	uc, _, _ := setupFull(t)
	big := chat.FileUpload{
		Name:   "katta.pdf",
		Size:   chat.MaxChatFileBytes + 1,
		Reader: bytes.NewReader(pdfBytes),
	}
	_, err := uc.Upload(context.Background(), "mentor1", testLessonID, big, "", "")
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)
}

func TestUpload_BoshFayl400(t *testing.T) {
	uc, _, _ := setupFull(t)
	_, err := uc.Upload(context.Background(), "mentor1", testLessonID,
		chat.FileUpload{Name: "bosh.pdf", Size: 0, Reader: bytes.NewReader(nil)}, "", "")
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)
}

func TestUpload_RuxsatsizTur400(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()
	for _, name := range []string{"virus.exe", "skript.sh", "video.mp4", "arxiv.zip", "nomsiz"} {
		_, err := uc.Upload(ctx, "mentor1", testLessonID, upload(name, pdfBytes), "", "")
		require.True(t, apperr.IsBadRequest(err), "%q rad etilishi kerak, olindi: %v", name, err)
	}
}

// Kengaytmani almashtirib xohlagan narsani yuklab bo'lmasin: mazmun ham
// tekshiriladi (`http.DetectContentType`).
func TestUpload_MazmunKengaytmagaMosKelmasa400(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()

	// ELF/skript baytlari `.png` niqobida.
	_, err := uc.Upload(ctx, "mentor1", testLessonID,
		upload("rasm.png", []byte("#!/bin/sh\nrm -rf /\n")), "", "")
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)

	// HTML `.pdf` niqobida (MinIO'dan `text/html` bo'lib ochilishi xavfi).
	_, err = uc.Upload(ctx, "mentor1", testLessonID,
		upload("hujjat.pdf", []byte("<html><script>alert(1)</script></html>")), "", "")
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)
}

// OOXML (docx) — ichida ZIP, sniff "application/zip" beradi va bu KUTILGAN.
func TestUpload_OfisHujjatiOtadi(t *testing.T) {
	uc, _, _ := setupFull(t)
	m, err := uc.Upload(context.Background(), "mentor1", testLessonID,
		upload("konspekt.docx", zipBytes), "", "")
	require.NoError(t, err)
	require.Equal(t,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		m.File.Mime)
}

// Nomdagi yo'l tashlanadi: klient "../../etc/passwd" yubora oladi.
func TestUpload_NomdagiYolTashlanadi(t *testing.T) {
	uc, _, _ := setupFull(t)
	m, err := uc.Upload(context.Background(), "mentor1", testLessonID,
		upload("../../etc/rasm.png", pngBytes), "", "")
	require.NoError(t, err)
	require.Equal(t, "rasm.png", m.File.Name)
	require.NotContains(t, m.File.Name, "..")
}

func TestUpload_TezlikChegarasi(t *testing.T) {
	uc, _, _ := setupFull(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := uc.UploadFromRoom(ctx, testLessonID, "spammer", "S", upload("a.png", pngBytes), "", "")
		require.NoError(t, err, "birinchi 5 ta yuklash o'tishi kerak")
	}
	_, err := uc.UploadFromRoom(ctx, testLessonID, "spammer", "S", upload("a.png", pngBytes), "", "")
	require.Error(t, err, "6-chi yuklash rad etilishi kerak")
	require.Equal(t, 429, apperr.As(err).HTTPStatus)

	// Boshqa ishtirokchi cheklanmaydi.
	_, err = uc.UploadFromRoom(ctx, testLessonID, "boshqa", "B", upload("a.png", pngBytes), "", "")
	require.NoError(t, err)
}

// Chiqarilgan ishtirokchi fayl ham yubora olmasligi kerak (chat/qo'l/ovoz bilan izchil).
func TestUploadFromRoom_YaroqsizDars(t *testing.T) {
	uc, _, _ := setupFull(t)
	_, err := uc.UploadFromRoom(context.Background(), "abc", "guest_a", "Ali",
		upload("a.png", pngBytes), "", "")
	require.Error(t, err)
}

// Fayl ham shaxsiy (DM) bo'la oladi va faqat ikki tomonga yetkaziladi.
func TestUpload_ShaxsiyFayl(t *testing.T) {
	uc, lk, _ := setupFull(t)
	m, err := uc.UploadFromRoom(context.Background(), testLessonID, "guest_a", "Ali",
		upload("javob.png", pngBytes), "", "mentor1")
	require.NoError(t, err)
	require.NotNil(t, m.ToIdentity)
	require.Equal(t, 1, lk.Calls["SendDataTo"])
	require.Equal(t, 0, lk.Calls["SendData"])

	// Uchinchi shaxs tarixda ham ko'rmaydi.
	hist, err := uc.HistoryForRoom(context.Background(), testLessonID, "guest_b", nil, 0)
	require.NoError(t, err)
	require.Empty(t, hist)
}

// Katta-kichik harf farqi kengaytmani chetlab o'tmasin.
func TestUpload_KengaytmaKattaHarfBilan(t *testing.T) {
	uc, _, _ := setupFull(t)
	m, err := uc.Upload(context.Background(), "mentor1", testLessonID,
		upload("RASM.PNG", pngBytes), "", "")
	require.NoError(t, err)
	require.Equal(t, "image/png", m.File.Mime)
	require.True(t, strings.HasSuffix(strings.ToLower(m.File.Name), ".png"))
}
