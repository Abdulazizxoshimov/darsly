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
        /** Ilova qilingan fayl (`null` — oddiy matnli xabar). */
        val file: ChatFile? = null,
    ) : RoomSignal

    /**
     * Xabar o'chirildi (moderatsiya, №6) — ID bo'yicha ro'yxatdan olib tashlanadi.
     *
     * Xabar MAZMUNI hodisada YO'Q va qabrtosh («xabar o'chirilgan») ham
     * qoldirilmaydi: o'chirilgan joyni belgilash buzg'unchiga aynan u xohlagan
     * e'tiborni berardi.
     */
    data class ChatDeleted(val id: String) : RoomSignal

    /**
     * So'rovnoma natijasi e'lon qilindi (№7).
     *
     * Natijaning O'ZI ham hodisa ichida keladi — 300 kishilik xonada har biri
     * alohida so'rov yuborsa bu 300 ta ortiqcha so'rov bo'lardi.
     */
    data class PollPublished(
        val pollId: String,
        val counts: List<Int>,
        val total: Int,
    ) : RoomSignal

    /** Chat xabariga ilova qilingan fayl. [url] — muddatli presigned havola. */
    data class ChatFile(
        val name: String,
        val size: Long,
        val mime: String,
        val url: String,
    )
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
            val sender = root.str("sender_name") ?: return null
            val body = root.str("body")
            val file = chatFile(root["file"] as? Map<*, *>)
            // ⚠️ FAYL XABARIDA `body` BO'SH BO'LISHI MUMKIN (№15): ilova
            // qilingan fayl izohsiz yuborilsa server `body` ni bo'sh qoldiradi.
            // Avval bu yerda `body` majburiy edi va bunday xabar JIM tashlanardi:
            // fayl yuborilardi, lekin xonadagi ustoz ekranida hech nima
            // ko'rinmasdi (tarixni qayta yuklaganda esa paydo bo'lardi).
            if (body == null && file == null) return null
            return RoomSignal.Chat(
                id = root.str("id").orEmpty(),
                senderIdentity = root.str("sender_identity").orEmpty(),
                senderName = sender,
                body = body.orEmpty(),
                toIdentity = root.str("to_identity"),
                file = file,
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

            KIND_CHAT_DELETED -> {
                val id = root.str("id") ?: return null
                RoomSignal.ChatDeleted(id)
            }

            KIND_POLL_PUBLISHED -> {
                val results = root["results"] as? Map<*, *> ?: return null
                val pollId = (results["poll"] as? Map<*, *>).str("id") ?: return null
                val counts = (results["counts"] as? List<*>).orEmpty()
                    .mapNotNull { (it as? Number)?.toInt() }
                RoomSignal.PollPublished(
                    pollId = pollId,
                    counts = counts,
                    total = (results["total"] as? Number)?.toInt() ?: counts.sum(),
                )
            }

            else -> null // 'wb' va kelajakdagi turlar — e'tiborsiz
        }
    }

    /** `file` obyekti; nomi bo'lmasa ilova yo'q deb qaraladi. */
    private fun chatFile(raw: Map<*, *>?): RoomSignal.ChatFile? {
        val name = raw.str("name") ?: return null
        return RoomSignal.ChatFile(
            name = name,
            size = (raw?.get("size") as? Number)?.toLong() ?: 0L,
            mime = raw.str("mime").orEmpty(),
            url = raw.str("url").orEmpty(),
        )
    }

    /** Bo'sh satr "qiymat yo'q" bilan bir xil — UI'da bo'sh qator chiqmasin. */
    private fun Map<*, *>?.str(key: String): String? =
        (this?.get(key) as? String)?.trim()?.takeIf { it.isNotEmpty() }

    private companion object {
        const val KIND_HAND = "hand"
        const val KIND_REACTION = "reaction"
        const val KIND_CHAT_DELETED = "chat_deleted"
        const val KIND_POLL_PUBLISHED = "poll_published"
        const val ACT_LOWER_ALL = "lower_all"
    }
}
