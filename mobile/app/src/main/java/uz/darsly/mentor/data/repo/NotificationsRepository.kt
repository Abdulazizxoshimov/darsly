package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.DarslyApi
import javax.inject.Inject
import uz.darsly.mentor.data.api.Notification
import java.io.IOException

/** Bir yangilanish natijasi: bildirishnomalar + serverdagi umumiy son. */
data class NotificationsPage(
    val items: List<Notification>,
    val total: Int,
    val truncated: Boolean,
)

/**
 * Bildirishnomalar (R2 · "Bildirishnomalar" bandi).
 *
 * Real-time kanal allaqachon bor edi ([uz.darsly.mentor.data.ws.RealtimeClient]),
 * lekin uni ko'rsatadigan ekran yo'q edi: kelgan xabar jimgina yo'qolardi.
 * Bu qatlam **tarixni** beradi, WS esa uning ustiga jonli qo'shimcha.
 *
 * Qatlam yo'nalishi darslar bilan bir xil: `ui/ → data/repo/ → data/api/`.
 */
class NotificationsRepository @Inject constructor(private val api: DarslyApi) {

    /**
     * To'liq ro'yxat — `total` qoplanguncha sahifalanadi (darslar bilan bir xil
     * naqsh, [LessonsRepository.refresh] izohiga qara).
     *
     * Bildirishnomalar darslardan **ko'p** bo'ladi (har dars eslatmasi bitta
     * yozuv), shuning uchun chegara ham muhimroq: oshsa [NotificationsPage.truncated]
     * bilan oshkora aytiladi, jimgina qirqilmaydi.
     */
    suspend fun refresh(pageSize: Int = PAGE_SIZE): Result<NotificationsPage> = runCatching {
        val all = mutableListOf<Notification>()
        var page = 1
        var total = 0
        var truncated = false
        while (true) {
            val env = api.notifications(limit = pageSize, page = page)
            val chunk = env.data.orEmpty()
            all += chunk
            total = env.total
            if (chunk.isEmpty() || all.size >= total) break
            if (page >= MAX_PAGES) {
                truncated = true
                break
            }
            page++
        }
        NotificationsPage(items = all, total = total, truncated = truncated)
    }

    suspend fun unreadCount(): Result<Int> = runCatching {
        api.unreadCount().data?.count ?: 0
    }

    suspend fun markRead(id: String): Result<Unit> = runCatching { api.markNotificationRead(id) }

    suspend fun markAllRead(): Result<Unit> = runCatching { api.markAllNotificationsRead() }

    companion object {
        const val PAGE_SIZE = 50

        /** 10 × 50 = 500 bildirishnoma. Oshsa foydalanuvchiga aytiladi. */
        const val MAX_PAGES = 10

        /** @see LessonsRepository.isOffline */
        fun isOffline(t: Throwable): Boolean = t is IOException

    }
}
