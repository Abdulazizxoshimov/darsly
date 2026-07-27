package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
)

// GetAppConfig godoc
// @Summary      Ilova konfiguratsiyasi (ochiq) — mobil klient versiya/majburiy yangilash
// @Description  Mobil ilova ishga tushganda so'raydi: o'z versiyasi min_version'dan past
// @Description  bo'lsa yangilashga majburlanadi. Auth talab qilinmaydi.
// @Tags         app
// @Produce      json
// @Success      200  {object}  entity.AppConfig
// @Router       /api/v1/app-config [get]
func GetAppConfig(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Qiymatlar startup'da config'dan yig'ilgan (app/wire.go) — handler'da
		// hech qanday logika yo'q, faqat qaytarish.
		hs.Success(c, h.AppConfig)
	}
}
