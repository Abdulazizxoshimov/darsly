package recording

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
)

// LiveKit — recording usecase talab qiladigan LiveKit operatsiyalari (DIP).
type LiveKit interface {
	Enabled() bool
	StartRoomRecording(ctx context.Context, roomName, objectKey string, s3 livekit.S3Config) (string, error)
	StopRecording(ctx context.Context, egressID string) error
}

type UseCase interface {
	// EnsureRecording — dars jonli bo'lganda yozib olishni AVTOMATIK boshlaydi.
	//
	// Mahsulot qoidasi: yozib olish **default yoniq** — ustoz uni yoqishni
	// unutmasligi uchun server o'zi boshlaydi. Majburiy EMAS: dars uchun
	// `is_recording_enabled = false` bo'lsa boshlanmaydi (ustozning ataylab
	// qilgan tanlovi hurmat qilinadi).
	//
	// ## QACHON chaqiriladi va NEGA aynan o'shanda (5 daqiqalik tuzoq)
	// Avval bu `room.HostToken` dan — ya'ni ustozga token berilgan zahoti —
	// chaqirilardi. Jonli sinovda muammo ochildi:
	//
	//	egress_aborted  "Start signal not received"  code 412
	//
	// LiveKit'ning room-composite egress'i xonaga kirib **kimdir media
	// chiqarishini kutadi** va 5 daqiqada kutgani kelmasa bekor bo'ladi.
	// Token berilishi bilan ulanish orasida ustozning ruxsat berishi, kamera
	// ochilishi bor; ulanish yiqilsa (ruxsat rad etildi, internet uzildi)
	// egress 5 daqiqa **bo'sh Chrome aylantirib** turadi va yozuv umuman
	// qolmaydi. 4 yadroli serverda bu ikki barobar zarar: yozuv ham yo'q,
	// protsessor ham bekorga band.
	//
	// Endi chaqiruv `track_published` webhook'idan keladi ([EnsureForRoom]),
	// ya'ni xonada **haqiqatan media bor** paytda. Shunda egress darhol
	// "start signal" oladi, kutish oynasi umuman ochilmaydi va hech kim
	// ulanmagan bo'lsa egress ishga ham tushmaydi.
	//
	// IDEMPOTENT: shu dars uchun allaqachon faol yozuv bo'lsa hech narsa
	// qilmaydi va xato ham qaytarmaydi. Bu shart, chunki ustoz dars davomida
	// qayta ulanishi mumkin (internet uzilishi, ilova qayta ishga tushishi) va
	// har `HostToken` da yangi egress boshlansa bitta dars uchun bir nechta
	// parallel yozuv ketardi — bu 4 yadroli serverni yiqitadi.
	//
	// Egalik tekshirilmaydi: chaqiruvchi (room usecase) uni allaqachon
	// tekshirgan; bu metod HTTP orqali ochilmaydi.
	EnsureRecording(ctx context.Context, lessonID string) error

	// EnsureForRoom — LiveKit webhook'i uchun: xona nomi bo'yicha [EnsureRecording].
	//
	// Yozuv AYNAN shu yo'l orqali boshlanadi (`track_published` hodisasida),
	// ya'ni xonada haqiqatan media paydo bo'lganda. Sabab pastda —
	// [EnsureRecording] izohidagi "5 daqiqalik tuzoq".
	EnsureForRoom(ctx context.Context, roomName string) error

	// StopActiveForLesson — dars yakunlanganda faol yozuvni to'xtatadi.
	//
	// LiveKit xona o'chirilganda egress o'zi ham tugaydi, lekin bu yerda
	// OSHKORA to'xtatiladi: shunda DB holati darhol `processing` ga o'tadi va
	// ustoz yozuvlar ro'yxatida "nima bo'lyapti" degan savolsiz qoladi.
	StopActiveForLesson(ctx context.Context, lessonID string) error

	// IsRecording — shu dars hozir yozib olinyaptimi.
	//
	// Ishtirokchiga (o'quvchiga) ko'rsatish uchun: yozuv holati mentor huquqli
	// `ListByLesson` ortida edi, ya'ni o'quvchi o'zi yozilayotganini bila
	// olmasdi. Bu maxfiylik talabi, qulaylik emas. Xato bo'lsa `false` —
	// "yozilmayapti" deb YOLG'ON ko'rsatgandan ko'ra indikatorni umuman
	// ko'rsatmagan yaxshi... aksincha: DB uzilganda ham yozuv davom etayotgan
	// bo'lishi mumkin, shuning uchun `activeFor` ning ehtiyotkor xulqi
	// (xato → "faol" deb hisoblash) shu yerda ham saqlanadi.
	IsRecording(ctx context.Context, lessonID string) bool

	// StartRecording — mentor yozib olishni boshlaydi (LiveKit Egress → MinIO).
	//
	// Avtomatik boshlash joriy etilgandan keyin bu yo'l odatda kerak bo'lmaydi,
	// lekin UI'da qoldi: dars o'rtasida qayta boshlash yoki avtomatik boshlash
	// biror sababga ko'ra ishlamagan holatlar uchun.
	StartRecording(ctx context.Context, mentorID, lessonID string) (*entity.Recording, error)
	// StopRecording — yozib olishni to'xtatadi.
	StopRecording(ctx context.Context, mentorID, recordingID string) error
	// ListByLesson — dars yozuvlari (mentor).
	ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Recording, error)
	// DownloadURL — tayyor yozuv uchun vaqtinchalik MinIO presigned havolasi.
	DownloadURL(ctx context.Context, mentorID, recordingID string) (*entity.RecordingDownload, error)
	// HandleEgress — LiveKit Egress webhook'idan kelgan yakuniy holatni qayta ishlaydi.
	HandleEgress(ctx context.Context, egressID string, completed bool, objectKey string, durationSec int, sizeBytes int64) error
}
