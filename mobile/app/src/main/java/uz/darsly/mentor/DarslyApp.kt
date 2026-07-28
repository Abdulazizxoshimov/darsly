package uz.darsly.mentor

import android.app.Application
import io.livekit.android.LiveKit
import io.livekit.android.util.LoggingLevel
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.Session
import uz.darsly.mentor.data.repo.NotificationsBadge
import uz.darsly.mentor.data.repo.NotificationsRepository
import uz.darsly.mentor.data.store.PrefsLessonsCache
import uz.darsly.mentor.data.ws.Realtime
import uz.darsly.mentor.data.ws.RealtimeEvent
import uz.darsly.mentor.service.LessonNotifications

class DarslyApp : Application() {

    /** Ilova umri davomida yashaydigan qamrov — faqat sessiya tozalash uchun. */
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    override fun onCreate() {
        super.onCreate()
        // LiveKit log'lari `adb logcat` da ko'rinsin (spike diagnostikasi uchun muhim).
        LiveKit.loggingLevel = if (BuildConfig.DEBUG) LoggingLevel.VERBOSE else LoggingLevel.WARN
        // M1: shifrlangan token saqlagichini ulaymiz. Birinchi Activity'dan OLDIN
        // bajariladi, shuning uchun `Session.isLoggedIn` navigatsiya boshida to'g'ri.
        Session.init(this)
        LessonNotifications.ensureChannel(this)
        observeLogoutCleanup()
        observeRealtime()
        observeUnreadBadge()
        // TODO(R2 · M44): Sentry Android init.
    }

    /**
     * REAL-TIME KANAL sessiya bilan birga yashaydi.
     *
     * Kirilgan bo'lsa — soket ochiladi (kutish xonasi so'rovlari, bildirishnomalar
     * **yangilashsiz** keladi); chiqilganda darhol yopiladi, aks holda ilova
     * boshqa foydalanuvchining hodisalarini olib turaverardi.
     *
     * `Application` darajasida turishining sababi: kanal ekranlar orasida uzilmasligi
     * kerak (ustoz ro'yxatdan xonaga o'tganda kutish so'rovi yo'qolmasin).
     */
    private fun observeRealtime() {
        appScope.launch {
            Session.loggedIn.collect { loggedIn ->
                if (loggedIn) Realtime.client.start(appScope) else Realtime.client.stop()
            }
        }
    }

    /**
     * O'QILMAGAN BILDIRISHNOMALAR NISHONI — pastki panelda.
     *
     * Nishon bildirishnomalar ekrani **ochilmasdan oldin** ham to'g'ri bo'lishi
     * kerak (aks holda ustoz uni ochishga sabab topmaydi), shuning uchun
     * boshlang'ich son shu yerda, kirish bilan olinadi. Keyingi jonli xabarlar
     * sonni oshiradi; ekran ochilganda esa `NotificationsViewModel` aniq
     * qiymatni yozib to'g'rilaydi.
     *
     * So'rov yiqilsa nishon shunchaki ko'rinmaydi — bu xato dialogi ko'rsatish
     * uchun sabab emas: ustoz hech narsa so'ramagan edi.
     */
    private fun observeUnreadBadge() {
        appScope.launch {
            Session.loggedIn.collect { loggedIn ->
                if (!loggedIn) {
                    NotificationsBadge.clear()
                    return@collect
                }
                NotificationsRepository.create().unreadCount()
                    .onSuccess { NotificationsBadge.set(it) }
            }
        }
        appScope.launch {
            Realtime.client.events.collect { event ->
                if (event is RealtimeEvent.Notification) NotificationsBadge.increment()
            }
        }
    }

    /**
     * MAXFIYLIK (🟡B): sessiya tugagach darslar keshi ham o'chirilsin.
     *
     * Avval keshni faqat "Chiqish" tugmasi tozalardi. Refresh muvaffaqiyatsiz bo'lib
     * **qattiq logout** sodir bo'lganda esa kesh diskda qolardi — shu telefonga
     * boshqa ustoz kirsa, ro'yxat yangilanguncha oldingi ustozning dars sarlavhalari
     * va **join havolalari** ko'rinardi.
     *
     * Shuning uchun tozalash bitta markaziy joyda: [Session.loggedIn] `false` bo'lishi
     * — sabab nima bo'lishidan qat'i nazar (tugma, refresh o'limi, keystore buzilishi)
     * — keshni o'chiradi. Boshlang'ich qiymat ATAYLAB tashlanmaydi: ilova sessiyasiz
     * ochilgan bo'lsa (masalan oldingi ishlashda token o'lgan) kesh baribir keraksiz.
     */
    private fun observeLogoutCleanup() {
        val cache = PrefsLessonsCache.create(this)
        appScope.launch {
            Session.loggedIn.collect { loggedIn ->
                if (!loggedIn) cache.clear()
            }
        }
    }
}
