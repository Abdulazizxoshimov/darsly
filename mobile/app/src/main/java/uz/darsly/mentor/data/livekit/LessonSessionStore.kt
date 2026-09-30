package uz.darsly.mentor.data.livekit

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import uz.darsly.mentor.data.api.RoomToken
import uz.darsly.mentor.di.ApplicationScope
import uz.darsly.mentor.service.LessonPlatform
import javax.inject.Inject
import javax.inject.Singleton

/**
 * Jonli dars sessiyasining YAGONA egasi (M2).
 *
 * ## Avval nima bo'lgan
 * To'rt mustaqil "ega" bor edi: `LessonSessionHolder` (process `object`),
 * foreground servis, `RoomViewModel` va navigatsiya. Har biri o'zicha
 * bo'shatardi va bir-birini bilmasdi. Oqibati C1: "Yakunlash" `viewModelScope`
 * da ishga tushar, navigatsiya ekranni yopib qamrovni bekor qilar va
 * bo'shatish HECH QACHON yetib bormasdi — xona, servis, MediaProjection tirik,
 * yozuv `moov`siz.
 *
 * ## Endi
 * Sessiya shu yerda yashaydi va **faqat shu yerdan** bo'shatiladi. Bo'shatish
 * [scope] — ilova umri qamrovida — bajariladi, chaqiruvchi (ViewModel, servis,
 * Activity) qamrovi bekor qilinsa ham tugaydi. `LessonService` egalikning
 * hayotiy-sikl qorovuli: u o'lganda (`onTaskRemoved`/`onDestroy`) [releaseAll]
 * chaqiriladi; [release] esa servisni to'xtatadi — ikkala yo'l ham idempotent
 * va bir nuqtaga keladi.
 *
 * UI sessiyani [session] oqimidan o'qiydi — global `object` ga to'g'ridan-to'g'ri
 * murojaat yo'q.
 */
@Singleton
class LessonSessionStore @Inject constructor(
    private val factory: LessonSessionFactory,
    private val platform: LessonPlatform,
    @ApplicationScope val scope: CoroutineScope,
) {
    private val _session = MutableStateFlow<LessonSession?>(null)
    val session: StateFlow<LessonSession?> = _session.asStateFlow()

    /** Boshlash va bo'shatish ketma-ket — bir vaqtda ikki sessiya bo'lmasin. */
    private val lock = Mutex()

    /** Yangi sessiya. Avvalgisi (bo'lsa) avval TO'LIQ bo'shatiladi. */
    suspend fun start(lessonId: String, token: RoomToken): LessonSession = lock.withLock {
        // Eskisi oqimdan AVVAL olinadi: uni bo'shatish servisni to'xtatadi va
        // servisning `onDestroy` "joriy sessiyani bo'shat" deb keladi — u
        // yangi sessiyani ko'rmasligi kerak.
        _session.value?.let { old ->
            _session.value = null
            teardown(old)
        }
        factory.create(lessonId, token).also { _session.value = it }
    }

    /**
     * [expected] sessiyani bo'shatadi. Joriy sessiya boshqa bo'lsa — hech nima
     * (M6: eski sessiyaning kechikkan tozalashi yangisini o'ldirmasin).
     */
    suspend fun release(expected: LessonSession) = lock.withLock {
        if (_session.value !== expected) return@withLock
        _session.value = null
        teardown(expected)
    }

    /**
     * HOZIRGI sessiyani (bo'lsa) bo'shatadi — servis o'lganda, logoutda.
     *
     * Sessiya chaqiruv paytida ushlanadi, qulf ostida emas: aks holda `start`
     * ichida to'xtatilgan servisning kechikkan `onDestroy` si qulfni kutib,
     * keyin YANGI sessiyani o'ldirardi.
     */
    suspend fun releaseAll() {
        val current = _session.value ?: return
        release(current)
    }

    /** [release] ning ilova qamrovidagi ko'rinishi — chaqiruvchi qamrovidan mustaqil. */
    fun releaseAsync(expected: LessonSession): Job = scope.launch { release(expected) }

    /** Sessiya SINXRON ushlanadi (yuqoridagi sabab) — korutina faqat bo'shatishni bajaradi. */
    fun releaseAllAsync(): Job? = _session.value?.let { releaseAsync(it) }

    private suspend fun teardown(s: LessonSession) {
        runCatching { s.release() }
        platform.hideShareFrame()
        platform.stopService()
    }
}
