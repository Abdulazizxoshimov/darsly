package uz.darsly.mentor.data.livekit

import com.squareup.moshi.JsonAdapter
import com.squareup.moshi.Moshi
import com.squareup.moshi.Types

/**
 * LiveKit data-channel xabari — dars ichidagi signal.
 *
 * Manba (shartnoma): `backend/internal/usecase/roomstate/roomstate.go` va
 * `frontend/src/livekit/messaging.js`. Uch klient (web, mobil, server) AYNI
 * shaklni ishlatadi, shuning uchun shakl shu yerda ham qotirilgan.
 */
sealed interface RoomSignal {
    /** Qo'l ko'tarildi/tushirildi. `at` — navbat tartibi uchun (Unix ms). */
    data class Hand(val identity: String, val name: String, val raised: Boolean, val at: Long) : RoomSignal

    /** Ustoz barcha qo'llarni tushirdi. */
    data object LowerAll : RoomSignal

    /** Emoji reaksiya (saqlanmaydi). */
    data class Reaction(val emoji: String, val name: String, val identity: String) : RoomSignal

    /** Chat xabari. `toIdentity != null` → shaxsiy. */
    data class Chat(
        val id: String,
        val senderIdentity: String,
        val senderName: String,
        val body: String,
        val toIdentity: String?,
    ) : RoomSignal
}

/**
 * Data-channel baytlarini turlangan signalga aylantiradi — **sof** funksiya
 * (Android'siz, JVM testida ishlaydi).
 *
 * ## Nega alohida sinf
 * `RealtimeParser` bilan bir xil sabab: real-time xatolarining eng yomon turi —
 * "xabar keldi, lekin noto'g'ri o'qildi va ilova jim yiqildi". Parse mantiqi
 * LiveKit SDK'sidan ajratilgani uchun har bir tur, buzilgan JSON va notanish
 * tur testlar bilan qoplanadi.
 *
 * ## Nega `wb` (doska) e'tiborsiz
 * Doska hozircha faqat webda chiziladi; mobil ustoz uni ko'rmaydi. Notanish
 * turlar `null` qaytaradi — ya'ni yangi signal qo'shilganda eski ilova yiqilmaydi,
 * shunchaki e'tiborsiz qoldiradi (oldinga moslik).
 */
class RoomDataParser(moshi: Moshi = Moshi.Builder().build()) {

    private val mapAdapter: JsonAdapter<Map<String, Any?>> = moshi.adapter(
        Types.newParameterizedType(Map::class.java, String::class.java, Any::class.java),
    )

    fun parse(bytes: ByteArray): RoomSignal? = parse(String(bytes, Charsets.UTF_8))

    fun parse(raw: String): RoomSignal? {
        val root = runCatching { mapAdapter.fromJson(raw) }.getOrNull() ?: return null

        // Backend chat xabarini `kind`siz, `entity.ChatMessage` shaklida yuboradi —
        // uni maydonlari bo'yicha tanib olamiz (web klient ham xuddi shunday qiladi).
        val kind = root.str("kind")
        if (kind == null) {
            val body = root.str("body") ?: return null
            val sender = root.str("sender_name") ?: return null
            return RoomSignal.Chat(
                id = root.str("id").orEmpty(),
                senderIdentity = root.str("sender_identity").orEmpty(),
                senderName = sender,
                body = body,
                toIdentity = root.str("to_identity"),
            )
        }

        return when (kind) {
            KIND_HAND -> {
                if (root.str("act") == ACT_LOWER_ALL) return RoomSignal.LowerAll
                val identity = root.str("identity") ?: return null
                RoomSignal.Hand(
                    identity = identity,
                    name = root.str("name") ?: identity,
                    raised = root["raised"] as? Boolean ?: false,
                    at = (root["at"] as? Number)?.toLong() ?: 0L,
                )
            }

            KIND_REACTION -> {
                val emoji = root.str("emoji") ?: return null
                RoomSignal.Reaction(
                    emoji = emoji,
                    name = root.str("name").orEmpty(),
                    identity = root.str("identity").orEmpty(),
                )
            }

            else -> null // 'wb', 'poll' va kelajakdagi turlar — e'tiborsiz
        }
    }

    /** Bo'sh satr "qiymat yo'q" bilan bir xil — UI'da bo'sh qator chiqmasin. */
    private fun Map<*, *>?.str(key: String): String? =
        (this?.get(key) as? String)?.trim()?.takeIf { it.isNotEmpty() }

    private companion object {
        const val KIND_HAND = "hand"
        const val KIND_REACTION = "reaction"
        const val ACT_LOWER_ALL = "lower_all"
    }
}
