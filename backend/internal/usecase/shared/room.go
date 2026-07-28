package shared

import "strings"

// roomPrefix — LiveKit xona nomining barqaror prefiksi.
//
// O'ZGARTIRMANG: jonli darslarning xonalari shu nom bilan yaratilgan, prefiks
// o'zgarsa ular bilan bog'liq hamma narsa (poll ovozi, webhook orqali yozuvni
// boshlash) jimgina uzilib qoladi.
const roomPrefix = "lesson_"

// RoomName dars ID sidan barqaror LiveKit xona nomini hosil qiladi.
// Room-token'ning qaysi darsga tegishli ekanini tekshirishда ham ishlatiladi
// (masalan poll ovozi begona darsga tegmasligi uchun).
func RoomName(lessonID string) string {
	return roomPrefix + lessonID
}

// LessonIDFromRoom — [RoomName] ning teskarisi: LiveKit xona nomidan dars ID si.
//
// LiveKit webhook'lari (`track_published` va h.k.) faqat xona NOMINI beradi,
// dars ID sini emas — shuning uchun bu teskari o'girish kerak. U bir joyda
// turishi muhim: prefiks ikki joyda mustaqil yozilgan bo'lsa, biri o'zgarganda
// ikkinchisi jimgina mos kelmay qolardi.
//
// `ok == false` — bu bizning xonamiz emas (begona nom); chaqiruvchi jimgina
// e'tiborsiz qoldirishi kerak, xato emas.
func LessonIDFromRoom(roomName string) (lessonID string, ok bool) {
	id, found := strings.CutPrefix(roomName, roomPrefix)
	if !found || id == "" {
		return "", false
	}
	return id, true
}
