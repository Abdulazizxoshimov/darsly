package chat

import (
	"context"
	"io"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — chat usecase talab qiladigan LiveKit operatsiyalari (DIP).
type LiveKit interface {
	Enabled() bool
	SendData(ctx context.Context, room string, data []byte) error
	// SendDataTo — faqat ko'rsatilgan ishtirokchilarga (shaxsiy xabar).
	SendDataTo(ctx context.Context, room string, data []byte, identities []string) error
}

// Storage — chat fayllari uchun obyekt saqlagich (DIP: `minio.Client` qondiradi).
//
// Faqat kerakli ikki amal: yuklash va vaqtinchalik havola. O'chirish YO'Q —
// moderatsiya xabarni yashiradi, faylni esa qoldiradi (moderatsiya izi bilan
// izchil; darsdan keyingi tozalash alohida retention ishi).
type Storage interface {
	Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error)
	PresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error)
}

// FileUpload — yuklanayotgan fayl (HTTP qatlamidan keladi).
//
// `Reader` OQIM: 20 MB'lik fayl xotiraga to'liq olinmaydi. Nomi va hajmi
// klientdan keladi va IKKALASI ham tekshiriladi (`validateFile`).
type FileUpload struct {
	Name   string
	Size   int64
	Reader io.Reader
}

// UseCase — dars ichidagi chat.
//
// Ikki kirish yo'li bor va bu ATAYLAB:
//   - `Send`/`History`/`Upload` — HOST yo'li (JWT + egalik). Ustoz dars tugagach ham
//     tarixni o'qiy oladi, ya'ni xonaga ulangan bo'lishi shart emas.
//   - `SendFromRoom`/`HistoryForRoom`/`UploadFromRoom` — XONA yo'li (LiveKit
//     room-token). Guest'da JWT yo'q; room-token esa faqat dars davomida
//     yaroqli — to'g'ri cheklov.
type UseCase interface {
	// Send — host xabar yuboradi: saqlaydi va tarqatadi. to bo'sh bo'lsa — hammaga.
	Send(ctx context.Context, mentorID, lessonID, body, to string) (*entity.ChatMessage, error)
	// History — dars chat tarixi (host), eng yangidan eskiga. before!=nil → kursor
	// (undan eski xabarlar). Sahifa hajmi limit (default/max 50).
	History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
	// Upload — host fayl ulashadi (chat xabari sifatida saqlanadi + tarqatiladi).
	Upload(ctx context.Context, mentorID, lessonID string, f FileUpload, body, to string) (*entity.ChatMessage, error)

	// Transcript — dars chatini yuklab olinadigan fayl qilib beradi (ish №21).
	//
	// Zoom dars oxirida chat faylini beradi; bizda tarix DB'da bor edi, lekin
	// undan chiqish yo'li yo'q edi — ya'ni ustoz o'z darsining yozishmasini
	// ilovadan tashqarida saqlay olmasdi.
	//
	// format: [FormatTXT] yoki [FormatHTML]; boshqasi → 400.
	// Ko'rinuvchanlik `History` bilan bir xil: ommaviy xabarlar + ustozning
	// O'Z shaxsiy yozishmalari (o'quvchilarning bir-biriga yozgani emas).
	Transcript(ctx context.Context, mentorID, lessonID, format string) (*entity.ChatTranscript, error)

	// SendFromRoom — xonadagi ISHTIROKCHI xabar yuboradi (room-token bilan
	// autentifikatsiya qilingan identity). Tezlik cheklovi shu yerda.
	SendFromRoom(ctx context.Context, lessonID, identity, name, body, to string) (*entity.ChatMessage, error)
	// HistoryForRoom — xonadagi ishtirokchi uchun tarix: ommaviy xabarlar +
	// faqat O'ZI ishtirok etgan shaxsiy yozishmalar.
	HistoryForRoom(ctx context.Context, lessonID, identity string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
	// UploadFromRoom — xonadagi ishtirokchi fayl ulashadi (hajm/tur/tezlik cheklovi bilan).
	UploadFromRoom(ctx context.Context, lessonID, identity, name string, f FileUpload, body, to string) (*entity.ChatMessage, error)

	// Delete — MODERATSIYA (№6): dars egasi xabarni tarixdan olib tashlaydi va
	// xonadagi klientlarga `chat_deleted` hodisasini yuboradi.
	//
	// Faqat dars egasi (mentor) — o'quvchi o'z xabarini ham o'chira olmaydi:
	// "yozib qo'yib, izini o'chirish" moderatsiyaga qarshi ishlardi.
	Delete(ctx context.Context, mentorID, lessonID, messageID string) error
}
