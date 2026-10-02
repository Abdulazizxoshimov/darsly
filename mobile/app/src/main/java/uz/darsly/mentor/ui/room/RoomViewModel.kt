package uz.darsly.mentor.ui.room

import android.content.Intent
import android.os.Build
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import io.livekit.android.events.RoomEvent
import io.livekit.android.room.Room
import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.Track
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.livekit.HandQueue
import uz.darsly.mentor.data.livekit.LessonSession
import uz.darsly.mentor.data.livekit.LessonSessionStore
import uz.darsly.mentor.data.livekit.RaisedHand
import uz.darsly.mentor.data.livekit.ReactionFeed
import uz.darsly.mentor.data.livekit.RoomDataParser
import uz.darsly.mentor.data.livekit.RoomReaction
import uz.darsly.mentor.data.livekit.RoomSignal
import uz.darsly.mentor.data.livekit.ScreenAudioPolicy
import uz.darsly.mentor.data.livekit.ShareBackgroundPlan
import uz.darsly.mentor.data.livekit.TransportSource
import uz.darsly.mentor.data.repo.LessonsRepository
import uz.darsly.mentor.data.repo.LocalRecordingRepository
import uz.darsly.mentor.data.repo.ModerationRepository
import uz.darsly.mentor.data.repo.RoomChatRepository
import uz.darsly.mentor.data.repo.RoomRepository
import uz.darsly.mentor.data.store.UiPrefs
import uz.darsly.mentor.di.ApplicationScope
import uz.darsly.mentor.service.LessonPlatform
import uz.darsly.mentor.util.LessonFormat
import java.io.File
import javax.inject.Inject

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
    val connState: ConnState = ConnState.DISCONNECTED,
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
     * ham chiqariladi (`LessonPlatform.updateSignals`). Aks holda darsning eng
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
    /** №25 sozlamasi — «Ko'proq» panelida o'zgartiriladi; manba [UiPrefs] (M9). */
    val autoBackgroundOnShare: Boolean = true,
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
        get() = connecting || connState.inRoom || screenOn || lessonLive

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

/** Lokal yozuv video bitreyti (bps) — VBR TARGETI ([uz.darsly.mentor.data.livekit.LocalRecorder] VBR rejimida).
 *  Ekran/slayd kontenti yaxshi siqiladi; 1.5 Mbps VBR Zoom-daraja ravshanlik beradi,
 *  statik kadrda bitni isrof qilmaydi. Yakuniy hajmni dars tugagach post-siqish
 *  (telefonda H.264 CRF) hal qiladi — bu faqat jonli oraliq faylni yengillashtiradi. */
private const val LOCAL_REC_BITRATE = 1_500_000

/**
 * Serverda ochilgan lokal yozuv qatori: `local-start` bergan ID va boshlanish vaqti.
 * Yuklashda davomiylik shundan hisoblanadi.
 */
private data class ActiveRecording(val id: String, val startedAtMs: Long)

/**
 * Xona ekranining ViewModel'i.
 *
 * ## Egalik (M2)
 * Sessiyaning egasi [LessonSessionStore]; bu sinf uni FAQAT store orqali oladi va
 * bo'shatadi. Bo'shatish ilova qamrovida ([appScope]) bajariladi — ekran yopilib
 * `viewModelScope` bekor qilinsa ham tugaydi (C1). Android tizimi bilan barcha
 * muloqot [LessonPlatform] orqali — shu tufayli bu sinf JVM testida yaratiladi.
 */
@HiltViewModel
class RoomViewModel @Inject constructor(
    private val rooms: RoomRepository,
    private val moderation: ModerationRepository,
    private val chatRepo: RoomChatRepository,
    private val lessons: LessonsRepository,
    private val localRec: LocalRecordingRepository,
    private val uiPrefs: UiPrefs,
    private val sessions: LessonSessionStore,
    private val platform: LessonPlatform,
    transports: TransportSource,
    @ApplicationScope private val appScope: CoroutineScope,
) : ViewModel() {

    private val _state = MutableStateFlow(RoomUiState(autoBackgroundOnShare = uiPrefs.autoBackgroundOnShare))
    val state: StateFlow<RoomUiState> = _state.asStateFlow()

    /** Joriy sessiya — UI sahnasi (`room`) va so'rovnoma paneli (room-token) uchun. */
    val session: StateFlow<LessonSession?> = sessions.session

    /** Bu ekran EGALIK QILAYOTGAN sessiya (store'dagi bilan bir xil bo'lishi tekshiriladi). */
    private var current: LessonSession? = null

    /** Serverda ochilgan lokal yozuv qatori (yozuv ketayotganda). */
    private var activeRecording: ActiveRecording? = null

    /**
     * Ustoz yozuvni XOHLAYDIMI — avto-sozlama yoki oxirgi qo'lda tanlov (H5).
     * Ekran treki qaytganda yozuv shu niyat bo'yicha qayta boshlanadi.
     */
    private var recordWanted = false

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
    private val reconnects = ReconnectController(
        transports = transports,
        scope = viewModelScope,
        onState = { transform -> _state.update(transform) },
        onLog = { line -> log(line) },
        onReconnected = { restoreScreenShare() },
    )

    /**
     * Ekran ulashish — alohida controller'da.
     *
     * Bu blok bu yerdagi eng murakkabi edi: o'z holati (niyat + saqlangan
     * rozilik), Android 14+ ning maxsus qoidalari va uch tarmoqli tiklash
     * mantiqi. Ajratilgach u LiveKit'siz sinaladi.
     */
    private val screenShare = ScreenShareController(
        platform = platform,
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

    // ── UI sozlamalari (M9: Compose `PrefsUiPrefs.create(ctx)` bilan Hilt'ni chetlab o'tardi) ──

    /** "Butun ekran" tushuntirishi hali ko'rsatilsinmi. */
    val shareTipEnabled: Boolean get() = uiPrefs.screenShareTipEnabled

    /** Ustoz "Boshqa eslatilmasin" dedi. */
    fun muteShareTip() {
        uiPrefs.screenShareTipEnabled = false
    }

    fun setAutoBackgroundOnShare(on: Boolean) {
        uiPrefs.autoBackgroundOnShare = on
        _state.update { it.copy(autoBackgroundOnShare = on) }
    }

    /**
     * `POST /lessons/:id/token` → host RoomToken → LiveKit'ga ulanish.
     * Sessiya [LessonSessionStore] da yashaydi (Activity qayta yaratilsa uzilmaydi).
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
            // KUTILMAGAN ISTISNO QOROVULI (🟡A).
            //
            // `platform.startService` (Android 12+ da fon rejimidan chaqirilsa
            // `ForegroundServiceStartNotAllowedException`) va `sessions.start`
            // otilsa korutina jim o'lardi va [joinGuard] CONNECTING da qotib qolardi:
            // ekranda **abadiy spinner**, `error == null` bo'lgani uchun "Qayta urinish"
            // tugmasi ham chiqmasdi — ya'ni tuzatilgan xatodan yomonroq holat.
            //
            // `CancellationException` qayta otiladi: u xato emas, ViewModel yopilishi
            // (yoki `viewModelScope` bekor qilinishi) signali.
            var s: LessonSession? = null
            try {
                val token = rooms.hostToken(lessonId)
                    .getOrElse { t ->
                        _state.update { it.copy(connecting = false, error = ApiErrors.humanError(t)) }
                        joinGuard.onFailed()
                        return@launch
                    }
                // `RoomRepository.hostToken` bo'sh javobni xatoga aylantiradi,
                // shuning uchun bu yerda `token` non-null.
                log("token OK · room=${token.roomName} · ws=${token.wsUrl}")
                // Host token berildi = server darsni `live` ga o'tkazdi. Bundan
                // keyin chiqish faqat "Yakunlash / Vaqtincha chiqish" savoli bilan.
                _state.update { it.copy(lessonLive = true) }

                // Sarlavha uchun dars nomi (UUID emas). Alohida, muvaffaqiyatsizlikka
                // chidamli so'rov: nom kelmasa ham dars boshlanaverishi kerak.
                lessons.byId(lessonId)
                    ?.let { lesson ->
                        recordWanted = lesson.isRecordingEnabled
                        _state.update {
                            it.copy(
                                lessonTitle = lesson.title,
                                recordingEnabled = lesson.isRecordingEnabled,
                            )
                        }
                    }

                // TARTIB: avval sessiya (store eskisini bo'shatib SERVISNI
                // to'xtatadi), keyin foreground servis — aks holda endigina
                // boshlangan servis eskisining `stopService` i bilan o'lardi.
                val session = sessions.start(lessonId, token)
                s = session
                current = session
                // Dars boshlanishi bilan foreground servis: mikrofon/kamera fon rejimida
                // o'lmasin (ekran ulashish tipi keyinroq, ruxsat olingandan so'ng qo'shiladi).
                platform.startService(withProjection = false)

                // Tizim pardasidagi "Stop sharing" — ustozning O'Z qarori.
                // Usiz keyingi qayta ulanish uni "uzilib qolgan ulashish" deb
                // tushunib, to'xtatilgan ulashishni qaytarib tiklardi.
                session.onUserStoppedShare = {
                    screenShare.reset()
                    log("ulashish tizim panelidan to'xtatildi")
                }
                observe(session)
                loadRoomState(session)
                loadChat(session)
                observeJobs += reconnects.observeNetwork(_state) { current }

                runCatching { session.connect() }
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
                        // Kirganda kamera va mikrofon O'CHIQ (mahsulot qarori 2026-08-15).
                        //
                        // Avval ikkalasi ham avtomatik yonardi (web useRoom.js bilan bir xil),
                        // lekin bu noqulay: mentor xonaga kirgan zahoti kutilmaganda
                        // ko'rinib/eshitilib qolardi (tayyor bo'lmagan holatda). Endi ikkalasi
                        // o'chiq turadi va mentor tayyor bo'lgach ControlBar'dan o'zi yoqadi.
                        log("kirildi — mic/kamera o'chiq (mentor o'zi yoqadi), kamera ruxsati=$withCamera")
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
                        releaseSession(session)
                    }
            } catch (c: CancellationException) {
                // ViewModel yopildi — bu xato emas, holatga tegmaymiz.
                s?.let { releaseSession(it) }
                throw c
            } catch (t: Throwable) {
                log("ULANISH ISTISNOSI: ${t::class.simpleName}: ${t.message}")
                _state.update {
                    it.copy(
                        connecting = false,
                        error = RoomErrors.startFailure(t),
                    )
                }
                s?.let { releaseSession(it) } ?: joinGuard.onFailed()
            }
        }
    }

    /**
     * Sessiyani to'liq bo'shatadi va qorovulni qayta urinishga tayyorlaydi.
     * Bo'shatishning o'zi store'da, ilova qamrovida — bu korutina bekor qilinsa ham tugaydi.
     */
    private fun releaseSession(s: LessonSession) {
        cancelObservers()
        if (current === s) current = null
        sessions.releaseAsync(s)
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
     */
    private fun onDisconnected(s: LessonSession, sdkReasonName: String?) {
        // M6: bu hodisa ESKI sessiyadan kechikib kelgan bo'lishi mumkin (tez
        // "Qayta boshlash"); yangi sessiyaga tegmaymiz.
        if (current !== s) return

        // ⚠️ QAYTA ULANISH UCHUN ATAYLAB UZILGAN — dars TUGAMAGAN.
        //
        // Qurilma sinovida (2026-07-28, LTE) aniqlangan zanjir:
        //   tarmoq almashdi → majburiy qayta ulanish → `room.disconnect()`
        //   → `Disconnected(CLIENT_INITIATED)` → bu handler uni "ustoz chiqdi"
        //   deb tushunib MediaProjection va foreground servisni bo'shatardi
        //   → dars O'LARDI.
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

        // Bo'shatish ALOHIDA korutinada: `cancelObservers()` kuzatuvchi job'larni
        // bekor qiladi va biz hozir AYNAN o'sha job ichida turamiz.
        val recording = takeActiveRecording()
        cancelObservers()
        current = null
        joinGuard.onFailed()
        appScope.launch {
            // Lokal yozuvni AVVAL to'xtatib yuklash navbatiga qo'yamiz (ekran
            // ulashish/projection bo'shatilishidan oldin fayl finalize bo'lsin).
            stopAndUploadLocalRecording(s, recording)
            runCatching { s.stopScreenShare() }
                .onFailure { log("ulashishni to'xtatish XATO: ${it.message}") }
            sessions.release(s)
            log("MediaProjection va foreground servis bo'shatildi")
        }
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
                    val left = ChatLog.delete(st.chat, signal.id)
                    if (left === st.chat) st else st.copy(chat = left)
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
                val myId = current?.roomToken?.identity
                val ui = ChatMessageUi(
                    id = signal.id,
                    name = signal.senderName,
                    body = signal.body,
                    self = signal.senderIdentity == myId,
                    toIdentity = signal.toIdentity,
                    file = signal.file?.let { ChatFileUi(it.name, it.size, it.url) },
                )
                _state.update { st ->
                    // ID bo'yicha dublikat kesish + o'qilmagan sanog'i — sof [ChatLog] da.
                    val res = ChatLog.add(st.chat, ui)
                    if (res.chat === st.chat) st
                    else st.copy(chat = res.chat, unreadChat = st.unreadChat + res.unreadDelta)
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
        val s = current ?: return
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
        val s = current ?: return
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
        val s = current ?: return
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
        val s = current ?: return
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
            val res = ChatLog.add(st.chat, ui)
            if (res.chat === st.chat) st else st.copy(chat = res.chat)
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
        val lessonId = current?.roomToken?.lessonId ?: return
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            rooms.participants(lessonId)
                .onSuccess { items ->
                    val raised = _state.value.hands.map { it.identity }.toSet()
                    val me = current?.roomToken?.identity
                    _state.update { st ->
                        // Ustozni filtrlash + qo'l belgisi — sof [RosterBuilder] da.
                        st.copy(
                            roster = RosterBuilder.build(
                                items = items,
                                me = me,
                                raised = raised,
                                identity = { it.identity },
                                name = { it.name },
                                audioMuted = { it.audioMuted },
                            ),
                        )
                    }
                }
                .onFailure { log("ishtirokchilarni yuklab bo'lmadi: ${it.message}") }
        }
    }

    /**
     * Moderatsiya amali. Har biridan keyin ro'yxat yangilanadi — ustoz natijani
     * darhol ko'rsin (aks holda "bosdim, hech nima o'zgarmadi" hissi qoladi).
     *
     * Xato [notice] ga (H2): repozitoriy endi 403/404/500 ni haqiqatan xato deb
     * qaytaradi va ustoz "mute qildim" deb yanglishmaydi. `error` emas —
     * u "Ulanmadi + Qayta urinish" kartasini chiqarardi.
     */
    private fun moderate(action: suspend (String) -> Result<Unit>) {
        val lessonId = current?.roomToken?.lessonId ?: return
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            action(lessonId)
                .onSuccess { refreshRoster() }
                .onFailure { _state.update { st -> st.copy(notice = ApiErrors.humanError(it)) } }
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
        platform.updateSignals(
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
            s.events.collect { event ->
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
                        onDisconnected(s, event.reason?.name)
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
                    //
                    // H3: o'quvchi ovozi LOKAL YOZUVGA ham shu yerda ulanadi. Avval
                    // sink'lar faqat yozuv boshlanganda ulanardi — Record bosilgandan
                    // KEYIN kirgan o'quvchi yozuvda jim qolardi.
                    is RoomEvent.TrackSubscribed -> {
                        (event.track as? AudioTrack)?.let { s.localRecordingOnRemoteAudio(it, added = true) }
                        refreshParticipants(s.room)
                    }
                    is RoomEvent.TrackUnsubscribed -> {
                        (event.track as? AudioTrack)?.let { s.localRecordingOnRemoteAudio(it, added = false) }
                        refreshParticipants(s.room)
                    }
                    is RoomEvent.TrackMuted,
                    is RoomEvent.TrackUnmuted,
                    // M5: "gapiryapti" nuri aynan shu hodisada o'zgaradi — usiz plitka
                    // keyingi tasodifiy hodisagacha eski holatda qotib turardi.
                    is RoomEvent.ActiveSpeakersChanged,
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
                    platform.showShareFrame(recording = _state.value.recordingEnabled)
                    onShareStarted()
                } else {
                    platform.hideShareFrame()
                }
                // H5: yozuv ekran trekiga bog'liq — trek ketsa segment yakunlanadi,
                // qaytsa (ustoz xohlagan bo'lsa) yangisi boshlanadi. Qaror sof.
                when (RecordControl.onShareChanged(on, recordWanted, s.isLocalRecording())) {
                    RecordControl.ShareAction.STOP -> stopAndUploadLocalRecording(s, takeActiveRecording())
                    RecordControl.ShareAction.START -> beginLocalRecording(s)
                    RecordControl.ShareAction.NONE -> Unit
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
                connState = ConnState.fromSdk(room.state.name),
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
        val s = current ?: return@launch
        val target = !_state.value.micOn
        runCatching { s.setMicrophoneEnabled(target) }
            .onSuccess { _state.update { it.copy(micOn = target).withAudioReason() } }
            .onFailure { log("mikrofon XATO: ${it.message}") }
    }

    fun toggleCam() = viewModelScope.launch {
        val s = current ?: return@launch
        val target = !_state.value.camOn
        runCatching { s.setCameraEnabled(target) }
            .onSuccess { _state.update { it.copy(camOn = target) } }
            .onFailure { log("kamera XATO: ${it.message}") }
    }

    /** M12: old ↔ orqa. Kamera o'chiq bo'lsa hech narsa qilinmaydi (UI ham bloklaydi). */
    fun flipCamera() {
        val s = current ?: return
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
     *
     * Yozuv bu yerda boshlanmaydi — u `screenShareOn` oqimidan (trek HAQIQATAN
     * chiqqanda) boshlanadi, shunda tiklangan ulashish ham bir xil yo'ldan o'tadi.
     */
    fun startScreenShare(resultData: Intent) = viewModelScope.launch {
        val s = current ?: run { log("ekran: sessiya yo'q"); return@launch }
        screenShare.start(s, resultData)
        notifySignals(vibrate = false)
    }

    fun stopScreenShare() = viewModelScope.launch {
        val s = current ?: return@launch
        stopAndUploadLocalRecording(s, takeActiveRecording())
        screenShare.stop(s)
        notifySignals(vibrate = false)
    }

    // ─── Client-side (lokal) yozuv orkestratsiyasi ─────────────────────────────

    /**
     * ⭐ RECORD tugmasi (Zoom kabi) — yozuvni QO'LDA boshlash/to'xtatish.
     *
     * Yozuv EKRANni yozadi, shuning uchun ekran ulashilmagan bo'lsa boshlab
     * bo'lmaydi — ustozga hint ko'rsatiladi (avval ulashsin). Qo'lda tanlov
     * [recordWanted] niyatini ham yangilaydi (H5).
     */
    fun toggleRecording() {
        val s = current ?: return
        when (RecordControl.toggle(s.isLocalRecording(), _state.value.screenOn)) {
            RecordControl.Action.STOP -> {
                recordWanted = false
                stopAndUploadLocalRecording(s, takeActiveRecording())
            }
            RecordControl.Action.START -> {
                recordWanted = true
                beginLocalRecording(s)
            }
            RecordControl.Action.HINT_NO_SCREEN -> _state.update {
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
            repeat(3) {
                val file = s.startLocalRecording(LOCAL_REC_BITRATE)
                if (file != null) {
                    activeRecording = ActiveRecording(id, System.currentTimeMillis())
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

    /** Joriy yozuv qatorini oladi va holatni "yozilmayapti" ga o'tkazadi. */
    private fun takeActiveRecording(): ActiveRecording? {
        val rec = activeRecording
        activeRecording = null
        _state.update { it.copy(isRecording = false) }
        return rec
    }

    /**
     * Lokal yozuvni to'xtatadi va ilova qamrovida serverga yuklaydi.
     *
     * Yozuvchini yakunlash bloklaydi (muxer), shuning uchun [appScope] da:
     * u ekran yopilsa ham tugaydi (H4: avval `GlobalScope` edi) va asosiy
     * oqimni ushlab turmaydi (M3). Yuklangan fayl O'CHIRILADI — u shu yozuvga
     * xos (nom noyob), jonli yozuvchi hech qachon o'sha yo'lni ushlamaydi.
     */
    private fun stopAndUploadLocalRecording(s: LessonSession, recording: ActiveRecording?): Job =
        appScope.launch {
            if (!s.isLocalRecording()) return@launch
            val file = s.stopLocalRecording()
            // Yuklash/yetim-fayl qarori sof [RecordControl] da.
            if (!RecordControl.canUpload(recording?.id, file != null)) {
                if (RecordControl.deleteOrphan(recording?.id, file != null)) file?.delete()
                return@launch
            }
            uploadRecording(recording ?: return@launch, file ?: return@launch)
        }

    private suspend fun uploadRecording(recording: ActiveRecording, file: File) {
        val durationSec = RecordControl.durationSec(recording.startedAtMs, System.currentTimeMillis())
        val endedAt = java.time.Instant.now().toString()
        localRec.upload(recording.id, file, durationSec, endedAt)
            .onSuccess { file.delete() }
            .onFailure { android.util.Log.e("RoomVM", "lokal yozuv yuklash XATO — fayl saqlanadi", it) }
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
        val s = current ?: return
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
     * C1: server so'rovi ham, bo'shatish ham [appScope] da. Avval bu
     * `viewModelScope` da edi — UI "Yakunlash" dan keyin darhol ekranni
     * yopardi, qamrov bekor bo'lardi va `leave()` HECH QACHON bajarilmasdi:
     * xona, servis, MediaProjection tirik, yozuv `moov`siz. Server xatosi
     * bo'lsa ham mahalliy resurslar baribir bo'shatiladi.
     */
    fun endLesson(lessonId: String) {
        val s = current
        val recording = takeActiveRecording()
        resetLocalState()
        appScope.launch {
            rooms.endLesson(lessonId)
                .onSuccess { log("dars yakunlandi (server)") }
                .onFailure { log("yakunlash XATO: ${it.message}") }
            if (s != null) teardown(s, recording)
        }
    }

    /** Darsni tark etish (xonani yopmaydi — bu `POST /lessons/:id/end` ishi). */
    fun leave() {
        val s = current
        val recording = takeActiveRecording()
        resetLocalState()
        if (s != null) appScope.launch { teardown(s, recording) }
    }

    /**
     * ViewModel'ning O'Z holatini darhol tozalaydi (sinxron) — ekran shu zahoti
     * "chiqdim" holatini ko'rsatadi; og'ir bo'shatish [teardown] da davom etadi.
     */
    private fun resetLocalState() {
        platform.hideShareFrame()
        cancelObservers()
        current = null
        recordWanted = false
        screenShare.reset()
        shareBackgroundHandled = false
        joinGuard.onReleased()
        _state.value = RoomUiState(autoBackgroundOnShare = uiPrefs.autoBackgroundOnShare)
    }

    /**
     * To'liq bo'shatish — ilova qamrovida. Tartib: yozuvni yakunlab yuklashga
     * qo'yish → store orqali sessiya + servis. Yozuv AVVAL: `release()` ham
     * yozuvni yakunlaydi (C2), lekin faqat shu yerda uni serverga bog'laydigan
     * ID bor.
     */
    private suspend fun teardown(s: LessonSession, recording: ActiveRecording?) {
        stopAndUploadLocalRecording(s, recording).join()
        sessions.release(s)
    }
}
