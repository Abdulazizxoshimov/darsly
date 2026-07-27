package shared

// RoomName dars ID sidan barqaror LiveKit xona nomini hosil qiladi.
// Room-token'ning qaysi darsga tegishli ekanini tekshirishда ham ishlatiladi
// (masalan poll ovozi begona darsga tegmasligi uchun).
func RoomName(lessonID string) string {
	return "lesson_" + lessonID
}
