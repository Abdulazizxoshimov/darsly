package uz.darsly.mentor.ui.room

/**
 * Ruxsat etilgan emoji reaksiyalar (№14).
 *
 * ## Nega ro'yxat KLIENTDA konstanta
 * Backend uni bermaydi, lekin USTIDAN tekshiradi
 * (`internal/usecase/roomstate/roomstate.go: allowedReactions`) — endpoint
 * ochiq bo'lgani uchun bu shart: aks holda `curl` bilan istalgan matn
 * yuborilib, u butun xona ekranida suzib o'tardi, ya'ni moderatsiyasiz matn
 * kanali paydo bo'lardi. Ro'yxatga yangi emoji qo'shilsa IKKALA joyga ham
 * qo'shilishi kerak, aks holda server 400 `unsupported reaction` beradi.
 *
 * To'plam va tartib web bilan bir xil
 * (`frontend/src/livekit/Controls.jsx: REACTIONS`).
 *
 * ⚠️ ❤️ ATAYLAB variatsiya selektori bilan (U+2764 U+FE0F): serverdagi kalit
 * aynan shunday va selektorsiz "❤" boshqa satr bo'lib, 400 olardi.
 */
object Reactions {

    val ALLOWED: List<String> = listOf("👍", "👏", "❤️", "😂", "😮", "🎉", "✋")

    fun isAllowed(emoji: String): Boolean = emoji in ALLOWED
}
