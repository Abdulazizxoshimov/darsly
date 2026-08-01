package telegram

import (
	"context"
	"fmt"
	"io"
)

// nopClient — `TELEGRAM_BOT_TOKEN` berilmaganda qaytariladigan klient.
//
// Naqsh `minio.NewNop()` va Sentry bilan bir xil: integratsiya sozlanmagan
// bo'lsa u JIMGINA o'chiq bo'ladi va tizimning qolgan qismi (yozib olish,
// retention ogohlantirishlari) hech qanday o'zgarishsiz ishlayveradi.
//
// Metodlar `nil` emas, XATO qaytaradi: agar biror joyda `Enabled()` tekshiruvi
// unutilsa, bu jimgina "muvaffaqiyat" bo'lib ketmasin — log'da aniq sabab
// ko'rinsin.
type nopClient struct{}

var errDisabled = fmt.Errorf("telegram: sozlanmagan (TELEGRAM_BOT_TOKEN bo'sh)")

func NewNop() Client { return nopClient{} }

func (nopClient) Enabled() bool         { return false }
func (nopClient) Local() bool           { return false }
func (nopClient) ArchiveChatID() int64  { return 0 }
func (nopClient) MaxUploadBytes() int64 { return 0 }
func (nopClient) BotUsername() string   { return "" }

func (nopClient) Init(context.Context) error { return errDisabled }

func (nopClient) SendMessage(context.Context, int64, string, *InlineKeyboard) (*Message, error) {
	return nil, errDisabled
}
func (nopClient) EditMessageText(context.Context, int64, int64, string, *InlineKeyboard) error {
	return errDisabled
}
func (nopClient) AnswerCallbackQuery(context.Context, string, string) error { return errDisabled }
func (nopClient) SendVideo(context.Context, int64, string, string, int) (*Message, error) {
	return nil, errDisabled
}
func (nopClient) SendVideoByFileID(context.Context, int64, string, string) (*Message, error) {
	return nil, errDisabled
}
func (nopClient) SendDocument(context.Context, int64, string, io.Reader, int64, string) (*Message, error) {
	return nil, errDisabled
}
func (nopClient) GetFile(context.Context, string) (*File, error)    { return nil, errDisabled }
func (nopClient) DownloadFile(context.Context, *File, string) error { return errDisabled }
func (nopClient) GetUpdates(context.Context, int64, int) ([]Update, error) {
	return nil, errDisabled
}
