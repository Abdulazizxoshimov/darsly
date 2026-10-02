package logger

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/zoom/darsly/internal/pkg/loki"
)

// ─── Levels ───────────────────────────────────────────────────────────────────

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	LevelPanic = "panic"
	LevelFatal = "fatal"
)

// ─── Field helpers ────────────────────────────────────────────────────────────

type Field = zapcore.Field

var (
	Int    = zap.Int
	Int64  = zap.Int64
	String = zap.String
	Error  = zap.Error
	Bool   = zap.Bool
	Any    = zap.Any
)

// SafeEmail masks email before logging: john@example.com → j***@example.com
func SafeEmail(key, email string) Field {
	return zap.String(key, MaskEmail(email))
}

// SafeString applies full PII masking to any free-form string.
func SafeString(key, value string) Field {
	return zap.String(key, MaskPII(value))
}

// ─── Logger interface ─────────────────────────────────────────────────────────

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)
}

// ─── Options ──────────────────────────────────────────────────────────────────

type Option func(*loggerImpl)

// WithLoki attaches a Loki core that ships warn+ logs to the given Loki URL.
// user and password are for Basic Auth (Grafana Cloud); leave empty for self-hosted Loki.
func WithLoki(lokiURL, user, password string, labels map[string]string) Option {
	return func(l *loggerImpl) {
		if lokiURL == "" {
			return
		}
		lokiCore := loki.New(lokiURL, user, password, labels, zapcore.WarnLevel)
		l.zap = l.zap.WithOptions(zap.WrapCore(func(existing zapcore.Core) zapcore.Core {
			return zapcore.NewTee(existing, lokiCore)
		}))
		l.lokiCore = lokiCore
	}
}

// ─── Implementation ───────────────────────────────────────────────────────────

type loggerImpl struct {
	zap      *zap.Logger
	lokiCore *loki.Core
}

// New creates a structured JSON logger that writes to stdout only.
// Pass WithLoki(...) to additionally ship logs to Grafana Loki.
func New(level, namespace, serviceName string, opts ...Option) Logger {
	if level == "" {
		level = LevelInfo
	}
	l := &loggerImpl{
		zap: newZapLogger(level),
	}
	l.zap = l.zap.Named(namespace)
	if serviceName != "" {
		l.zap = l.zap.With(zap.String("service_name", serviceName))
	}
	zap.RedirectStdLog(l.zap)

	for _, opt := range opts {
		opt(l)
	}
	return l
}

func (l *loggerImpl) Debug(ctx context.Context, msg string, fields ...Field) {
	l.zap.Debug(msg, l.fromCtx(ctx, fields)...)
}

func (l *loggerImpl) Info(ctx context.Context, msg string, fields ...Field) {
	l.zap.Info(msg, l.fromCtx(ctx, fields)...)
}

func (l *loggerImpl) Warn(ctx context.Context, msg string, fields ...Field) {
	l.zap.Warn(msg, l.fromCtx(ctx, fields)...)
}

func (l *loggerImpl) Error(ctx context.Context, msg string, fields ...Field) {
	l.zap.Error(msg, l.fromCtx(ctx, fields)...)
}

func (l *loggerImpl) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.zap.Fatal(msg, l.fromCtx(ctx, fields)...)
}

// fromCtx extracts trace_id, span_id, user_id from context and prepends them.
func (l *loggerImpl) fromCtx(ctx context.Context, extra []Field) []Field {
	fields := make([]Field, 0, 3+len(extra))
	if v, _ := ctx.Value(traceIDKey).(string); v != "" {
		fields = append(fields, zap.String("trace_id", v))
	}
	if v, _ := ctx.Value(spanIDKey).(string); v != "" {
		fields = append(fields, zap.String("span_id", v))
	}
	if v, _ := ctx.Value(userIDKey).(string); v != "" {
		fields = append(fields, zap.String("user_id", v))
	}
	return append(fields, extra...)
}

// ─── Utility functions ────────────────────────────────────────────────────────

// Cleanup flushes buffered log entries (stdout + Loki). Call on shutdown.
func Cleanup(l Logger) error {
	if v, ok := l.(*loggerImpl); ok {
		if v.lokiCore != nil {
			v.lokiCore.Stop()
		}
		return v.zap.Sync()
	}
	return nil
}

// ─── Zap internals ────────────────────────────────────────────────────────────

func newZapLogger(level string) *zap.Logger {
	atomicLevel := zap.NewAtomicLevelAt(parseLevel(level))

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.MessageKey = "message"
	encoderCfg.LevelKey = "level"
	encoderCfg.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339Nano))
	}

	// Production yoki stdout TTY bo'lmasa (docker/systemd/pipe) — mashina o'qiydigan JSON,
	// ANSI rang kodlarisiz (aks holda journald/Loki'da "\x1b[31m" axlati qoladi).
	// Faqat lokal terminalda rangli konsol.
	var enc zapcore.Encoder
	if useJSONEncoder() {
		encoderCfg.EncodeLevel = zapcore.LowercaseLevelEncoder
		enc = zapcore.NewJSONEncoder(encoderCfg)
	} else {
		encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		enc = zapcore.NewConsoleEncoder(encoderCfg)
	}

	consoleCore := zapcore.NewCore(
		enc,
		zapcore.AddSync(os.Stdout),
		atomicLevel,
	)

	return zap.New(consoleCore, zap.AddCaller(), zap.AddCallerSkip(1))
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	case LevelPanic:
		return zapcore.PanicLevel
	case LevelFatal:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// useJSONEncoder — JSON format tanlanadimi: APP_ENV=production yoki stdout terminal emas.
func useJSONEncoder() bool {
	if os.Getenv("APP_ENV") == "production" {
		return true
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice == 0
}
