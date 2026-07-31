package postgres

import "strings"

// likeEscaper — LIKE/ILIKE naqshidagi maxsus belgilarni ekranlaydi.
//
// Tartib muhim: `\` BIRINCHI almashtirilishi kerak, aks holda keyingi
// almashtirishlar qo'ygan `\` belgilari ikkinchi marta ekranlanib ketardi.
var likeEscaper = strings.NewReplacer(
	`\`, `\\`,
	`%`, `\%`,
	`_`, `\_`,
)

// SearchPattern foydalanuvchi kiritgan matndan xavfsiz ILIKE naqshini yasaydi.
//
// # Muammo
//
// Qiymatlar Squirrel orqali PARAMETRLANGAN, ya'ni SQL-inyeksiya yo'q. Lekin
// `%` va `_` LIKE naqshining o'zida maxsus ma'noga ega:
//
//	`%`  → istalgan uzunlikdagi istalgan matn
//	`_`  → istalgan bitta belgi
//
// Ekranlanmasa foydalanuvchi qidiruvga bitta `%` yozib butun jadvalni
// (`WHERE title ILIKE '%%%'`) chiqarib olardi — sahifalash bo'lgani uchun bu
// katastrofa emas, lekin ikki muammo bor: (a) filtr ma'nosini yo'qotadi va
// qidiruv "hech narsa topmadi"/"hammasini topdi" bo'lib tuyuladi; (b) `%_%_%_`
// kabi naqshlar indeksdan foydalana olmaydigan og'ir skanlarga aylanadi —
// arzon so'rov bilan bazani yuklash mumkin.
//
// PostgreSQL default ekran belgisi `\`, shuning uchun `ESCAPE` bandi shart emas.
func SearchPattern(s string) string {
	return "%" + likeEscaper.Replace(s) + "%"
}
