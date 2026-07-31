package uz.darsly.mentor.data.repo

import uz.darsly.mentor.data.api.BlocklistEntry
import uz.darsly.mentor.data.api.DarslyApi
import java.io.IOException
import javax.inject.Inject

/**
 * Ustozning doimiy qora ro'yxati (№4).
 *
 * ## Nega alohida repozitoriy, [ModerationRepository] ichida emas
 * Moderatsiya amallari DARSGA tegishli va xona ekranidan chaqiriladi
 * (`lessonId` talab qiladi). Qora ro'yxat esa ustozning **hisobiga** tegishli
 * va dars tashqarisidan — Kabinetdan — boshqariladi. Ikkisini bir joyga qo'yish
 * xona ekrani va Kabinet ekranini keraksiz ravishda bir-biriga bog'lardi.
 *
 * ## Nega bu ekran umuman kerak
 * Ban qo'yish telefondan mumkin edi (`RoomViewModel.removeParticipant`), OLIB
 * TASHLASH esa faqat webda. Ya'ni dars o'rtasida shoshib bosilgan «Doimiy»
 * telefonda tuzatib bo'lmas holatga aylanardi, ustoz esa asosan telefondan
 * ishlaydi (PRODUCT.md:31).
 */
class BlocklistRepository @Inject constructor(private val api: DarslyApi) {

    suspend fun list(): Result<List<BlocklistEntry>> = runCatching {
        api.blocklist().data.orEmpty()
    }

    /** Yozuvni o'chiradi — o'quvchi shu ism bilan yana kira oladi. */
    suspend fun unblock(entryId: String): Result<Unit> = runCatching { api.unblock(entryId) }

    companion object {
        /** @see LessonsRepository.isOffline */
        fun isOffline(t: Throwable): Boolean = t is IOException
    }
}
