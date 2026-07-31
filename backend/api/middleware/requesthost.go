package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/internal/usecase/shared"
)

// RequestHost so'rovning Host sarlavhasini usecase qatlamiga kontekst orqali
// yetkazadi.
//
// Iste'molchisi — livekit.ClientWSURL "auto" rejimi: klientga qaytariladigan
// signaling manzili klient API'ga qaysi host bilan kelganidan yasaladi (dev'da
// telefon LAN IP bilan, brauzer localhost bilan keladi — ikkalasiga o'z manzili
// kerak). Reverse-proxy ortida gin o'zi X-Forwarded-Host'ni qo'llamaydi, lekin
// production'da "auto" ishlatilmaydi (oshkora LIVEKIT_CLIENT_WS_URL beriladi).
func RequestHost() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request = c.Request.WithContext(
			shared.WithRequestHost(c.Request.Context(), c.Request.Host),
		)
		c.Next()
	}
}
