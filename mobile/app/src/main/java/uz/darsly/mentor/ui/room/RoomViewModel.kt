package uz.darsly.mentor.ui.room

import android.app.Application
import android.content.Intent
import android.os.Build
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import io.livekit.android.events.RoomEvent
import io.livekit.android.events.collect
import io.livekit.android.room.Room
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.Net
import uz.darsly.mentor.data.livekit.LessonSession
import uz.darsly.mentor.data.livekit.LessonSessionHolder
import uz.darsly.mentor.data.livekit.ScreenAudioPolicy
import uz.darsly.mentor.service.LessonNotifications
import uz.darsly.mentor.service.LessonService

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
    val error: String? = null,
    /** B-2: mikrofonga ruxsat berilmadi — dars boshlanmaydi. */
    val micDenied: Boolean = false,
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

class RoomViewModel(app: Application) : AndroidViewModel(app) {

    private val _state = MutableStateFlow(RoomUiState())
    val state: StateFlow<RoomUiState> = _state.asStateFlow()

    private var session: LessonSession? = null

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

                val s = LessonSessionHolder.start(ctx, lessonId, token)
                session = s
                observe(s)

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
        val reason = RoomStatus.endReasonOf(sdkReasonName)
        val message = RoomStatus.endMessage(reason)
        log("dars tugadi: $reason")

        _state.update {
            it.copy(
                connecting = false,
                reconnecting = false,
                screenOn = false,
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
                    is RoomEvent.TrackPublished -> log("track e'lon qilindi: ${event.publication.source}")
                    // DIQQAT: TrackPublicationFailed'da exception `val` emas (SDK 2.27.0),
                    // shuning uchun faqat track nomini log qilamiz.
                    is RoomEvent.TrackPublicationFailed -> log("TRACK E'LON XATOSI: ${event.track.name}")
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

        val notification = LessonNotifications.build(ctx, "Ekran ulashilmoqda")
        runCatching { s.startScreenShare(resultData, notification) }
            .onSuccess {
                log("EKRAN ULASHISH BOSHLANDI (720p/15fps)")
                // Ekran audiosi: mikrofon track'iga mikslanadi (API 29+ talab qiladi).
                val ok = s.startScreenAudio()
                log(if (ok) "ekran audiosi YOQILDI" else "ekran audiosi yoqilmadi (API<29 yoki mikrofon o'chiq)")
            }
            .onFailure { t ->
                // Asl xato jurnalda (diagnostika/Sentry uchun), UI'da o'zbekcha matn.
                log("EKRAN ULASHISH XATO: ${t::class.simpleName}: ${t.message}")
                _state.update {
                    it.copy(error = "Ekranni ulashib bo'lmadi — qayta urinib ko'ring")
                }
                LessonService.start(ctx, withProjection = false)
            }
    }

    fun stopScreenShare() = viewModelScope.launch {
        val s = session ?: return@launch
        runCatching { s.stopScreenShare() }
            .onSuccess { log("ekran ulashish to'xtatildi") }
            .onFailure { log("to'xtatish XATO: ${it.message}") }
        LessonService.start(getApplication(), withProjection = false)
    }

    /** Darsni tark etish (xonani yopmaydi — bu `POST /lessons/:id/end` ishi). */
    fun leave() {
        cancelObservers()
        LessonSessionHolder.stop()
        session = null
        joinGuard.onReleased()
        LessonService.stop(getApplication())
        _state.value = RoomUiState()
    }
}
