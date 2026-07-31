package uz.darsly.mentor.data.repo

import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import uz.darsly.mentor.util.ChatUpload

/** Yuborishga tayyor ilova: tozalangan nom, hajm va mazmun. */
data class ChatAttachment(
    val name: String,
    val size: Long,
    val bytes: ByteArray,
) {
    // `ByteArray` massiv bo'lgani uchun data-class'ning avtomatik `equals`i
    // havola bo'yicha solishtirardi — bu jimgina yolg'on natija beradi.
    override fun equals(other: Any?): Boolean =
        this === other || (
            other is ChatAttachment && name == other.name && size == other.size &&
                bytes.contentEquals(other.bytes)
            )

    override fun hashCode(): Int =
        31 * (31 * name.hashCode() + size.hashCode()) + bytes.contentHashCode()
}

/**
 * `content://` URI'ni yuborishga tayyor ilovaga aylantiradi.
 *
 * ## Nega INTERFEYS
 * Yagona implementatsiya Android `ContentResolver`iga bog'langan, ya'ni JVM
 * testida umuman ishlamaydi. Interfeys bo'lmasa [RoomChatRepository] ning
 * HTTP tomonini (o'chirish, tarix, multipart) sinash uchun butun Android
 * kontekstini emulyatsiya qilish kerak bo'lardi. `TokenStore`/`LessonsCache`
 * ham aynan shu sababdan interfeys.
 */
interface ChatAttachments {

    /**
     * Xatolar [IllegalArgumentException] bo'lib chiqadi — ularning matni
     * FOYDALANUVCHIGA ko'rsatish uchun tayyor (`ApiErrors.humanError` uni
     * o'zgartirmasdan o'tkazadi).
     */
    fun read(uri: Uri): Result<ChatAttachment>
}

/**
 * `ContentResolver` orqali o'qiydigan haqiqiy implementatsiya.
 *
 * ## Nega fayl XOTIRAGA o'qiladi
 * Chegara 20 MB va u [ChatUpload.validate] bilan O'QISHDAN OLDIN tekshiriladi.
 * Oqim (streaming) `RequestBody` provayder URI'sini yuklash paytida qayta
 * ochishni talab qiladi — OkHttp qayta urinishida (masalan 401 → refresh →
 * retry) ba'zi provayderlar oqimni ikkinchi marta bermaydi va yuklama
 * jimgina buzilardi. Bir marta o'qib olingan massiv esa har urinishda bir xil.
 */
class ContentChatAttachments(private val ctx: Context) : ChatAttachments {

    override fun read(uri: Uri): Result<ChatAttachment> = runCatching {
        val resolver = ctx.contentResolver
        var name = ChatUpload.FALLBACK_NAME
        var size = -1L

        resolver.query(uri, null, null, null, null)?.use { c ->
            if (c.moveToFirst()) {
                c.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                    .takeIf { it >= 0 && !c.isNull(it) }
                    ?.let { name = ChatUpload.sanitizeName(c.getString(it)) }
                c.getColumnIndex(OpenableColumns.SIZE)
                    .takeIf { it >= 0 && !c.isNull(it) }
                    ?.let { size = c.getLong(it) }
            }
        }

        // Hajm ma'lum bo'lsa — O'QISHDAN OLDIN tekshiramiz: 100 MB'lik videoni
        // xotiraga olib, keyin "juda katta" deyish `OutOfMemoryError` bo'lardi.
        if (size >= 0) ChatUpload.validate(name, size)?.let { throw IllegalArgumentException(it) }

        val bytes = resolver.openInputStream(uri)?.use { input ->
            // Chegaradan bitta bayt ortiq o'qiymiz: "aynan chegarada" va
            // "chegaradan katta" ni ajratish uchun.
            input.readBytesUpTo(ChatUpload.MAX_BYTES + 1)
        } ?: throw IllegalArgumentException("Faylni o'qib bo'lmadi — boshqasini tanlang")

        // Provayder hajmni bermagan (yoki noto'g'ri bergan) bo'lsa — haqiqiy
        // o'lcham bo'yicha qayta tekshiramiz.
        ChatUpload.validate(name, bytes.size.toLong())?.let { throw IllegalArgumentException(it) }

        ChatAttachment(name = name, size = bytes.size.toLong(), bytes = bytes)
    }

    private fun java.io.InputStream.readBytesUpTo(limit: Long): ByteArray {
        val out = java.io.ByteArrayOutputStream()
        val buf = ByteArray(64 * 1024)
        var total = 0L
        while (true) {
            val n = read(buf)
            if (n <= 0) break
            out.write(buf, 0, n)
            total += n
            // Chegaradan oshdi — qolganini o'qishning ma'nosi yo'q (tekshiruv
            // baribir rad etadi, lekin ortiqcha xotira band qilinmaydi).
            if (total > limit) break
        }
        return out.toByteArray()
    }
}
