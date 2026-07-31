package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BearerToken — statik token bilan himoya (JWT emas).
//
// Foydalanish joyi: mashinaga mo'ljallangan, foydalanuvchi hisobi bo'lmagan
// endpointlar — hozircha `/metrics` (Prometheus scrape). Bunday chaqiruvchida
// hisob ham, sessiya ham yo'q, ya'ni JWT/Casbin zanjiri umuman mos kelmaydi.
//
// Taqqoslash `subtle.ConstantTimeCompare` bilan: oddiy `==` satrlarni birinchi
// farqda to'xtatadi va javob vaqti bo'yicha tokenni bayt-bayt topish mumkin
// bo'lardi. Bu yerda o'lchov shovqini katta bo'lsa-da, to'g'ri usul arzon.
func BearerToken(expected string) gin.HandlerFunc {
	want := []byte(expected)
	return func(c *gin.Context) {
		got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		// Uzunlik ham tekshiriladi: ConstantTimeCompare turli uzunlikda 0
		// qaytaradi, lekin bu niyatni kodda ochiq ko'rsatadi.
		if len(got) != len(expected) || subtle.ConstantTimeCompare([]byte(got), want) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
