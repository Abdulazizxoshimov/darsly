package shared

import (
	"github.com/google/uuid"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// ValidateID resurs ID'si UUID shaklida ekanini DB'ga BORMASDAN tekshiradi.
//
// Nega kerak: barcha jadval PK'lari `uuid` tipida. Yaroqsiz matn Postgres'ga yetsa
// `22P02 invalid input syntax for type uuid` xatosi qaytadi va u apperr emas —
// handler uni 500 INTERNAL_ERROR qilib ko'rsatadi hamda Sentry'ga yozadi. Ya'ni
// `GET /lessons/abc` kabi bitta so'rov bilan har qanday foydalanuvchi log/Sentry'ni
// soxta 500'lar bilan to'ldirib, real nosozliklarni ko'rinmas qila oladi.
//
// Nega NotFound (BadRequest emas): mavjud xatti-harakat bilan izchillik. Repo
// GetByID topilmasa allaqachon `NotFound(resource)` qaytaradi, `OwnedLesson` esa
// begona resursga `Forbidden`. Yaroqsiz UUID — hech qachon mavjud bo'lolmaydigan ID,
// demak "topilmadi" semantik jihatdan to'g'ri va javob mavjud bo'lmagan (lekin
// shakli to'g'ri) UUID javobidan farq qilmaydi → hujumchi uchun classification
// oracle'i ham qolmaydi (ochiq `waitingroom` yo'llarida bu ayniqsa muhim).
//
// resource — apperr.NotFound uchun nom ("lesson", "recording", ...); repo'dagi
// nom bilan AYNAN bir xil bo'lishi kerak, aks holda javoblar farqlanib qoladi.
func ValidateID(id, resource string) error {
	if _, err := uuid.Parse(id); err != nil {
		return apperr.NotFound(resource)
	}
	return nil
}
