package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS sets permissive CORS headers. Adjust AllowOrigins for production.
func CORS(allowOrigins ...string) gin.HandlerFunc {
	origins := map[string]struct{}{}
	for _, o := range allowOrigins {
		origins[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Faqat allowlist'da aniq ko'rsatilgan Origin aks ettiriladi. Allowlist bo'sh
		// bo'lsa hech qanday Origin credentials bilan aks ettirilmaydi (bo'sh-allowlist
		// = har kimga ochiq degani EMAS — bu credential leak xavfini oldini oladi).
		allowed := ""
		if origin != "" {
			if _, ok := origins[origin]; ok {
				allowed = origin
			}
		}

		if allowed != "" {
			c.Header("Access-Control-Allow-Origin", allowed)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			// Content-Disposition — fayl NOMI shu sarlavhada keladi (chat
			// transkripti, yozuv yuklab olish). Brauzer JS'ga faqat oq
			// ro'yxatdagi sarlavhalarni ko'rsatadi, shuning uchun usiz boshqa
			// origindagi SPA fayllarni server bergan nom o'rniga zaxira nom
			// bilan saqlaydi. Bir originda muammo bilinmaydi — aynan shuning
			// uchun oson o'tkazib yuboriladi.
			c.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Request-ID")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
