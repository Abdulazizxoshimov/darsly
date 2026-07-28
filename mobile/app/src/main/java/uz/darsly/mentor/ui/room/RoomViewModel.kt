package uz.darsly.mentor.ui.room

import android.app.Application
import android.content.Intent
import android.os.Build
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
import io.livekit.android.room.track.Track
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.LowerHandReq
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.api.SendRoomChatReq
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
import uz.darsly.mentor.service.LessonNotifications
import uz.darsly.mentor.service.LessonService
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
        get() = connecting || connState == "connected" || connState == "reconnecting" || screenOn

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

/**
 * Majburiy qayta ulanish urinishlari orasidagi kutish (ms).
 *
 * Birinchisi 800 ms: tarmoq TASDIQLANGAN bo'lsa ham marshrut/DNS keshi bir
 * lahza kechikishi mumkin. Keyingilari o'sib boradi — umumiy oyna ~46 soniya,
 * bu [uz.darsly.mentor.data.livekit.LessonReconnectPolicy] oynasi bilan mos.
 */
private val RECONNECT_DELAYS = longArrayOf(800, 1500, 3000, 6000, 10000, 10000, 15000)

class RoomViewModel(app: Application) : AndroidViewModel(app) {

    private val _state = MutableStateFlow(RoomUiState())
    val state: StateFlow<RoomUiState> = _state.asStateFlow()

    private var session: LessonSession? = null

    /** Data-channel signallarini o'qiydi (sof, JVM testida qoplangan). */
    private val dataParser = RoomDataParser()

    /** Reaksiya yozuvlariga barqaror kalit — Compose `key` uchun. */
    private var reactionSeq = 0L

    /** Majburiy qayta ulanish sikli (bir vaqtda bittasi). */
    private var reconnectJob: Job? = null

    /**
     * Hozir O'ZIMIZ qayta ulanish uchun uzayapmizmi.
     *
     * `@Volatile` emas — hammasi `viewModelScope` (asosiy dispetcher) ichida
     * o'qiladi va yoziladi.
     */
    private var intentionalReconnect = false

    /**
     * Ustoz ekran ulashishni YOQQAN va o'zi to'xtatmagan (C-11 · 3-gipoteza).
     *
     * [RoomUiState.screenOn] dan farqi: u "hozir oqim ketyaptimi" degan FAKT,
     * bu esa "ustoz nima xohlaydi" degan NIYAT. Qayta ulanishda fakt yo'qoladi,
     * niyat esa qoladi — tiklash qarori aynan shu ikkisining farqidan tug'iladi.
     */
    private var shareWanted = false

    /**
     * Oxirgi `createScreenCaptureIntent()` natijasi.
     *
     * Android 13 va pastda qayta ishlatiladi; 14+ da platforma taqiqlaydi va
     * saqlanganidan foyda yo'q — lekin qaror [ScreenSharePlan] da, shuning uchun
     * bu yerda versiya tekshiruvi YO'Q (aks holda qoida ikki joyga bo'linardi).
     */
    private var shareToken: Intent? = null

    /**
     * Ulanish qorovuli. Xato bo'lganda holat IDLE ga qaytadi — ustoz "Qayta urinish"
     * bosa haqiqatan qayta urinib ko'riladi (avvalgi versiyada bloklanib qolardi).
     */
    private val joinGuard = JoinGuard()

    /** Sessiya oqimlarini kuzatuvchi korutinalar — bo'shatishda bekor qilinadi. */
    private val observeJobs = mutableListOf<Job>()

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
        _state.update { it.copy(connecting = true, error = null, micDenied = false) }
        viewModelScope.launch {
            val ctx = getApplication<Application>()
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

                val token = runCatching { Net.api.hostToken(lessonId).data }
                    .getOrElse { t ->
                        _state.update { it.copy(connecting = false, error = ApiErrors.humanError(t)) }
                        releaseSession()
                        LessonService.stop(ctx)
                        return@launch
                    }
                if (token == null) {
                    _state.update { it.copy(connecting = false, error = "Token bo'sh keldi") }
                    releaseSession()
                    LessonService.stop(ctx)
                    return@launch
                }
                log("token OK · room=${token.roomName} · ws=${token.wsUrl}")

                // Sarlavha uchun dars nomi (UUID emas). Alohida, muvaffaqiyatsizlikka
                // chidamli so'rov: nom kelmasa ham dars boshlanaverishi kerak.
                runCatching { Net.api.lessons(limit = 50).data.orEmpty().firstOrNull { it.id == lessonId } }
                    .getOrNull()
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
                    shareWanted = false
                    shareToken = null
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
        if (intentionalReconnect) {
            log("uzilish qayta ulanish uchun — dars davom etadi")
            return
        }
        val reason = RoomStatus.endReasonOf(sdkReasonName)
        val message = RoomStatus.endMessage(reason)
        log("dars tugadi: $reason")

        // Dars TUGADI — tiklaydigan ulashish yo'q (aks holda "Qayta boshlash"
        // dan keyin ustoz so'ramagan taklif chiqib qolardi).
        shareWanted = false
        shareToken = null

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
            ).withAudioReason()
        }

        // Bo'shatish ALOHIDA korutinada: `releaseSession()` kuzatuvchi job'larni
        // bekor qiladi va biz hozir AYNAN o'sha job ichida turamiz — shu joyda
        // to'g'ridan-to'g'ri chaqirsak o'zimizni bekor qilib, tozalashni yarim
        // yo'lda qoldirardik.
        viewModelScope.launch {
            runCatching { s.stopScreenShare() }
                .onFailure { log("ulashishni to'xtatish XATO: ${it.message}") }
            releaseSession()
            LessonService.stop(getApplication())
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
        observeJobs += viewModelScope.launch {
            var prev: Transport? = null
            NetworkMonitor.transports(getApplication()).collect { now ->
                val st = _state.value
                val connected = st.connState == "connected" || st.connState == "reconnecting"
                val note = NetworkSwitchPolicy.label(prev, now)
                if (note != null) log("tarmoq: $note")
                _state.update { it.copy(networkNote = note) }

                if (NetworkSwitchPolicy.shouldForceReconnect(prev, now, connected)) {
                    log("tarmoq almashdi — majburiy qayta ulanish")
                    forceReconnect()
                }
                prev = now
            }
        }
    }

    /**
     * Majburiy qayta ulanish: eski (yaroqsiz) ulanishni uzib, qaytadan ulanadi.
     *
     * SDK'ning o'z retry siklini kutmasdan — chunki u "half-open" soketni faqat
     * timeout orqali sezadi.
     *
     * ## Nega TAKRORIY urinish shart (qurilma sinovi, 2026-07-28)
     * Birinchi versiya bir marta urinardi. LTE'ga o'tishda o'sha yagona urinish
     * `UnknownHostException` bilan yiqildi (tarmoq hali DNS uchun tayyor emas edi)
     * va dars **butunlay o'lik qoldi**: `disconnect()` chaqirilgani uchun SDK'ning
     * o'z retry sikli ham ishlamasdi. Ya'ni tuzatish o'zi yangi nosozlik yasagandi.
     *
     * Endi urinishlar backoff bilan takrorlanadi va ulanish tiklanishi bilan
     * to'xtaydi. Chegaradan oshsa — foydalanuvchiga rost xabar beriladi.
     */
    private fun forceReconnect() {
        val s = session ?: return
        // Bir vaqtda bitta qayta ulanish sikli.
        if (reconnectJob?.isActive == true) return
        reconnectJob = viewModelScope.launch {
            intentionalReconnect = true
            try {
                _state.update { it.copy(reconnecting = true) }
                runCatching { s.room.disconnect() }

                for ((attempt, delayMs) in RECONNECT_DELAYS.withIndex()) {
                    delay(delayMs)
                    if (session !== s) return@launch // sessiya almashdi/yopildi
                    val result = runCatching { s.connect() }
                    if (result.isSuccess) {
                        log("qayta ulandi (${attempt + 1}-urinish)")
                        _state.update { it.copy(reconnecting = false, networkNote = null) }
                        restoreScreenShare()
                        return@launch
                    }
                    // SABAB ham yoziladi: qurilma sinovida "muvaffaqiyatsiz" degan
                    // quruq qator xatoni topishga yordam bermadi (DNS mi, imzo mi,
                    // dublikat identity mi — bilib bo'lmasdi).
                    val t = result.exceptionOrNull()
                    log("qayta ulanish ${attempt + 1}-urinish muvaffaqiyatsiz: ${t?.let { it::class.simpleName }}: ${t?.message}")
                }
                log("qayta ulanib bo'lmadi — urinishlar tugadi")
                _state.update { it.copy(reconnecting = false, networkNote = "Qayta ulanib bo'lmadi") }
            } finally {
                // Bayroq HAR QANDAY holatda tushadi: aks holda keyingi HAQIQIY
                // uzilish ham jimgina yutilib, dars osilib qolardi.
                intentionalReconnect = false
            }
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
            runCatching { Net.api.roomState(lessonId, s.roomToken.token).data }
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
            is RoomSignal.Chat -> {
                val myId = session?.roomToken?.identity
                val ui = ChatMessageUi(
                    id = signal.id,
                    name = signal.senderName,
                    body = signal.body,
                    self = signal.senderIdentity == myId,
                    toIdentity = signal.toIdentity,
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
                val r = RoomReaction(reactionSeq, signal.emoji, signal.name)
                _state.update { it.copy(reactions = ReactionFeed.add(it.reactions, r)) }
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
            runCatching { Net.api.roomChat(lessonId, s.roomToken.token).data.orEmpty() }
                .onSuccess { items ->
                    // Server eng yangidan eskiga beradi — UI'da teskarisi kerak.
                    val ui = items.asReversed().map {
                        ChatMessageUi(
                            id = it.id,
                            name = it.senderName,
                            body = it.body,
                            self = it.senderIdentity == myId,
                            toIdentity = it.toIdentity,
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
            runCatching {
                Net.api.sendRoomChat(lessonId, SendRoomChatReq(s.roomToken.token, body, to)).data
            }.onSuccess { m ->
                if (m == null) return@onSuccess
                val ui = ChatMessageUi(m.id, m.senderName, m.body, self = true, toIdentity = m.toIdentity)
                _state.update { st ->
                    if (st.chat.any { it.id == ui.id }) st else st.copy(chat = (st.chat + ui).takeLast(MAX_CHAT))
                }
            }.onFailure {
                _state.update { st -> st.copy(error = ApiErrors.humanError(it)) }
            }
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
            runCatching { Net.api.participants(lessonId).data.orEmpty() }
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

    fun muteAll() = moderate { Net.api.muteAll(it) }
    fun muteParticipant(identity: String) = moderate { Net.api.muteParticipant(it, identity) }
    fun removeParticipant(identity: String) = moderate { Net.api.removeParticipant(it, identity) }
    fun allowSpeak(identity: String) = moderate { Net.api.allowSpeak(it, identity) }
    fun revokeSpeak(identity: String) = moderate { Net.api.revokeSpeak(it, identity) }

    fun lowerHand(identity: String) = moderate { Net.api.lowerHand(it, LowerHandReq(identity)) }
    fun lowerAllHands() = moderate { Net.api.lowerAllHands(it) }

    /** Foreground bildirishnoma matnini joriy signallar bilan yangilaydi. */
    private fun notifySignals(vibrate: Boolean) {
        val st = _state.value
        LessonNotifications.updateSignals(
            ctx = getApplication(),
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
                    // ⭐ Dars ichidagi signallar: qo'l ko'tarish va reaksiyalar.
                    // Manba serverda (`roomstate`), bu yerda faqat qo'llaymiz.
                    is RoomEvent.DataReceived -> onRoomSignal(dataParser.parse(event.data))
                    else -> Unit
                }
            }
        }
        observeJobs += viewModelScope.launch {
            s.screenShareOn.collect { on ->
                _state.update { it.copy(screenOn = on).withAudioReason() }
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
        val ctx = getApplication<Application>()

        LessonService.start(ctx, withProjection = true)
        log("FGS mediaProjection tipi bilan ishga tushdi")

        val notification = LessonNotifications.build(ctx, "Ekran ulashilmoqda", summary = false)
        runCatching { s.startScreenShare(resultData, notification) }
            .onSuccess {
                log("EKRAN ULASHISH BOSHLANDI (720p/15fps)")
                // Uzilishdan keyin tiklash uchun NIYAT va rozilik saqlanadi (C-11).
                shareWanted = true
                shareToken = resultData
                _state.update { it.copy(restoreShare = false) }
                notifySignals(vibrate = false)
                // Ekran audiosi: mikrofon track'iga mikslanadi (API 29+ talab qiladi).
                val ok = s.startScreenAudio()
                log(if (ok) "ekran audiosi YOQILDI" else "ekran audiosi yoqilmadi (API<29 yoki mikrofon o'chiq)")
            }
            .onFailure { t ->
                // Asl xato jurnalda (diagnostika/Sentry uchun), UI'da o'zbekcha matn.
                log("EKRAN ULASHISH XATO: ${t::class.simpleName}: ${t.message}")
                // Saqlangan rozilik yaroqsiz bo'lib chiqdi (Android uni bir marta
                // beradi) — uni tashlaymiz, aks holda keyingi tiklash ham shu
                // o'lik token bilan urinardi.
                shareToken = null
                _state.update {
                    if (shareWanted) {
                        // Ustoz ulashayotgan edi: bu tiklash urinishining yiqilishi.
                        // "Qayta urinib ko'ring" o'rniga bir bosishlik taklif kerak.
                        it.copy(restoreShare = true)
                    } else {
                        it.copy(error = "Ekranni ulashib bo'lmadi — qayta urinib ko'ring")
                    }
                }
                LessonService.start(ctx, withProjection = false)
            }
    }

    fun stopScreenShare() = viewModelScope.launch {
        val s = session ?: return@launch
        // Ustozning O'Z qarori — endi tiklanmaydi.
        shareWanted = false
        shareToken = null
        _state.update { it.copy(restoreShare = false) }
        runCatching { s.stopScreenShare() }
            .onSuccess { log("ekran ulashish to'xtatildi") }
            .onFailure { log("to'xtatish XATO: ${it.message}") }
        LessonService.start(getApplication(), withProjection = false)
        notifySignals(vibrate = false)
    }

    /** Ustoz "Keyinroq" dedi — taklif yopiladi, lekin niyat saqlanadi. */
    fun dismissRestoreShare() {
        _state.update { it.copy(restoreShare = false) }
        log("ulashishni tiklash taklifi yopildi")
    }

    /**
     * ⭐ UZILISHDAN KEYIN EKRAN ULASHISHNI TIKLASH (C-11 · 3-gipoteza).
     *
     * Qayta ulanish muvaffaqiyatli bo'lgach chaqiriladi. Qaror [ScreenSharePlan] da
     * (sof, testlar ostida); bu yerda faqat uni bajarish.
     *
     * ## Nega Android 14+ da avtomatik EMAS
     * Platforma har yozib olish sessiyasi uchun yangi rozilik talab qiladi — eski
     * `Intent` qayta ishlatilsa `SecurityException`. Ya'ni "hech narsa so'ramay
     * tiklash" texnik jihatdan mumkin emas. Shuning uchun taklif ikki kanal orqali
     * beriladi: ekranda karta VA bildirishnoma — ustoz ulashish paytida odatda
     * boshqa ilovada (PDF, GeoGebra) bo'ladi va kartani ko'rmaydi.
     */
    private fun restoreScreenShare() {
        val s = session ?: return
        // HAQIQAT bayroqdan emas, e'lon qilingan trekdan olinadi: qayta ulanishda
        // trek yo'qoladi, bayroq esa `true` qolib "ulashilmoqda" deb yolg'on
        // ko'rsatardi (qurilmada 2026-07-28 da aynan shu ko'rindi).
        val sharingNow = s.reconcileScreenShare()
        if (shareWanted) log("qayta ulanishdan keyin ekran treki: ${if (sharingNow) "bor" else "yo'q"}")

        val action = ScreenSharePlan.afterReconnect(
            wanted = shareWanted,
            sharingNow = sharingNow,
            hasToken = shareToken != null,
            sdkInt = Build.VERSION.SDK_INT,
        )
        when (action) {
            ScreenSharePlan.Action.NONE -> Unit

            ScreenSharePlan.Action.REUSE_TOKEN -> {
                val token = shareToken ?: return
                log("ekran ulashish avtomatik tiklanmoqda")
                startScreenShare(token)
            }

            ScreenSharePlan.Action.ASK_CONSENT -> {
                log("ekran ulashish uzildi — rozilik qayta so'raladi (Android 14+)")
                _state.update { it.copy(restoreShare = true) }
                // Ustoz boshqa ilovada bo'lsa kartani ko'rmaydi — titratamiz.
                LessonNotifications.alert(
                    getApplication(),
                    "Ekran ulashish uzildi — davom ettirish uchun bosing",
                )
            }
        }
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
            runCatching { Net.api.endLesson(lessonId) }
                .onSuccess { log("dars yakunlandi (server)") }
                .onFailure { log("yakunlash XATO: ${it.message}") }
            leave()
        }
    }

    /** Darsni tark etish (xonani yopmaydi — bu `POST /lessons/:id/end` ishi). */
    fun leave() {
        cancelObservers()
        LessonSessionHolder.stop()
        session = null
        shareWanted = false
        shareToken = null
        joinGuard.onReleased()
        LessonService.stop(getApplication())
        _state.value = RoomUiState()
    }
}
