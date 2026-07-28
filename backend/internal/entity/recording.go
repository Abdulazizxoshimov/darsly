package entity

import "time"

// Recording — dars yozuvi (LiveKit Egress → MinIO).
type Recording struct {
	ID          string     `json:"id"`
	LessonID    string     `json:"lesson_id"`
	EgressID    string     `json:"egress_id"`
	ObjectKey   string     `json:"-"` // MinIO ichidagi yo'l (ichki)
	Status      string     `json:"status"`
	DurationSec int        `json:"duration_sec"`
	SizeBytes   int64      `json:"size_bytes"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Yozuv statuslari
const (
	RecordingStatusRecording  = "recording"  // egress faol
	RecordingStatusProcessing = "processing" // to'xtatildi, yuklanmoqda
	RecordingStatusReady      = "ready"      // MinIO'da tayyor
	RecordingStatusFailed     = "failed"
)

// Qayta kodlash (CRF) holatlari — `recordings.transcode_status`.
//
// Nega alohida holat: yozuv `ready` bo'lishi bilan ustoz uni yuklab ola oladi,
// qayta kodlash esa fonda va keyinroq bo'ladi. Ikkalasini bitta ustunga
// tiqish "tayyor, lekin hali tayyor emas" degan chalkash holat yasardi.
const (
	TranscodePending = "pending"
	TranscodeRunning = "running"
	TranscodeDone    = "done"
	TranscodeFailed  = "failed"
	TranscodeSkipped = "skipped"
)

// RecordingDownload — vaqtinchalik yuklab olish havolasi.
type RecordingDownload struct {
	URL         string `json:"url"`
	ExpiresInS  int    `json:"expires_in_s"`
	DurationSec int    `json:"duration_sec"`
	SizeBytes   int64  `json:"size_bytes"`
}
