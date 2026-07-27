package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/zoom/darsly/internal/pkg/logger"
)

const HeaderRequestID = "X-Request-ID"

// RequestID injects a unique request ID into the request context and response header.
// Muhim: ID request Context'iga ham qo'yiladi — shunda usecase ichidagi barcha
// log.Info(ctx, ...) chaqiruvlari trace_id bilan yoziladi (bir so'rov bo'ylab korrelyatsiya).
//
// OTEL yoqilgan bo'lsa (otelgin span yaratgan) log trace_id/span_id sifatida REAL
// OTEL identifikatorlari ishlatiladi — shunda Tempo/Jaeger trace'idan Loki log'iga
// trace_id orqali o'tish mumkin. OTEL o'chiq bo'lsa request_id trace_id sifatida qoladi.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Set(HeaderRequestID, id)
		c.Header(HeaderRequestID, id)

		traceID, spanID := id, ""
		if sc := oteltrace.SpanContextFromContext(c.Request.Context()); sc.HasTraceID() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}
		c.Request = c.Request.WithContext(logger.WithTrace(c.Request.Context(), traceID, spanID))
		c.Next()
	}
}
