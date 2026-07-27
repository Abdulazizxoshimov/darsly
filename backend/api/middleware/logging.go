package middleware

import (
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// sensitiveQueryParams — loglarga tushmasligi kerak bo'lgan query paramlar.
// WS jonli access JWT'ni ?token= orqali, guest admit-capability'ni ?request_id=
// orqali oladi — bularni plaintext loglash sessiya o'g'irlashga olib keladi (H-3).
var sensitiveQueryParams = map[string]bool{
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"request_id":    true,
}

// redactQuery sezgir query paramlarni "REDACTED" bilan almashtiradi, qolganini
// (page, limit, ...) audit uchun qoldiradi. Parse xato bo'lsa butun query'ni
// niqoblaydi (leak'dan ko'ra ehtiyot).
func redactQuery(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "REDACTED"
	}
	for key := range values {
		if sensitiveQueryParams[key] {
			values.Set(key, "REDACTED")
		}
	}
	return values.Encode()
}

// Logger — HAR bir so'rovni strukturaviy yozadi (request_id, method, path, status,
// latency). gin.Logger() o'rniga (u strukturasiz stdout'ga chiqaradi). Log darajasi
// status'ga qarab: 5xx→Error, 4xx→Warn, qolgani→Info. request_id/user_id ctx'dan tushadi.
func Logger(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		status := c.Writer.Status()
		reqID, _ := c.Get(HeaderRequestID)
		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.Int("status", status),
			logger.Int64("latency_ms", time.Since(start).Milliseconds()),
			logger.String("ip", c.ClientIP()),
			logger.Any("request_id", reqID),
		}
		if q := c.Request.URL.RawQuery; q != "" {
			fields = append(fields, logger.String("query", redactQuery(q)))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, logger.String("errors", c.Errors.String()))
		}

		switch {
		case status >= 500:
			log.Error(c.Request.Context(), "http request", fields...)
		case status >= 400:
			log.Warn(c.Request.Context(), "http request", fields...)
		default:
			log.Info(c.Request.Context(), "http request", fields...)
		}
	}
}
