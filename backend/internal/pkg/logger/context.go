package logger

import "context"

// contextKey is unexported — boshqa paketlar bilan to'qnashmaydi.
type contextKey string

const (
	traceIDKey contextKey = "trace_id"
	spanIDKey  contextKey = "span_id"
	userIDKey  contextKey = "user_id"
)

// WithTrace stores trace_id and span_id in the context.
// Interceptor har bir request boshida avtomatik chaqiradi.
// Handler ichida qo'lda chaqirishga hojat yo'q.
//
//	ctx = logger.WithTrace(ctx, traceID, spanID)
func WithTrace(ctx context.Context, traceID, spanID string) context.Context {
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	return context.WithValue(ctx, spanIDKey, spanID)
}

// WithUser stores user_id in the context.
// Authenticated endpoint-larda JWT/session dan keyin chaqiring.
//
//	ctx = logger.WithUser(ctx, claims.UserID)
func WithUser(ctx context.Context, userID interface{}) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
