package uz.darsly.mentor

import android.app.Application
import dagger.hilt.android.HiltAndroidApp
import javax.inject.Inject
import io.livekit.android.LiveKit
import io.livekit.android.util.LoggingLevel
import io.sentry.android.core.SentryAndroid
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.SessionManager
import uz.darsly.mentor.data.repo.NotificationsBadge
import uz.darsly.mentor.data.repo.NotificationsRepository
import uz.darsly.mentor.data.store.LessonsCache
import uz.darsly.mentor.data.ws.RealtimeClient
import uz.darsly.mentor.data.ws.RealtimeEvent
import uz.darsly.mentor.service.LessonNotifications

@HiltAndroidApp
class DarslyApp : Application() {

    // Hilt injeksiyasi. Avval bularning hammasi global `object` edi
    // (`Session`, `Realtime`, `NotificationsBadge`) — natijada ViewModel'larni
    // ham, bu observer'larni ham test qilib bo'lmasdi.
    @Inject lateinit var session: SessionManager
    @Inject lateinit var realtime: RealtimeClient
    @Inject lateinit var badge: NotificationsBadge
    @Inject lateinit var notifications: NotificationsRepository
    @Inject lateinit var lessonsCache: LessonsCache
    @Inject lateinit var uploadResumer: uz.darsly.mentor.data.repo.PendingUploadResumer

    /** Ilova umri davomida yashaydigan qamrov — faqat sessiya tozalash uchun. */
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    override fun onCreate() {
        super.onCreate()
        // LiveKit log'lari `adb logcat` da ko'rinsin (spike diagnostikasi uchun muhim).
        LiveKit.loggingLevel = if (BuildConfig.DEBUG) LoggingLevel.VERBOSE else LoggingLevel.WARN
        LessonNotifications.ensureChannel(this)
        observeLogoutCleanup()
        observeRealtime()
        observeUnreadBadge()
        initCrashReporting()
    }

    /**
     * CRASH-HISOBOTLARI (M15).
     *
     * # Nega kerak
     *
     * Qurilma sinovida tarmoq almashuvi paytida NATIV yiqilish qayd etilgan:
     * `libjingle_peerconnection_so.so` ichida SIGABRT (uch takrordan birida).
     * U bizning Kotlin kodimizda emas, WebRTC ichida sodir bo'ladi — ya'ni
     * oddiy `try/catch` ham, Java darajasidagi handler ham uni KO'RMAYDI.
     * Hozircha bu "bilib turilgan xavf" edi, lekin foydalanuvchilarda qanchalik
     * tez-tez takrorlanishini o'lchash imkoni yo'q edi: ustoz shunchaki
     * "ilova o'chib qoldi" deydi va iz qolmaydi.
     *
     * `sentry-android` NDK handler'ini ham o'rnatadi, ya'ni aynan shu turdagi
     * yiqilish stack-trace bilan yetib keladi. Shundan keyingina "livekit-android
     * ni yangilash" yoki `disconnect()` xulqini o'zgartirish qarorini
     * TAXMIN emas, ma'lumot asosida qabul qilish mumkin bo'ladi.
     *
     * # DSN bo'lmasa
     *
     * SDK UMUMAN ishga tushirilmaydi. Bu ataylab: yarim sozlangan telemetriya
     * (DSN yo'q, lekin SDK tirik) faqat batareya va tarmoq sarflaydi.
     */
    private fun initCrashReporting() {
        val dsn = BuildConfig.SENTRY_DSN
        if (dsn.isBlank()) return

        SentryAndroid.init(this) { options ->
            options.dsn = dsn
            options.environment = if (BuildConfig.DEBUG) "debug" else "production"
            options.release = "${BuildConfig.APPLICATION_ID}@${BuildConfig.VERSION_NAME}"
            // Nativ (NDK) yiqilishlar — bu integratsiyaning ASOSIY sababi.
            options.isEnableNdk = true
            // Namuna olish: yiqilishlar 100%, tracing esa o'chiq (bizga
            // performance emas, barqarorlik kerak; tracing trafik sarflaydi).
            options.sampleRate = 1.0
            options.tracesSampleRate = 0.0
            // MAXFIYLIK: foydalanuvchi ma'lumotlari (IP, qurilma identifikatori)
            // yuborilmaydi. Dars mazmuni va join havolalari hech qachon
            // hisobotga tushmasligi kerak.
            options.isSendDefaultPii = false
        }
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
            session.loggedIn.collect { loggedIn ->
                if (loggedIn) {
                    realtime.start(appScope)
                    // Yuklanmay qolgan lokal yozuvlarni AVTOMATIK qayta yuklaymiz
                    // (ilova o'rtada o'lgan bo'lsa ham — tugma kerak emas).
                    uploadResumer.resume(appScope)
                } else {
                    realtime.stop()
                    // S4: chiqilgach o'quvchilar yozuvi telefonda qolmasin —
                    // keyingi kirgan ustoz oldingisining darsini ko'rmaydi.
                    uploadResumer.clearAll(appScope)
                }
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
            session.loggedIn.collect { loggedIn ->
                if (!loggedIn) {
                    badge.clear()
                    return@collect
                }
                notifications.unreadCount()
                    .onSuccess { badge.set(it) }
            }
        }
        appScope.launch {
            realtime.events.collect { event ->
                if (event is RealtimeEvent.Notification) badge.increment()
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
     * Shuning uchun tozalash bitta markaziy joyda: [session.loggedIn] `false` bo'lishi
     * — sabab nima bo'lishidan qat'i nazar (tugma, refresh o'limi, keystore buzilishi)
     * — keshni o'chiradi. Boshlang'ich qiymat ATAYLAB tashlanmaydi: ilova sessiyasiz
     * ochilgan bo'lsa (masalan oldingi ishlashda token o'lgan) kesh baribir keraksiz.
     */
    private fun observeLogoutCleanup() {
        val cache = lessonsCache
        appScope.launch {
            session.loggedIn.collect { loggedIn ->
                if (!loggedIn) cache.clear()
            }
        }
    }
}
