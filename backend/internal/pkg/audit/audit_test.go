package audit

import (
	"context"
	"testing"

	"github.com/zoom/darsly/internal/pkg/logger"
)

type levelLog struct{ level string }

func (l *levelLog) Debug(context.Context, string, ...logger.Field) { l.level = "debug" }
func (l *levelLog) Info(context.Context, string, ...logger.Field)  { l.level = "info" }
func (l *levelLog) Warn(context.Context, string, ...logger.Field)  { l.level = "warn" }
func (l *levelLog) Error(context.Context, string, ...logger.Field) { l.level = "error" }
func (l *levelLog) Fatal(context.Context, string, ...logger.Field) { l.level = "fatal" }

// Loki core faqat Warn+ yuboradi — audit Info'da qolsa markaziy izga yetmaydi.
func TestRecord_UsesWarnSoLokiReceivesIt(t *testing.T) {
	l := &levelLog{}
	Record(context.Background(), l, "waitingroom.admit", "u1")
	if l.level != "warn" {
		t.Fatalf("audit darajasi warn bo'lishi kerak, got %q", l.level)
	}
}
