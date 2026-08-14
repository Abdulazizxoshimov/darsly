package uz.darsly.mentor.ui.room

import android.content.Context
import android.content.Intent
import android.os.Build
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
import io.livekit.android.room.track.Track
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.repo.LessonsRepository
import uz.darsly.mentor.data.repo.LocalRecordingRepository
import uz.darsly.mentor.data.repo.ModerationRepository
import uz.darsly.mentor.data.repo.RoomChatRepository
import uz.darsly.mentor.data.repo.RoomRepository
import uz.darsly.mentor.data.livekit.HandQueue
import uz.darsly.mentor.data.livekit.LessonSession
import uz.darsly.mentor.data.livekit.LessonSessionHolder
import uz.darsly.mentor.data.livekit.NetworkMonitor
import uz.darsly.mentor.data.livekit.NetworkSwitchPolicy
import uz.darsly.mentor.data.livekit.Transport
import uz.darsly.mentor.data.livekit.RaisedHand
import uz.darsly.mentor.data.livekit.ReactionFeed
import uz.darsly.mentor.data.livekit.RoomDataParser
import uz.darsly.mentor.data.livekit.RoomReaction
import uz.darsly.mentor.data.livekit.RoomSignal
import uz.darsly.mentor.data.livekit.ScreenAudioPolicy
import uz.darsly.mentor.data.livekit.ScreenSharePlan
import uz.darsly.mentor.data.livekit.ShareBackgroundPlan
import uz.darsly.mentor.data.store.UiPrefs
import uz.darsly.mentor.service.LessonNotifications
import uz.darsly.mentor.service.LessonService
import uz.darsly.mentor.service.ShareFrameOverlay
import uz.darsly.mentor.util.LessonFormat

/**
 * Ekranda ko'rsatiladigan ishtirokchi (M20).
 *
 * `videoTrack` — SDK turi: uni UI qatlamiga olib chiqish ataylab. Video kadrni
 * ko'rsatish uchun trekning o'ziga renderer ulash kerak; uni "sof" turga o'rash
 * faqat ortiqcha qatlam qo'shardi va hech narsani soddalashtirmasdi.
 */
data class ParticipantUi(
    val identity: String,
    val name: String,
    val videoTrack: VideoTrack?,
    val micMuted: Boolean,
    val speaking: Boolean,
)

/** Chat xabarining ko'rinish modeli. `privateWith` — shaxsiy bo'lsa suhbatdosh. */
data class ChatMessageUi(
    val id: String,
    val name: String,
    val body: String,
    val self: Boolean,
    val toIdentity: String?,
    /** Ilova qilingan fayl (`null` — oddiy matnli xabar). */
    val file: ChatFileUi? = null,
)

/**
 * Chatdagi fayl kartochkasi.
 *
 * [url] **muddatli** (presigned, 1 soat) — u har javobda qaytadan imzolanadi.
 * Shu sabab uzoq turgan xona ekranidagi eski havola ishlamay qolishi mumkin;
 * bu ongli murosa (muqobil — har bosishda alohida so'rov).
 */
data class ChatFileUi(
    val name: String,
    val size: Long,
    val url: String,
)

/**
 * Moderatsiya paneli qatori.
 *
 * `canSpeak` — o'quvchiga so'zga ruxsat berilganmi. Bu ma'lumot LiveKit'dan
 * kelmaydi (`ListParticipants` huquqlarni qaytarmaydi), shuning uchun ustoz
 * bergan ruxsat KLIENTDA eslab qolinadi va panel shuni ko'rsatadi. Server
 * haqiqati baribir LiveKit'da — bu faqat tugma matnini to'g'ri chiqarish uchun.
 */
data class RosterEntry(
    val identity: String,
    val name: String,
    val audioMuted: Boolean,
    val handRaised: Boolean,
)

data class RoomUiState(
    val connecting: Boolean = false,
    val connState: String = "disconnected",
    val roomName: String? = null,
    val identity: String? = null,
    val participantCount: Int = 0,
    val micOn: Boolean = false,
    val camOn: Boolean = false,
    val screenOn: Boolean = false,
    val screenAudioOn: Boolean = false,
    /** B-4: audio oqmasa — NEGA oqmayotgani. UI shu sababni ko'rsatadi. */
    val screenAudioBlocked: ScreenAudioPolicy.Blocked = ScreenAudioPolicy.Blocked.NOT_SHARING,
    /** M16: aloqa sifati (LiveKit `ConnectionQualityChanged` dan). */
    val linkQuality: RoomStatus.LinkQuality = RoomStatus.LinkQuality.UNKNOWN,
    /** M16: LiveKit qayta ulanishga urinyapti — ustoz buni ko'rishi kerak. */
    val reconnecting: Boolean = false,
    /** B-6: dars tugadi (server yopdi / aloqa uzildi) — resurslar bo'shatilgan. */
    val endedMessage: String? = null,
    /**
     * Dars SERVERDA hali `live`mi. [lessonActive] dan farqi: bu — darsning o'zi,
     * u — bizning ulanishimiz. Aloqa uzilsa ham dars serverda davom etayotgan
     * bo'lishi mumkin; bunda chiqishdan oldin "Yakunlaysizmi?" so'ralishi SHART,
     * aks holda dars abadiy `live` bo'lib qoladi (2026-07-30 da qurilmada ko'rindi).
     */
    val lessonLive: Boolean = false,
    /** M12: old kamera faolmi (`false` = orqa kamera). */
    val cameraFront: Boolean = true,
    /** M20: xonadagi o'quvchilar (o'zimizdan tashqari). */
    val participants: List<ParticipantUi> = emptyList(),
    /** M20: o'z kameramiz treki — kichik oynada ko'rsatiladi. */
    val localVideo: VideoTrack? = null,
    /** Dars nomi — sarlavhada UUID o'rniga shu ko'rsatiladi. */
    val lessonTitle: String? = null,
    /**
     * Shu dars yozib olinadimi (`is_recording_enabled`).
     *
     * Xonadagi REC indikatori AYNAN shunga bog'lanadi. Uni "ulangan bo'lsak
     * yozilyapti" deb taxmin qilish mumkin emas: yozuv default yoniq bo'lsa-da,
     * ustoz uni o'chirgan bo'lishi mumkin — u holda REC ko'rsatish yolg'on
     * bo'lardi (va maxfiylik masalasida yolg'on eng yomoni).
     */
    val recordingEnabled: Boolean = false,
    /**
     * HOZIR haqiqatan yozib olinyaptimi (lokal recorder ishlayaptimi).
     *
     * [recordingEnabled] darsning SOZLAMASI (avto-yozuv yoniqmi) bo'lsa, bu —
     * ustoz Record tugmasi bilan boshqaradigan JORIY holat. Tepadagi REC
     * indikatori aynan shunga bog'lanadi: sozlama yoniq bo'lsa-yu, ustoz
     * yozuvni to'xtatgan bo'lsa, indikator yonmasligi kerak (maxfiylik).
     */
    val isRecording: Boolean = false,
    /** Record bosildi-yu, yozib bo'lmadi (masalan, ekran ulashilmagan) — o'tkinchi hint. */
    val recordHint: String? = null,
    val error: String? = null,
    /** B-2: mikrofonga ruxsat berilmadi — dars boshlanmaydi. */
    val micDenied: Boolean = false,
    /**
     * Qo'l ko'targanlar — NAVBAT tartibida (server `at` bo'yicha).
     *
     * Ustoz ekran ulashganda ilova fonda qoladi va bu ro'yxat ekranda
     * ko'rinmaydi — shuning uchun u bir vaqtning o'zida BILDIRISHNOMA matniga
     * ham chiqariladi (`LessonNotifications.signals`). Aks holda darsning eng
     * muhim signali aynan eng muhim paytda yo'qolardi.
     */
    val hands: List<RaisedHand> = emptyList(),
    /** Oxirgi reaksiyalar (o'tkinchi, saqlanmaydi). */
    val reactions: List<RoomReaction> = emptyList(),
    /** Chat xabarlari (eskidan yangiga). Server saqlaydi — bu faqat ko'rinish. */
    val chat: List<ChatMessageUi> = emptyList(),
    /** O'qilmagan xabarlar — chat oynasi yopiq bo'lganda ortadi. */
    val unreadChat: Int = 0,
    /** Fayl yuklanmoqda — chatdagi biriktirish tugmasi bloklanadi. */
    val chatUploading: Boolean = false,
    /**
     * Chat/reaksiya amalining natijasi (snackbar uchun).
     *
     * [error] dan ATAYLAB alohida: `error` xona ekranida "Ulanmadi" kartasini
     * chiqaradi va "Qayta urinish" tugmasini beradi — o'chirilmagan bitta
     * xabar yoki 429 olgan reaksiya uchun bu butunlay noto'g'ri javob bo'lardi.
     */
    val notice: String? = null,
    /**
     * So'rovnoma o'zgargani signali — har `poll_published` da bittaga ortadi.
     *
     * Nega SON, `Boolean` emas: bu HODISA, va bayroq bilan ketma-ket ikki
     * o'zgarish bitta bo'lib qolardi. Panel shu qiymatga `LaunchedEffect`
     * bilan bog'lanadi va har ortishda ro'yxatni qayta so'raydi.
     */
    val pollRevision: Int = 0,
    /** Xonadagi ishtirokchilar (moderatsiya paneli uchun, serverdan). */
    val roster: List<RosterEntry> = emptyList(),
    /**
     * Tarmoq almashgani haqidagi qisqa izoh (C-11).
     *
     * Ustoz Wi-Fi'dan chiqib ketganda ekran jim qotib qolmasligi kerak: u nima
     * bo'layotganini bilsa kutadi, bilmasa ilovani yopib qayta ochadi (va dars
     * haqiqatan uziladi).
     */
    val networkNote: String? = null,
    /**
     * Uzilishdan keyin ekran ulashishni tiklash TAKLIFI (C-11 · 3-gipoteza).
     *
     * Android 14+ da rozilikni qayta so'ramasdan tiklab bo'lmaydi (platforma
     * cheklovi, `ScreenSharePlan` ga qarang) — shuning uchun bu bayroq yonganda
     * ekranda bir bosishlik "Davom ettirish" kartasi chiqadi.
     */
    val restoreShare: Boolean = false,
    /**
     * ⭐ Ulashish boshlandi — ilovani fonga olish TALABI (№25, bir martalik hodisa).
     *
     * Nega holatda va nega bir martalik: qarorni [ShareBackgroundPlan] chiqaradi
     * (u sof va test ostida), bajarilishi esa Activity'ni talab qiladi
     * (`moveTaskToBack`) — ya'ni faqat Compose qatlamida mumkin. Ko'rsatilgach
     * `shareBackgroundShown()` uni tozalaydi, aks holda ekran har qayta
     * chizilganda ilova o'zini yana fonga tashlardi.
     */
    val shareBackground: ShareBackgroundPlan.Decision? = null,
    /** Spike diagnostikasi: ekranga chiqadigan qisqa jurnal. */
    val log: List<String> = emptyList(),
) {
    /** Xona ekranidagi "Ekran audiosi" qatori — rost matn (B-4). */
    val screenAudioLabel: String
        get() = ScreenAudioPolicy.statusLabel(screenAudioOn, screenAudioBlocked)

    /** M16: aloqa yozuvi; `null` bo'lsa ko'rsatilmaydi (hammasi joyida). */
    val linkLabel: String?
        get() = RoomStatus.linkLabel(linkQuality, reconnecting)

    /** M16: indikator ogohlantiruvchi rangda bo'lsinmi. */
    val linkIsWarning: Boolean
        get() = RoomStatus.isWarning(linkQuality, reconnecting)

    /**
     * Dars **haqiqatan ketyaptimi**? (🟢L)
     *
     * Tizim "Orqaga" tugmasi shu qiymatga qarab tasdiq so'raydi: jonli dars
     * o'rtasida tasodifiy bosilgan "Orqaga" 90 daqiqalik darsni uzib qo'ymasligi
     * kerak. Sof xususiyat — JVM testida sinaladi.
     *
     * `connecting` ham hisoblanadi: ulanish jarayonida chiqib ketish yarim ochilgan
     * sessiya va tirik foreground servis qoldirardi.
     */
    val lessonActive: Boolean
        get() = connecting || connState == "connected" || connState == "reconnecting" ||
            screenOn || lessonLive

    /**
     * "Ekran audiosi" sababini joriy holatdan qayta hisoblaydi (B-4).
     *
     * `RECORD_AUDIO` ruxsati bu yerda tekshirilmaydi: mikrofonsiz xonaga umuman
     * kirilmaydi (B-2), ya'ni [micOn] `true` bo'lsa ruxsat bor.
     *
     * [sdkInt] OSHKORA parametr: `Build.VERSION.SDK_INT` ni funksiya ichida o'qish
     * yashirin global bog'liqlik bo'lardi va bu holatni testda tekshirib bo'lmasdi
     * (JVM testida stub `0` qaytaradi → "eski Android" degan yolg'on sabab).
     */
    fun withAudioReason(sdkInt: Int = Build.VERSION.SDK_INT): RoomUiState = copy(
        screenAudioBlocked = ScreenAudioPolicy.blockedBy(
            sdkInt = sdkInt,
            hasRecordAudio = true,
            sharing = screenOn,
            micOn = micOn,
        ),
    )
}

/**
 * Xotirada saqlanadigan eng ko'p chat xabari.
 *
 * Cheklov MAJBURIY: 90 daqiqalik darsda chat cheksiz o'ssa ro'yxat ham, Compose
 * render'i ham og'irlashadi. To'liq tarix serverda qoladi.
 */
private const val MAX_CHAT = 200

/** Lokal yozuv video bitreyti (bps). Ekran/slayd kontenti yaxshi siqiladi —
 *  2 Mbps Zoom-darajа ravshanlik + mo''tadil hajm (soatiga ~0.4–0.9 GB). */
private const val LOCAL_REC_BITRATE = 2_000_000

@HiltViewModel
class RoomViewModel @Inject constructor(
    @ApplicationContext private val appContext: Context,
    private val rooms: RoomRepository,
    private val moderation: ModerationRepository,
    private val chatRepo: RoomChatRepository,
    private val lessons: LessonsRepository,
    private val localRec: LocalRecordingRepository,
    private val uiPrefs: UiPrefs,
) : ViewModel() {

    // Client-side (lokal) yozuv holati.
    private var localRecId: String? = null
    private var localRecStartMs = 0L

    private val _state = MutableStateFlow(RoomUiState())
    val state: StateFlow<RoomUiState> = _state.asStateFlow()

    private var session: LessonSession? = null

    /** Data-channel signallarini o'qiydi (sof, JVM testida qoplangan). */
    private val dataParser = RoomDataParser()

    /** Reaksiya yozuvlariga barqaror kalit — Compose `key` uchun. */
    private var reactionSeq = 0L

    /** Eskirgan reaksiyalarni tozalovchi yagona ish (qarang: [scheduleReactionPrune]). */
    private var reactionPruneJob: Job? = null

    /**
     * Tarmoq almashuvi va majburiy qayta ulanish — alohida controller'da.
     *
     * Unda backoff sikli, `intentional` bayrog'i va tarmoq kuzatuvi birga
     * yashaydi; ular bu sinfda boshqa mas'uliyatlar bilan aralashib yotgan edi.
     */
    private val reconnects by lazy {
        ReconnectController(
            appContext = appContext,
            scope = viewModelScope,
            onState = { transform -> _state.update(transform) },
            onLog = { line -> log(line) },
            onReconnected = { restoreScreenShare() },
        )
    }

    /**
     * Ekran ulashish — alohida controller'da.
     *
     * Bu blok bu yerdagi eng murakkabi edi: o'z holati (niyat + saqlangan
     * rozilik), Android 14+ ning maxsus qoidalari va uch tarmoqli tiklash
     * mantiqi. Ajratilgach u LiveKit'siz sinaladi va bu sinf 130 qatorga
     * yengillashdi.
     */
    private val screenShare = ScreenShareController(
        appContext = appContext,
        onState = { transform -> _state.update(transform) },
        onLog = { line -> log(line) },
    )

    /**
     * Ulanish qorovuli. Xato bo'lganda holat IDLE ga qaytadi — ustoz "Qayta urinish"
     * bosa haqiqatan qayta urinib ko'riladi (avvalgi versiyada bloklanib qolardi).
     */
    private val joinGuard = JoinGuard()

    /** Sessiya oqimlarini kuzatuvchi korutinalar — bo'shatishda bekor qilinadi. */
    private val observeJobs = mutableListOf<Job>()

    /**
     * Shu dars sessiyasida "fonga o'tish" qarori allaqachon bajarilganmi (№25).
     *
     * QAMROVI ATAYLAB DARS, ulashish emas. `ScreenShareController.reset()` da
     * saqlansa bo'lardi, lekin u ustoz ulashishni to'xtatganda ham chaqiriladi —
     * ya'ni ikkinchi ulashishda ilova yana o'zini fonga tashlardi. Ustoz esa
     * o'sha payt ataylab ILOVAGA QAYTGAN bo'lishi mumkin (masalan chatni
     * o'qigan). Kutilmagan fonga o'tish — kutilmagan foreground'dan yomonroq.
     */
    private var shareBackgroundHandled = false

    private fun log(line: String) {
        _state.update { it.copy(log = (it.log + line).takeLast(40)) }
    }

    /** B-2: mikrofon rad etildi — ulanmaymiz, foydalanuvchiga tushuntiramiz. */
    fun onMicrophoneDenied() {
        _state.update { it.copy(connecting = false, micDenied = true) }
        log("mikrofonga ruxsat berilmadi — ulanish boshlanmadi")
    }

    fun clearPermissionError() {
        _state.update { it.copy(micDenied = false) }
    }

    /**
     * `POST /lessons/:id/token` → host RoomToken → LiveKit'ga ulanish.
     * Sessiya [LessonSessionHolder] da yashaydi (Activity qayta yaratilsa uzilmaydi).
     *
     * DIQQAT (B-2): bu funksiya FAQAT ruxsat natijasi ma'lum bo'lgach chaqiriladi.
     * [withCamera] = kamera ruxsati berilganmi; berilmagan bo'lsa kamera yoqilmaydi
     * (dars ovoz + ekran ulashish bilan davom etadi).
     */
    fun join(lessonId: String, withCamera: Boolean = true) {
        if (!joinGuard.tryBegin()) return
        // Yangi dars = "fonga o'tish" qarori yana bir marta beriladi (№25).
        shareBackgroundHandled = false
        _state.update { it.copy(connecting = true, error = null, micDenied = false) }
        viewModelScope.launch {
            val ctx = appContext
            // KUTILMAGAN ISTISNO QOROVULI (🟡A).
            //
            // Quyidagi ikki chaqiruv `runCatching` bilan qoplanmagan edi:
            // `LessonService.start` (Android 12+ da fon rejimidan chaqirilsa
            // `ForegroundServiceStartNotAllowedException`) va `LessonSessionHolder.start`.
            // Ular otilsa korutina jim o'lardi va [joinGuard] CONNECTING da qotib qolardi:
            // ekranda **abadiy spinner**, `error == null` bo'lgani uchun "Qayta urinish"
            // tugmasi ham chiqmasdi — ya'ni tuzatilgan xatodan yomonroq holat.
            //
            // `CancellationException` qayta otiladi: u xato emas, ViewModel yopilishi
            // (yoki `viewModelScope` bekor qilinishi) signali.
            try {
                // Dars boshlanishi bilan foreground servis: mikrofon/kamera fon rejimida
                // o'lmasin (ekran ulashish tipi keyinroq, ruxsat olingandan so'ng qo'shiladi).
                LessonService.start(ctx, withProjection = false)

                val token = rooms.hostToken(lessonId)
                    .getOrElse { t ->
                        _state.update { it.copy(connecting = false, error = ApiErrors.humanError(t)) }
                        releaseSession()
                        LessonService.stop(ctx)
                        return@launch
                    }
                // `RoomRepository.hostToken` bo'sh javobni xatoga aylantiradi,
                // shuning uchun bu yerda `token` non-null (avvalgi qo'shimcha
                // null-tekshiruv o'lik kodga aylandi).
                log("token OK · room=${token.roomName} · ws=${token.wsUrl}")
                // Host token berildi = server darsni `live` ga o'tkazdi. Bundan
                // keyin chiqish faqat "Yakunlash / Vaqtincha chiqish" savoli bilan.
                _state.update { it.copy(lessonLive = true) }

                // Sarlavha uchun dars nomi (UUID emas). Alohida, muvaffaqiyatsizlikka
                // chidamli so'rov: nom kelmasa ham dars boshlanaverishi kerak.
                lessons.byId(lessonId)
                    ?.let { lesson ->
                        _state.update {
                            it.copy(
                                lessonTitle = lesson.title,
                                recordingEnabled = lesson.isRecordingEnabled,
                            )
                        }
                    }

                val s = LessonSessionHolder.start(ctx, lessonId, token)
                session = s
                // Tizim pardasidagi "Stop sharing" — ustozning O'Z qarori.
                // Usiz keyingi qayta ulanish uni "uzilib qolgan ulashish" deb
                // tushunib, to'xtatilgan ulashishni qaytarib tiklardi.
                s.onUserStoppedShare = {
                    screenShare.reset()
                    log("ulashish tizim panelidan to'xtatildi")
                }
                observe(s)
                loadRoomState(s)
                loadChat(s)
                observeNetwork()

                runCatching { s.connect() }
                    .onSuccess {
                        log("LiveKit ulandi")
                        joinGuard.onAttached()
                        _state.update {
                            it.copy(
                                connecting = false,
                                roomName = token.roomName,
                                identity = token.identity,
                            )
                        }
                        // Host media: kamera + mikrofon (web useRoom.js bilan bir xil).
                        runCatching { s.setMicrophoneEnabled(true) }
                            .onSuccess {
                                _state.update { st -> st.copy(micOn = true).withAudioReason() }
                                log("mikrofon yoqildi")
                            }
                            .onFailure { log("mikrofon XATO: ${it.message}") }
                        if (withCamera) {
                            runCatching { s.setCameraEnabled(true) }
                                .onSuccess { _state.update { st -> st.copy(camOn = true) }; log("kamera yoqildi") }
                                .onFailure { log("kamera XATO: ${it.message}") }
                        } else {
                            log("kamera ruxsati yo'q — kamerasiz davom etamiz")
                        }
                    }
                    .onFailure { t ->
                        // Asl (inglizcha, WebRTC/LiveKit) matn diagnostika jurnalida qoladi,
                        // foydalanuvchi esa tushunarli o'zbekcha xabar ko'radi.
                        log("ULANISH XATO: ${t::class.simpleName}: ${t.message}")
                        _state.update {
                            it.copy(
                                connecting = false,
                                error = "Darsga ulanib bo'lmadi — internetni tekshirib qayta urinib ko'ring",
                            )
                        }
                        // KRITIK: sessiyani bo'shatamiz. Aks holda (a) qorovul qayta
                        // urinishni bloklaydi, (b) LiveKit Room xotirada osilib qoladi.
                        releaseSession()
                        LessonService.stop(ctx)
                    }
            } catch (c: CancellationException) {
                // ViewModel yopildi — bu xato emas, holatga tegmaymiz.
                releaseSession()
                throw c
            } catch (t: Throwable) {
                log("ULANISH ISTISNOSI: ${t::class.simpleName}: ${t.message}")
                _state.update {
                    it.copy(
                        connecting = false,
                        error = RoomErrors.startFailure(t),
                    )
                }
                releaseSession()
                runCatching { LessonService.stop(ctx) }
            }
        }
    }

    /** Sessiyani to'liq bo'shatadi va qorovulni qayta urinishga tayyorlaydi. */
    private fun releaseSession() {
        cancelObservers()
        LessonSessionHolder.stop()
        session = null
        joinGuard.onFailed()
    }

    /** UI "Qayta urinish" tugmasi uchun. */
    fun retryJoin(lessonId: String, withCamera: Boolean = true) {
        _state.update { it.copy(error = null, endedMessage = null) }
        join(lessonId, withCamera)
    }

    /**
     * ⭐ B-6 — DARS TUGADI, RESURSLAR BO'SHATILADI (maxfiylik).
     *
     * MUAMMO: xona serverda yopilganda (`POST /lessons/:id/end`, boshqa qurilmadan
     * yakunlash yoki aloqaning butunlay uzilishi) ilova faqat holat matnini
     * yangilardi. LiveKit `Room` uzilgan bo'lsa ham **MediaProjection ushlanib
     * turardi** — ya'ni dars tugagandan keyin ham telefon ekrani yozib olinardi.
     * Ustoz esa dars tugadi deb o'ylab shaxsiy ilovalarini ochardi.
     *
     * QOIDA: uzilish sababidan **qat'i nazar** proyeksiya va foreground servis
     * bo'shatiladi. "Ba'zi hollarda ushlab turish" — jimgina yozib olish degani.
     *
     * `stopScreenShare()` ataylab `release()` dan OLDIN va alohida chaqiriladi:
     * u `ScreenAudioCapturer` ni ham bo'shatadi va SDK'ning `ScreenCaptureService`
     * ini to'xtatadi; keyin `release()` `Room` ni yopadi.
     */
    private fun onDisconnected(sdkReasonName: String?) {
        val s = session ?: return // biz allaqachon o'zimiz chiqib bo'lganmiz

        // ⚠️ QAYTA ULANISH UCHUN ATAYLAB UZILGAN — dars TUGAMAGAN.
        //
        // Qurilma sinovida (2026-07-28, LTE) aniqlangan zanjir:
        //   tarmoq almashdi → majburiy qayta ulanish → `room.disconnect()`
        //   → `Disconnected(CLIENT_INITIATED)` → bu handler uni "ustoz chiqdi"
        //   deb tushunib MediaProjection va foreground servisni bo'shatardi
        //   → dars O'LARDI.
        // Ya'ni C-11 uchun yozilgan tuzatish o'zi darsni tugatardi.
        //
        // Shu bayroq ikkalasini ajratadi: bizning uzilishimiz jimgina o'tadi,
        // haqiqiy uzilish esa avvalgidek qayta ishlanadi.
        if (reconnects.intentional) {
            log("uzilish qayta ulanish uchun — dars davom etadi")
            return
        }
        val reason = RoomStatus.endReasonOf(sdkReasonName)
        val message = RoomStatus.endMessage(reason)
        log("dars tugadi: $reason")

        // Dars TUGADI — tiklaydigan ulashish yo'q (aks holda "Qayta boshlash"
        // dan keyin ustoz so'ramagan taklif chiqib qolardi).
        screenShare.reset()
        ShareFrameOverlay.hide()

        _state.update {
            it.copy(
                connecting = false,
                reconnecting = false,
                screenOn = false,
                restoreShare = false,
                screenAudioOn = false,
                micOn = false,
                camOn = false,
                endedMessage = message,
                // Xonani server yopgan (HOST_ENDED) — dars haqiqatan tugagan.
                // Tarmoq uzilishida esa dars serverda LIVE bo'lib qolaveradi —
                // bayroq saqlanadi va chiqishda "Yakunlaysizmi?" so'raladi.
                lessonLive = if (reason == RoomStatus.EndReason.HOST_ENDED ||
                    reason == RoomStatus.EndReason.REMOVED
                ) false else it.lessonLive,
            ).withAudioReason()
        }

        // Bo'shatish ALOHIDA korutinada: `releaseSession()` kuzatuvchi job'larni
        // bekor qiladi va biz hozir AYNAN o'sha job ichida turamiz — shu joyda
        // to'g'ridan-to'g'ri chaqirsak o'zimizni bekor qilib, tozalashni yarim
        // yo'lda qoldirardik.
        viewModelScope.launch {
            // Lokal yozuvni AVVAL to'xtatib yuklash navbatiga qo'yamiz (ekran
            // ulashish/projection bo'shatilishidan oldin fayl finalize bo'lsin).
            stopAndUploadLocalRecording(s)
            runCatching { s.stopScreenShare() }
                .onFailure { log("ulashishni to'xtatish XATO: ${it.message}") }
            releaseSession()
            LessonService.stop(appContext)
            log("MediaProjection va foreground servis bo'shatildi")
        }
    }

    /**
     * Tarmoq almashuvini kuzatadi (C-11).
     *
     * LiveKit SDK'sining o'z qayta ulanishi bor, lekin Android'da eski interfeys
     * DARHOL o'lmaydi: soket ochiq ko'rinadi, paket ketmaydi. SDK buni faqat
     * timeout orqali sezadi va ustoz shu vaqt jim ekranga qarab turadi.
     * Transport o'zgarishi esa "eski yo'l yaroqsiz" degan ANIQ signal.
     *
     * Qaror [NetworkSwitchPolicy] da — u sof va testlar bilan qotirilgan.
     */
    private fun observeNetwork() {
        observeJobs += reconnects.observeNetwork(_state) { session }
    }

    /**
     * Majburiy qayta ulanish. Sikl va uning sabablari `ReconnectController` da.
     */
    private fun forceReconnect() {
        val s = session ?: return
        reconnects.force(s, isCurrent = { session === it })
    }

    /**
     * Xonaga ulangach ko'tarilgan qo'llarni SERVERDAN yuklaydi.
     *
     * Data-channel faqat KELAJAKDAGI signallarni beradi. Ustoz darsga kech
     * qo'shilsa yoki ilovani qayta ishga tushirsa, u paytgacha ko'tarilgan
     * qo'llar unga hech qachon yetib bormasdi — ya'ni o'quvchi so'rab o'tirar,
     * ustoz esa hech nima ko'rmasdi.
     *
     * Xato jimgina yutiladi: holat yo'qligi darsni to'xtatmaydi (birinchi yangi
     * signal ro'yxatni baribir to'ldiradi).
     */
    private fun loadRoomState(s: LessonSession) {
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank()) return // eski backend — maydon yo'q
        viewModelScope.launch {
            rooms.state(lessonId, s.roomToken.token)
                .onSuccess { resp ->
                    val hands = resp?.hands.orEmpty().map {
                        RaisedHand(
                            identity = it.identity,
                            name = it.name.ifBlank { it.identity },
                            at = LessonFormat.instantMs(it.raisedAt),
                        )
                    }
                    if (hands.isNotEmpty()) {
                        _state.update { st -> st.copy(hands = HandQueue.replaceAll(hands)) }
                        notifySignals(vibrate = false)
                    }
                }
                .onFailure { log("xona holatini yuklab bo'lmadi: ${it.message}") }
        }
    }

    /**
     * Data-channel signalini holatga qo'llaydi va bildirishnomani yangilaydi.
     *
     * Bildirishnoma AYNAN shu yerda yangilanadi (UI qatlamida emas): ustoz ekran
     * ulashganda ilova fonda bo'ladi va Compose umuman render qilmaydi — signal
     * faqat shu yo'l bilan ko'zga tashlanadi.
     */
    private fun onRoomSignal(signal: RoomSignal?) {
        if (signal == null) return
        when (signal) {
            // Xabar o'chirildi (moderatsiya) — ID bo'yicha olib tashlanadi.
            // Qabrtosh QOLDIRILMAYDI: server ham xabar mazmunini yubormaydi
            // va o'chirilgan joyni belgilash buzg'unchiga e'tibor berardi.
            is RoomSignal.ChatDeleted -> {
                _state.update { st ->
                    val left = st.chat.filterNot { it.id == signal.id }
                    if (left.size == st.chat.size) st else st.copy(chat = left)
                }
                return
            }

            // Natija e'lon qilindi. Ustozning O'ZI e'lon qilgan bo'lsa panel
            // allaqachon yangilangan; bu tarmoq — boshqa qurilmadan (web'dan)
            // e'lon qilinsa mobil panel ham eskirib qolmasin.
            is RoomSignal.PollPublished -> {
                _state.update { it.copy(pollRevision = it.pollRevision + 1) }
                return
            }

            is RoomSignal.Chat -> {
                val myId = session?.roomToken?.identity
                val ui = ChatMessageUi(
                    id = signal.id,
                    name = signal.senderName,
                    body = signal.body,
                    self = signal.senderIdentity == myId,
                    toIdentity = signal.toIdentity,
                    file = signal.file?.let { ChatFileUi(it.name, it.size, it.url) },
                )
                _state.update { st ->
                    // ID bo'yicha dublikat kesiladi: o'z xabarimizni optimistik
                    // qo'shamiz, keyin server echo'si ham keladi.
                    if (st.chat.any { it.id == ui.id }) st
                    else st.copy(
                        chat = (st.chat + ui).takeLast(MAX_CHAT),
                        unreadChat = if (ui.self) st.unreadChat else st.unreadChat + 1,
                    )
                }
                return
            }

            is RoomSignal.Reaction -> {
                reactionSeq += 1
                val now = System.currentTimeMillis()
                val r = RoomReaction(reactionSeq, signal.emoji, signal.name, at = now)
                _state.update { it.copy(reactions = ReactionFeed.add(it.reactions, r)) }
                scheduleReactionPrune()
            }
            else -> {
                val before = _state.value.hands
                val after = HandQueue.apply(before, signal)
                if (after === before) return // hech nima o'zgarmadi — bildirishnomaga ham tegmaymiz
                _state.update { it.copy(hands = after) }
                // Vibratsiya faqat YANGI qo'lda: har o'zgarishda titratish (masalan
                // qo'l tushirilganda) ustozni bezovta qilardi.
                val newHand = after.size > before.size
                notifySignals(vibrate = newHand)
                return
            }
        }
        notifySignals(vibrate = false)
    }

    // ── Chat ────────────────────────────────────────────────────────────────

    /** Chat oynasi ochilganda o'qilmaganlar nolga tushadi. */
    fun markChatRead() {
        _state.update { it.copy(unreadChat = 0) }
    }

    /** Chat tarixini serverdan yuklaydi (xonaga kirganda va qayta ulanganda). */
    private fun loadChat(s: LessonSession) {
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank()) return
        val myId = s.roomToken.identity
        viewModelScope.launch {
            chatRepo.history(lessonId, s.roomToken.token)
                .onSuccess { items ->
                    // Server eng yangidan eskiga beradi — UI'da teskarisi kerak.
                    val ui = items.asReversed().map {
                        ChatMessageUi(
                            id = it.id,
                            name = it.senderName,
                            body = it.body,
                            self = it.senderIdentity == myId,
                            toIdentity = it.toIdentity,
                            file = it.file?.let { f -> ChatFileUi(f.name, f.size, f.url) },
                        )
                    }
                    _state.update { st -> st.copy(chat = ui.takeLast(MAX_CHAT)) }
                }
                .onFailure { log("chat tarixini yuklab bo'lmadi: ${it.message}") }
        }
    }

    /** Xabar yuboradi. `to` bo'sh bo'lsa — hammaga. */
    fun sendChat(body: String, to: String = "") {
        val s = session ?: return
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank() || body.isBlank()) return
        viewModelScope.launch {
            chatRepo.send(lessonId, s.roomToken.token, body, to).onSuccess { m ->
                if (m == null) return@onSuccess
                appendOwnMessage(m)
            }.onFailure {
                _state.update { st -> st.copy(error = ApiErrors.humanError(it)) }
            }
        }
    }

    /**
     * ⭐ FAYL ULASHISH (№15).
     *
     * [uri] — `ActivityResultContracts.GetContent` natijasi. Hajm/tur
     * tekshiruvi repozitoriyda, so'rov YUBORILMASDAN oldin: 20 MB'lik faylni
     * sekin mobil internetda yuklab bo'lib, keyin 400 olish ustoz uchun
     * bekorga ketgan bir necha daqiqa bo'lardi.
     *
     * Xato [notice] ga tushadi, [error] ga EMAS: bu dars ulanishining xatosi
     * emas va "Qayta urinish → xonaga qayta ulanish" ma'nosiz javob bo'lardi.
     */
    fun sendChatFile(uri: android.net.Uri, body: String = "", to: String = "") {
        val s = session ?: return
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank() || _state.value.chatUploading) return
        _state.update { it.copy(chatUploading = true) }
        viewModelScope.launch {
            chatRepo.upload(lessonId, uri, body, to)
                .onSuccess { m ->
                    _state.update { it.copy(chatUploading = false, notice = "Fayl yuborildi") }
                    if (m != null) appendOwnMessage(m)
                }
                .onFailure { t ->
                    log("fayl yuborilmadi: ${t.message}")
                    _state.update { it.copy(chatUploading = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * ⭐ XABARNI O'CHIRISH (moderatsiya, №6) — faqat dars egasi qila oladi.
     *
     * Ro'yxatdan olib tashlash `chat_deleted` hodisasi kelganda bo'ladi, lekin
     * bu yerda ham DARHOL olib tashlanadi: data-channel xabari kechikishi
     * mumkin va ustoz "o'chirdim, hali ham turibdi" holatini ko'rardi —
     * moderatsiyada esa aynan tezlik muhim. Server rad etsa xabar qaytadan
     * ko'rinmaydi (tarix qayta yuklanganda tiklanadi), lekin sabab ko'rsatiladi.
     */
    fun deleteChat(messageId: String) {
        val s = session ?: return
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank() || messageId.isBlank()) return
        _state.update { st -> st.copy(chat = st.chat.filterNot { it.id == messageId }) }
        viewModelScope.launch {
            chatRepo.delete(lessonId, messageId)
                .onSuccess { log("xabar o'chirildi: $messageId") }
                .onFailure { t ->
                    _state.update { it.copy(notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * Emoji reaksiya yuborish (№14).
     *
     * Optimistik ko'rsatish YO'Q: server xabarni bizga ham qaytaradi
     * (data-channel), ya'ni qo'shimcha lokal yozuv DUBLIKAT bo'lardi.
     */
    fun sendReaction(emoji: String) {
        val s = session ?: return
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank() || !Reactions.isAllowed(emoji)) return
        viewModelScope.launch {
            rooms.sendReaction(lessonId, s.roomToken.token, emoji)
                .onFailure { t ->
                    _state.update { it.copy(notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /** Snackbar ko'rsatildi — takror chiqmasin. */
    fun noticeShown() = _state.update { it.copy(notice = null) }

    /** O'z xabarimizni optimistik qo'shadi (server echo'si ID bo'yicha kesiladi). */
    private fun appendOwnMessage(m: uz.darsly.mentor.data.api.ChatMessageDto) {
        val ui = ChatMessageUi(
            id = m.id,
            name = m.senderName,
            body = m.body,
            self = true,
            toIdentity = m.toIdentity,
            file = m.file?.let { ChatFileUi(it.name, it.size, it.url) },
        )
        _state.update { st ->
            if (st.chat.any { it.id == ui.id }) st else st.copy(chat = (st.chat + ui).takeLast(MAX_CHAT))
        }
    }

    // ── Moderatsiya ─────────────────────────────────────────────────────────

    /**
     * Ishtirokchilar ro'yxatini serverdan yangilaydi.
     *
     * LiveKit'ning lokal ro'yxati (`room.remoteParticipants`) ham bor, lekin
     * moderatsiya uchun SERVER ko'rinishi ishlatiladi: u mute holatini ham
     * beradi va ustoz bosgan tugma natijasi bilan bir manbadan keladi.
     */
    fun refreshRoster() {
        val lessonId = session?.roomToken?.lessonId ?: return
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            rooms.participants(lessonId)
                .onSuccess { items ->
                    val raised = _state.value.hands.map { it.identity }.toSet()
                    val me = session?.roomToken?.identity
                    _state.update { st ->
                        st.copy(
                            roster = items
                                .filter { it.identity != me } // ustozning o'zi ro'yxatda kerak emas
                                .map {
                                    RosterEntry(
                                        identity = it.identity,
                                        name = it.name.ifBlank { it.identity },
                                        audioMuted = it.audioMuted,
                                        handRaised = it.identity in raised,
                                    )
                                },
                        )
                    }
                }
                .onFailure { log("ishtirokchilarni yuklab bo'lmadi: ${it.message}") }
        }
    }

    /**
     * Moderatsiya amali. Har biridan keyin ro'yxat yangilanadi — ustoz natijani
     * darhol ko'rsin (aks holda "bosdim, hech nima o'zgarmadi" hissi qoladi).
     */
    private fun moderate(action: suspend (String) -> Unit) {
        val lessonId = session?.roomToken?.lessonId ?: return
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            runCatching { action(lessonId) }
                .onSuccess { refreshRoster() }
                .onFailure { _state.update { st -> st.copy(error = ApiErrors.humanError(it)) } }
        }
    }

    /** [allowSelfUnmute] false — «o'zi ocholmasin» ham yoqiladi (Zoom andozasi). */
    fun muteAll(allowSelfUnmute: Boolean? = null) = moderate { moderation.muteAll(it, allowSelfUnmute) }
    fun muteParticipant(identity: String) = moderate { moderation.mute(it, identity) }
    /** [permanent] true — mentorning barcha darslaridan doimiy qora ro'yxat. */
    fun removeParticipant(identity: String, permanent: Boolean = false) =
        moderate { moderation.remove(it, identity, if (permanent) "mentor" else "lesson") }
    fun allowSpeak(identity: String) = moderate { moderation.allowSpeak(it, identity) }
    fun revokeSpeak(identity: String) = moderate { moderation.revokeSpeak(it, identity) }

    fun lowerHand(identity: String) = moderate { moderation.lowerHand(it, identity) }
    fun lowerAllHands() = moderate { moderation.lowerAllHands(it) }

    /**
     * Muddati o'tgan reaksiyalarni ekrandan olib tashlaydi.
     *
     * Yagona ish rejalashtiriladi ([reactionPruneJob]): xonada 10 kishi bir
     * vaqtda reaksiya bossa 10 ta bir xil kutish korutinasi ochilardi.
     * Qaror va TTL sof [ReactionFeed] da — bu yerda faqat vaqt o'tkazish.
     */
    private fun scheduleReactionPrune() {
        if (reactionPruneJob?.isActive == true) return
        reactionPruneJob = viewModelScope.launch {
            while (_state.value.reactions.any { it.at > 0L }) {
                delay(ReactionFeed.TTL_MS / 2)
                val now = System.currentTimeMillis()
                _state.update { st ->
                    val left = ReactionFeed.prune(st.reactions, now)
                    if (left === st.reactions) st else st.copy(reactions = left)
                }
            }
        }
    }

    /** Foreground bildirishnoma matnini joriy signallar bilan yangilaydi. */
    private fun notifySignals(vibrate: Boolean) {
        val st = _state.value
        LessonNotifications.updateSignals(
            ctx = appContext,
            hands = st.hands.size,
            lastReaction = st.reactions.firstOrNull()?.let { "${it.emoji} ${it.name}".trim() },
            vibrate = vibrate,
        )
    }

    /**
     * Sessiya oqimlarini kuzatish. Job'lar saqlanadi va sessiya bo'shatilganda
     * bekor qilinadi — aks holda qayta urinishlarda bo'shatilgan `Room` ustidagi
     * eski kollektorlar to'planib qolardi.
     */
    private fun observe(s: LessonSession) {
        observeJobs += viewModelScope.launch {
            s.room.events.collect { event ->
                when (event) {
                    is RoomEvent.Connected -> {
                        _state.update { it.copy(reconnecting = false) }
                        setConn(s.room)
                    }
                    // M16: qayta ulanish JIM bo'lmasligi kerak — ustoz "dars buzildi"
                    // deb o'ylab telefonni qayta ishga tushirmasligi uchun ko'rsatamiz.
                    is RoomEvent.Reconnecting -> {
                        log("qayta ulanmoqda")
                        _state.update { it.copy(reconnecting = true) }
                        setConn(s.room)
                    }
                    is RoomEvent.Reconnected -> {
                        log("qayta ulandi")
                        _state.update { it.copy(reconnecting = false) }
                        setConn(s.room)
                        // SDK o'z tiklanishida ekran trekini saqlab qolishi ham,
                        // yo'qotishi ham mumkin — qaror faktga qaraydi, shuning
                        // uchun bu yerdan chaqirish xavfsiz (ketayotgan ulashishga
                        // tegilmaydi).
                        restoreScreenShare()
                    }
                    is RoomEvent.Disconnected -> {
                        log("uzildi: ${event.reason}")
                        setConn(s.room)
                        onDisconnected(event.reason?.name)
                    }
                    // M16: aloqa sifati — FAQAT o'zimizning ko'rsatkichimiz
                    // (o'quvchining yomon interneti ustozning indikatorini qizartirmasin).
                    is RoomEvent.ConnectionQualityChanged -> {
                        if (event.participant == s.room.localParticipant) {
                            _state.update {
                                it.copy(linkQuality = RoomStatus.qualityOf(event.quality.name))
                            }
                        }
                    }
                    is RoomEvent.ParticipantConnected,
                    is RoomEvent.ParticipantDisconnected,
                    -> setConn(s.room)
                    is RoomEvent.TrackPublished -> {
                        log("track e'lon qilindi: ${event.publication.source}")
                        refreshParticipants(s.room)
                    }
                    // M20: video plitkalari aynan shu hodisalarda paydo bo'ladi/yo'qoladi.
                    is RoomEvent.TrackSubscribed,
                    is RoomEvent.TrackUnsubscribed,
                    is RoomEvent.TrackMuted,
                    is RoomEvent.TrackUnmuted,
                    -> refreshParticipants(s.room)
                    // DIQQAT: TrackPublicationFailed'da exception `val` emas (SDK 2.27.0),
                    // shuning uchun faqat track nomini log qilamiz.
                    is RoomEvent.TrackPublicationFailed -> log("TRACK E'LON XATOSI: ${event.track.name}")
                    // ⭐ Dars ichidagi signallar: qo'l ko'tarish, reaksiya, chat.
                    // Manba serverda (`roomstate`/`chat`), bu yerda faqat qo'llaymiz.
                    //
                    // C-1: `event.participant` — LiveKit tasdiqlagan yuboruvchi. U
                    // `null` bo'lsa xabar SERVERDAN kelgan (backend `SendData`);
                    // aks holda uni xonadagi biror klient yuborgan. Biz qabul
                    // qiladigan uch turning (hand/reaction/chat) hammasi faqat
                    // serverdan keladi, shuning uchun klient publish'lari
                    // BUTUNLAY rad etiladi. Busiz istalgan o'quvchi
                    // `publishData({sender_name:"Ustoz Ali", body:"..."})` yuborib
                    // ustozning o'z ekranida soxta xabar chiqara olardi.
                    is RoomEvent.DataReceived ->
                        if (event.participant == null) onRoomSignal(dataParser.parse(event.data))
                        else log("DATA: klient publish'i rad etildi (${event.participant?.identity?.value})")
                    else -> Unit
                }
            }
        }
        observeJobs += viewModelScope.launch {
            s.screenShareOn.collect { on ->
                _state.update { it.copy(screenOn = on).withAudioReason() }
                // Efir-ramka: ulashish ketayotganda ekran chetida rangli hoshiya —
                // ustoz BOSHQA ilovaga o'tsa ham efir/yozuv ketayotganini ko'radi.
                // Yozuv yoniq bo'lsa qizil, aks holda mint (ShareFrameOverlay izohi).
                if (on) {
                    ShareFrameOverlay.show(
                        appContext,
                        ShareFrameOverlay.colorFor(_state.value.recordingEnabled),
                    )
                    onShareStarted()
                } else {
                    ShareFrameOverlay.hide()
                }
            }
        }
        observeJobs += viewModelScope.launch {
            s.screenAudioOn.collect { on ->
                _state.update { it.copy(screenAudioOn = on).withAudioReason() }
            }
        }
        observeJobs += viewModelScope.launch {
            s.cameraFront.collect { front -> _state.update { it.copy(cameraFront = front) } }
        }
    }

    private fun cancelObservers() {
        observeJobs.forEach { it.cancel() }
        observeJobs.clear()
        // Qayta ulanish sikli ham to'xtaydi. Avval u faqat KEYINGI urinishda
        // (`isCurrent` tekshiruvida) to'xtardi — ya'ni ustoz chiqib ketgandan
        // keyin ham bir necha soniya fon urinishlari davom etardi.
        reconnects.cancel()
    }

    private fun setConn(room: Room) {
        _state.update {
            it.copy(
                connState = room.state.name.lowercase(),
                participantCount = room.remoteParticipants.size + 1,
            )
        }
        refreshParticipants(room)
    }

    /**
     * O'quvchilar ro'yxatini LiveKit holatidan qayta yig'adi (M20).
     *
     * Nega har hodisada TO'LIQ qayta yig'iladi (inkremental emas): xona 25-30 kishilik,
     * ro'yxat kichik, va LiveKit holati yagona haqiqat manbai. Inkremental yangilash
     * "qo'shildi/chiqdi/trek keldi/trek ketdi" hodisalarining har birini to'g'ri
     * ushlashni talab qiladi — bitta o'tkazib yuborilgan hodisa ekranda **qotib qolgan
     * plitka** qoldiradi va buni foydalanuvchi darhol ko'radi.
     */
    private fun refreshParticipants(room: Room) {
        val list = room.remoteParticipants.values.map { p ->
            val cameraTrack = p.videoTrackPublications
                .firstOrNull { (pub, _) -> pub.source == Track.Source.CAMERA }
                ?.second as? VideoTrack
            ParticipantUi(
                identity = p.identity?.value.orEmpty(),
                name = p.name?.takeIf { it.isNotBlank() } ?: "O'quvchi",
                videoTrack = cameraTrack,
                micMuted = !p.isMicrophoneEnabled,
                speaking = p.isSpeaking,
            )
        }.sortedBy { it.name }

        val local = room.localParticipant.videoTrackPublications
            .firstOrNull { (pub, _) -> pub.source == Track.Source.CAMERA }
            ?.second as? VideoTrack

        _state.update { it.copy(participants = list, localVideo = local) }
    }

    fun toggleMic() = viewModelScope.launch {
        val s = session ?: return@launch
        val target = !_state.value.micOn
        runCatching { s.setMicrophoneEnabled(target) }
            .onSuccess { _state.update { it.copy(micOn = target).withAudioReason() } }
            .onFailure { log("mikrofon XATO: ${it.message}") }
    }

    fun toggleCam() = viewModelScope.launch {
        val s = session ?: return@launch
        val target = !_state.value.camOn
        runCatching { s.setCameraEnabled(target) }
            .onSuccess { _state.update { it.copy(camOn = target) } }
            .onFailure { log("kamera XATO: ${it.message}") }
    }

    /** M12: old ↔ orqa. Kamera o'chiq bo'lsa hech narsa qilinmaydi (UI ham bloklaydi). */
    fun flipCamera() {
        val s = session ?: return
        if (s.flipCamera()) {
            log(if (s.cameraFront.value) "old kameraga o'tildi" else "orqa kameraga o'tildi")
        } else {
            log("kamera o'chiq — almashtirish mumkin emas")
        }
    }

    /**
     * ⭐ EKRAN ULASHISHNI BOSHLASH.
     *
     * [resultData] — `MediaProjectionManager.createScreenCaptureIntent()` dan qaytgan
     * `Activity.RESULT_OK` natijasi.
     *
     * TARTIB (Android 14+ uchun majburiy):
     *  1. `LessonService`ni `mediaProjection` tipi bilan QAYTA ishga tushiramiz —
     *     endi proyeksiya ruxsati bor, shuning uchun tip qabul qilinadi.
     *  2. Keyin `setScreenShareEnabled(true, ScreenCaptureParams(...))`.
     *     SDK ichida ham FGS avval, `startCapture()` keyin (LocalParticipant.kt:389-390).
     */
    fun startScreenShare(resultData: Intent) = viewModelScope.launch {
        val s = session ?: run { log("ekran: sessiya yo'q"); return@launch }
        screenShare.start(s, resultData)
        maybeStartLocalRecording(s)
        notifySignals(vibrate = false)
    }

    fun stopScreenShare() = viewModelScope.launch {
        val s = session ?: return@launch
        stopAndUploadLocalRecording(s)
        screenShare.stop(s)
        notifySignals(vibrate = false)
    }

    // ─── Client-side (lokal) yozuv orkestratsiyasi ─────────────────────────────

    /**
     * AVTO-yozuv: dars sozlamasi (`is_recording_enabled`) yoniq bo'lsa, ekran
     * ulashilganда yozuvni o'zi boshlaydi. O'chirilgan bo'lsa hech narsa qilmaydi —
     * ustoz xohlasa Record tugmasi bilan [toggleRecording] orqali qo'lда boshlaydi.
     */
    private fun maybeStartLocalRecording(s: LessonSession) {
        if (!_state.value.recordingEnabled) return
        beginLocalRecording(s)
    }

    /**
     * ⭐ RECORD tugmasi (Zoom kabi) — yozuvni QO'LDA boshlash/to'xtatish.
     *
     * Yozuv EKRANni yozadi, shuning uchun ekran ulashilmagan bo'lsa boshlab
     * bo'lmaydi — ustozga hint ko'rsatiladi (avval ulashsin).
     */
    fun toggleRecording() {
        val s = session ?: return
        when {
            s.isLocalRecording() -> stopAndUploadLocalRecording(s)
            _state.value.screenOn -> beginLocalRecording(s)
            else -> _state.update {
                it.copy(recordHint = "Yozib olish ekranni yozadi — avval ekran ulashing")
            }
        }
    }

    /** UI hint'ni ko'rsatdi — tozalanadi. */
    fun recordHintShown() = _state.update { it.copy(recordHint = null) }

    /**
     * Lokal yozuvni boshlaydi (avto yoki qo'lda). Server RECORDING_MODE=local
     * bo'lsa egress boshlanmaydi — yozuv shu telefonda. Ekran treki hali
     * chiqmagan bo'lishi mumkin, shuning uchun qisqa retry.
     */
    private fun beginLocalRecording(s: LessonSession) {
        if (s.isLocalRecording()) return
        val lessonId = s.roomToken.lessonId
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            val id = localRec.localStart(lessonId).getOrElse {
                log("lokal yozuv: local-start XATO: ${it.message}")
                _state.update { it.copy(recordHint = "Yozib olishni boshlab bo'lmadi") }
                return@launch
            }
            // Ekran treki tayyor bo'lishini kutamiz (3 urinish).
            repeat(3) { attempt ->
                val file = s.startLocalRecording(lessonId, LOCAL_REC_BITRATE)
                if (file != null) {
                    localRecId = id
                    localRecStartMs = System.currentTimeMillis()
                    _state.update { it.copy(isRecording = true) }
                    log("lokal yozuv boshlandi: ${file.name}")
                    return@launch
                }
                delay(400)
            }
            log("lokal yozuv qo'llab-quvvatlanmadi (ekran treki chiqmadi)")
            _state.update { it.copy(recordHint = "Yozib olishni boshlab bo'lmadi") }
        }
    }

    /** Lokal yozuvni to'xtatadi va app-scoped korutinada serverga yuklaydi. */
    @OptIn(kotlinx.coroutines.DelicateCoroutinesApi::class)
    private fun stopAndUploadLocalRecording(s: LessonSession) {
        if (!s.isLocalRecording()) return
        _state.update { it.copy(isRecording = false) }
        val id = localRecId
        val startMs = localRecStartMs
        localRecId = null
        val file = s.stopLocalRecording()
        if (id == null || file == null) {
            file?.delete()
            return
        }
        val durationSec = ((System.currentTimeMillis() - startMs) / 1000).toInt().coerceAtLeast(1)
        val endedAt = java.time.Instant.now().toString()
        // App-scoped: xonadan chiqilgach ham yuklash davom etsin (MVP; keyinroq
        // WorkManager bilan qattiqlashtiriladi — ilova o'ldirilsa omon qolsin).
        GlobalScope.launch(Dispatchers.IO) {
            localRec.upload(id, file, durationSec, endedAt)
                .onSuccess { file.delete() }
                .onFailure { android.util.Log.e("RoomVM", "lokal yozuv yuklash XATO", it) }
        }
    }

    /**
     * ⭐ ULASHISH HAQIQATAN BOSHLANDI — Zoom kabi ilovani fonga olamiz (№25).
     *
     * Signal `screenShareOn` OQIMIDAN olinadi, `startScreenShare()` dan emas:
     * ustoz tizim dialogida "Boshlash" ni bosgani hali ulashish boshlangani
     * emas (SDK `SecurityException` bilan yiqilishi mumkin). Ilovani fonga
     * olish esa faqat efir haqiqatan ketayotganda to'g'ri.
     *
     * Qaror [ShareBackgroundPlan] da (sof, JVM testida qoplangan) — bu yerda
     * faqat kirish qiymatlarini yig'ish va natijani UI'ga uzatish.
     */
    private fun onShareStarted() {
        val decision = ShareBackgroundPlan.onShareStarted(
            sharing = true,
            alreadyHandled = shareBackgroundHandled,
            autoBackground = uiPrefs.autoBackgroundOnShare,
            cameraOn = _state.value.camOn,
        )
        if (decision.isEmpty) return
        shareBackgroundHandled = true
        log(if (decision.moveToBack) "ulashish boshlandi — ilova fonga olinadi" else "ulashish boshlandi")
        _state.update { it.copy(shareBackground = decision) }
    }

    /** UI xabarni ko'rsatdi va (kerak bo'lsa) ilovani fonga oldi — hodisa tozalanadi. */
    fun shareBackgroundShown() {
        _state.update { it.copy(shareBackground = null) }
    }

    /** Ustoz "Keyinroq" dedi — taklif yopiladi, lekin niyat saqlanadi. */
    fun dismissRestoreShare() = screenShare.dismissRestorePrompt()

    /**
     * Uzilishdan keyin ekran ulashishni tiklash (C-11 · 3-gipoteza).
     *
     * Qaror va uning sabablari `ScreenShareController` da; bu yerda faqat
     * korutina qamrovi beriladi (controller `suspend` funksiyani o'zi
     * ishga tushira olmaydi).
     */
    private fun restoreScreenShare() {
        val s = session ?: return
        val consent = screenShare.planAfterReconnect(s) ?: return
        viewModelScope.launch { screenShare.start(s, consent) }
    }

    /**
     * ⭐ DARSNI YAKUNLASH — xona serverda yopiladi va status `ended` bo'ladi.
     *
     * Chiqishdan farqi: [leave] faqat shu qurilmani xonadan oladi, dars esa
     * `live` bo'lib qolaveradi. Mobil ilovada yakunlash yo'li YO'Q edi — shu
     * sabab darslar abadiy "Jonli" bo'lib turardi.
     *
     * Server xatosi bo'lsa ham mahalliy resurslar baribir bo'shatiladi: ustoz
     * ekranda qulflanib qolmasligi kerak (dars statusini keyin ro'yxatdan yoki
     * web'dan tuzatish mumkin).
     */
    fun endLesson(lessonId: String) {
        viewModelScope.launch {
            rooms.endLesson(lessonId)
                .onSuccess { log("dars yakunlandi (server)") }
                .onFailure { log("yakunlash XATO: ${it.message}") }
            leave()
        }
    }

    /** Darsni tark etish (xonani yopmaydi — bu `POST /lessons/:id/end` ishi). */
    fun leave() {
        ShareFrameOverlay.hide()
        // Lokal yozuvni AVVAL finalize qilib yuklash navbatiga qo'yamiz — aks holda
        // `cancelObservers()` disconnect kuzatuvchisini o'chirib, `session=null` esa
        // `onDisconnected()` dagi to'xtatishni ham chetlab o'tardi → fayl `moov`siz
        // yaroqsiz qolardi (Yakunlash yo'li yozuvni orfan qilib qo'yardi).
        session?.let { stopAndUploadLocalRecording(it) }
        cancelObservers()
        LessonSessionHolder.stop()
        session = null
        screenShare.reset()
        shareBackgroundHandled = false
        joinGuard.onReleased()
        LessonService.stop(appContext)
        _state.value = RoomUiState()
    }
}
