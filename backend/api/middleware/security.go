package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				// 'wasm-unsafe-eval' — LiveKit WASM audio/video codeclariga kerak;
				// to'liq 'unsafe-eval' (ixtiyoriy JS eval) endi berilmaydi.
				"script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: blob: https:; "+
				"font-src 'self' data:; "+
				// LiveKit SFU va MinIO turli host'larda bo'lishi mumkin — protokol
				// darajasida cheklaymiz (yalang'och '*' emas).
				"connect-src 'self' ws: wss: https:; "+
				"frame-ancestors 'none';",
		)
		if os.Getenv("APP_ENV") == "production" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}
