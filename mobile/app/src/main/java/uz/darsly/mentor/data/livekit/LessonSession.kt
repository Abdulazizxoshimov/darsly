package uz.darsly.mentor.data.livekit

import android.Manifest
import android.app.Notification
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import android.util.DisplayMetrics
import android.view.WindowManager
import androidx.core.content.ContextCompat
import io.livekit.android.LiveKit
import io.livekit.android.RoomOptions
import io.livekit.android.audio.ScreenAudioCapturer
import io.livekit.android.room.Room
import io.livekit.android.room.participant.VideoTrackPublishDefaults
import io.livekit.android.room.track.CameraPosition
import io.livekit.android.room.track.LocalAudioTrack
import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.LocalScreencastVideoTrack
import java.io.File
import io.livekit.android.room.track.LocalVideoTrack
import io.livekit.android.room.track.LocalVideoTrackOptions
import io.livekit.android.room.track.Track
import io.livekit.android.room.track.VideoCaptureParameter
import io.livekit.android.room.track.screencapture.ScreenCaptureParams
import io.livekit.android.util.LKLog
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uz.darsly.mentor.data.api.RoomToken
import uz.darsly.mentor.service.LessonNotifications

/**
 * Bir dars sessiyasi — LiveKit [Room] va uning media holati.
 *
 * ARXITEKTURA QARORI (M15, roadmap §4):
 * Bu obyekt [LessonSessionHolder] ichida process darajasida (singleton) yashaydi,
 * Activity'ga bog'lanmagan. Shu sababli ustoz boshqa ilovaga o'tsa yoki ekranni
 * burasa ham Room uzilmaydi. `LessonService` (foreground) uni tirik ushlab turadi.
 *
 * Web klient bilan mos (frontend/src/livekit/useRoom.js): adaptiveStream + dynacast + simulcast.
 */
class LessonSession(
    appContext: Context,
    val lessonId: String,
    val roomToken: RoomToken,
) {
    private val ctx = appContext.applicationContext

    /**
     * Ekran ulashish sifati: 15 fps (roadmap M18 "matn rejimi").
     * Slayd/kod matni o'qiladigan bo'lishi uchun rezolyutsiya muhim, fps emas —
     * shuning uchun 15 fps da bitrate matn keskinligiga ketadi.
     *
     * ⭐ O'LCHAM ENDI QOTIB QOLMAGAN (№23): u qurilma ekranining NISBATIDAN
     * hisoblanadi ([ScreenCaptureSize]). Avvalgi qat'iy 1280×720 tik telefonda
     * kadrning 78% ini qora yo'lga sarflardi — `docs/PRODUCT.md` da o'lchangan.
     */
    val room: Room = LiveKit.create(
        appContext = ctx,
        options = RoomOptions(
            adaptiveStream = true,
            dynacast = true,
            videoTrackCaptureDefaults = LocalVideoTrackOptions(position = CameraPosition.FRONT),
            screenShareTrackCaptureDefaults = LocalVideoTrackOptions(
                isScreencast = true,
                captureParams = screenCaptureParams(ctx),
            ),
            // Ekran ulashish va ovoz sozlamalari — past internetli hududlar uchun.
            // Sabablar va raqamlar `MediaTuning` da (u sof va testlar ostida).
            screenShareTrackPublishDefaults = MediaTuning.screenSharePublish(),
            videoTrackPublishDefaults = MediaTuning.cameraPublish(),
            audioTrackPublishDefaults = MediaTuning.audioPublish(),
            // C-11: tarmoq almashuvida (Wi-Fi ↔ LTE) tez tiklanish.
            // Sabab va raqamlar `LessonReconnectPolicy` da (u sof va test ostida).
            reconnectPolicy = LessonReconnectPolicy(),
        ),
    )

    private val _screenShareOn = MutableStateFlow(false)
    val screenShareOn: StateFlow<Boolean> = _screenShareOn.asStateFlow()

    /**
     * Ustoz ulashishni TIZIM panelidan ("Stop sharing") to'xtatdi.
     *
     * Nega alohida signal: [screenShareOn] `false` bo'lishining ikki sababi bor —
     * ustoz to'xtatdi yoki qayta ulanish trekni oldi. Birinchisida tiklash
     * KERAK EMAS (ustoz ataylab to'xtatgan), ikkinchisida kerak. Oqimning o'zi
     * bu ikkisini ajratmaydi.
     *
     * Bizning `stopCapture()` chaqiruvimizda ishlamaydi: WebRTC avval
     * `MediaProjection.Callback` ni yechadi, keyin `stop()` qiladi.
     */
    var onUserStoppedShare: (() -> Unit)? = null

    private val _screenAudioOn = MutableStateFlow(false)
    val screenAudioOn: StateFlow<Boolean> = _screenAudioOn.asStateFlow()

    /**
     * M12: hozir old kamera faolmi. Boshlang'ich qiymat `RoomOptions` dagi
     * `CameraPosition.FRONT` bilan mos (yuqorida).
     */
    private val _cameraFront = MutableStateFlow(true)
    val cameraFront: StateFlow<Boolean> = _cameraFront.asStateFlow()

    /** Ekran audiosi mikser'i — to'xtatilganda `releaseAudioResources()` majburiy. */
    private var screenAudioCapturer: ScreenAudioCapturer? = null

    /** Ustoz mikrofonni yoqmoqchimi (UI niyati). Server holati [applyAudioPlan] da hisoblanadi. */
    private var micWanted = false

    /**
     * ADM buferida mikrofon namunalari o'chirilsinmi.
     *
     * `@Volatile`: qiymatni UI oqimi yozadi, **audio oqimi** (ADM callback'i, har ~10 ms)
     * o'qiydi.
     */
    @Volatile
    private var silenceMicSamples = false

    suspend fun connect() {
        room.connect(roomToken.wsUrl, roomToken.token)
    }

    suspend fun setCameraEnabled(enabled: Boolean) {
        room.localParticipant.setCameraEnabled(enabled)
    }

    /**
     * Mikrofon (C-6 bilan yangilangan).
     *
     * Ustoz mute bosganda ekran audiosi **to'xtamaydi** — [ScreenAudioPlan] bo'yicha
     * ovoz `screen_share_audio` trekiga o'tadi va mikrofon namunalari buferda
     * o'chiriladi. Zoom xulqi: mute ustozning ovozini to'xtatadi, video ovozini emas.
     */
    suspend fun setMicrophoneEnabled(enabled: Boolean) {
        micWanted = enabled
        applyAudioPlan()
    }

    /**
     * [ScreenAudioPlan] qarorini haqiqiy treklarga qo'llaydi.
     *
     * Uch narsa bir vaqtda to'g'ri bo'lishi kerak: mikrofon trekining server holati,
     * ekran-audio trekining server holati va bufer mazmuni. Shuning uchun ular
     * bitta joyda, bitta rejadan qo'llanadi.
     */
    private suspend fun applyAudioPlan() {
        val plan = ScreenAudioPlan.plan(
            micWanted = micWanted,
            screenAudioActive = screenAudioCapturer != null,
        )
        silenceMicSamples = plan.silenceMicSamples

        // Ekran ovozi ustoz ovozi bilan birga ketganda pasaytiriladi (ustoz eshitilsin),
        // yolg'iz qolganda to'liq balandlik.
        //
        // Versiya sharti lint uchun oshkora: `ScreenAudioCapturer.gain` `@RequiresApi(Q)`.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            screenAudioCapturer?.gain =
                if (plan.silenceMicSamples) SCREEN_AUDIO_GAIN_SOLO else SCREEN_AUDIO_GAIN_MIXED
        }

        room.localParticipant.setMicrophoneEnabled(plan.micTrackLive)
        _screenAudioOn.value = screenAudioCapturer != null
    }

    /**
     * Old ↔ orqa kamera (M12).
     *
     * Maqsad pozitsiya **oshkora** uzatiladi (`switchCamera(position = …)`), aks holda
     * SDK "keyingi qurilmaga" o'tadi va uch kamerali telefonlarda ustoz qaysi
     * kameradaligini bilmay qoladi. Natija [cameraFront] da saqlanadi — UI tugma
     * yozuvini shundan oladi.
     *
     * @return almashtirish bajarildimi (kamera o'chiq bo'lsa `false`).
     */
    fun flipCamera(): Boolean {
        val track = room.localParticipant
            .getTrackPublication(Track.Source.CAMERA)?.track as? LocalVideoTrack
            ?: return false
        val target = if (_cameraFront.value) CameraPosition.BACK else CameraPosition.FRONT
        track.switchCamera(position = target)
        _cameraFront.value = target == CameraPosition.FRONT
        return true
    }

    /**
     * ⭐ EKRAN ULASHISH — spike'ning yuragi.
     *
     * [resultData] = `MediaProjectionManager.createScreenCaptureIntent()` natijasi
     * (`registerForActivityResult` orqali olinadi).
     *
     * ANDROID 14+ (API 34) TARTIBI:
     * `foregroundServiceType=mediaProjection` bo'lgan servis `startForeground()` bilan
     * MediaProjection yaratilishidan OLDIN ishga tushgan bo'lishi SHART, aks holda
     * SecurityException. Bu tartibni LiveKit SDK O'ZI bajaradi — biz [ScreenCaptureParams]
     * uzatsak, `LocalParticipant.setTrackEnabled` ketma-ketligi shunday:
     *
     *   createScreencastTrack(resultData)   // ScreenCapturerAndroid yasaladi, projection HALI yo'q
     *   track.startForegroundService(...)   // ← FGS avval
     *   track.startCapture()                // ← MediaProjection shundan keyin olinadi
     *
     * (Manba: livekit-android 2.27.0 · LocalParticipant.kt:384-390 — tekshirilgan.)
     *
     * Bizning [LessonService] ham parallel ishlaydi: u Room'ni fon rejimida tirik
     * ushlaydi (SDK'ning ScreenCaptureService'i faqat proyeksiya uchun).
     */
    suspend fun startScreenShare(resultData: Intent, notification: Notification?) {
        room.localParticipant.setScreenShareEnabled(
            enabled = true,
            screenCaptureParams = ScreenCaptureParams(
                mediaProjectionPermissionResultData = resultData,
                notificationId = LessonNotifications.SCREEN_CAPTURE_NOTIFICATION_ID,
                notification = notification,
                // Foydalanuvchi tizim panelidan "Stop sharing" bosganda chaqiriladi.
                onStop = {
                    _screenShareOn.value = false
                    releaseScreenAudio()
                    onUserStoppedShare?.invoke()
                },
            ),
        )
        _screenShareOn.value = true
    }

    suspend fun stopScreenShare() {
        releaseScreenAudio()
        room.localParticipant.setScreenShareEnabled(false)
        _screenShareOn.value = false
    }

    // ─── Client-side (lokal) yozib olish ────────────────────────────────────────

    private var localRecorder: LocalRecorder? = null
    private var localRecFile: File? = null

    /**
     * Telefon ekranни (va ovozни) LOKAL faylga yozishni boshlaydi — ekran
     * ulashish MediaProjection'idan IKKINCHI VirtualDisplay olib. Ovoz LiveKit
     * audio treklaridan (ustoz + o'quvchilar) [AudioTrack.addSink] orqali.
     *
     * @return yozuv fayli (muvaffaqiyat) yoki null (qo'llab-quvvatlanmadi →
     *         chaqiruvchi server egress fallback'iga o'tadi).
     */
    fun startLocalRecording(lessonId: String, bitrate: Int): File? {
        // Ekran-ulashish VIDEO trekini olamiz — LocalRecorder unga VideoSink
        // ulab kadrlarni oladi (Android 14+ 2-VirtualDisplayni bloklaydi).
        val screenTrack = room.localParticipant
            .getTrackPublication(Track.Source.SCREEN_SHARE)?.track as? LocalScreencastVideoTrack ?: return null

        val localAudio = room.localParticipant
            .getTrackPublication(Track.Source.MICROPHONE)?.track as? AudioTrack
        val remoteAudio = room.remoteParticipants.values
            .flatMap { it.audioTrackPublications }
            .mapNotNull { it.second as? AudioTrack }

        val dir = File(ctx.filesDir, "recordings").apply { mkdirs() }
        val file = File(dir, "$lessonId.mp4")

        val rec = LocalRecorder(file)
        return if (rec.start(screenTrack, bitrate, localAudio, remoteAudio)) {
            localRecorder = rec
            localRecFile = file
            file
        } else {
            null
        }
    }

    /** Lokal yozuvни to'xtatadi va faylni qaytaradi (yuklash uchun). */
    fun stopLocalRecording(): File? {
        val rec = localRecorder ?: return null
        localRecorder = null
        rec.stop()
        val file = localRecFile
        localRecFile = null
        return file?.takeIf { it.exists() && it.length() > 0 }
    }

    /** O'quvchi dars o'rtasida qo'shilsa/chiqsa — lokal yozuvга ovozини ulash/uzish. */
    fun localRecordingOnRemoteAudio(track: AudioTrack, added: Boolean) {
        val rec = localRecorder ?: return
        if (added) rec.addRemoteAudio(track) else rec.removeRemoteAudio(track)
    }

    fun isLocalRecording(): Boolean = localRecorder != null

    /**
     * Ekran ulashish bayrog'ini LiveKit'dagi HAQIQIY holat bilan tenglashtiradi.
     *
     * ## Nega kerak (qurilma sinovi, 2026-07-28)
     * [_screenShareOn] — bizning bayrog'imiz: `startScreenShare` da yoqiladi va
     * faqat ustoz to'xtatganda o'chadi. Qayta ulanishda esa trek SERVERDA
     * yo'qoladi, bayroq esa `true` bo'lib qolaveradi — ekranda "Ekraningiz
     * ulashilmoqda" yozuvi turadi, o'quvchilar esa hech narsa ko'rmaydi.
     * Bu darsdagi eng yomon xato turi: interfeys YOLG'ON gapiradi.
     *
     * Shuning uchun tiklash qarori bayroqdan emas, e'lon qilingan TREKDAN
     * boshlanadi. Farq bo'lsa — haqiqat ustun.
     *
     * @return hozir ekran treki haqiqatan e'lon qilinganmi
     */
    fun reconcileScreenShare(): Boolean {
        val published = room.localParticipant
            .getTrackPublication(Track.Source.SCREEN_SHARE)?.track != null
        _screenShareOn.value = published
        return published
    }

    /**
     * ⭐ EKRAN AUDIOSI — mikrofondan MUSTAQIL (M14 · C-6).
     *
     * TASDIQLANGAN CHEKLOVLAR (`livekit-android 2.27.0`, `javap` bilan tekshirilgan):
     *  1. `ScreenAudioCapturer` `@RequiresApi(Q)` — Android 10+. `minSdk 26` bo'lgani
     *     uchun 8.0/8.1/9 da bu funksiya PRINSIPIAL ishlamaydi (platforma cheklovi).
     *  2. Bufer callback'i **ADM darajasida** (bitta `AudioBufferCallbackDispatcher`,
     *     `audioModule(...)` ga uzatiladi), trek darajasida EMAS; telefonda audio
     *     kirishi ham bitta. Ya'ni "ekran audiosi uchun mustaqil MANBA" olish imkoni yo'q.
     *  3. `RECORD_AUDIO` ruxsati kerak; fon rejimi uchun FGS `microphone` tipi kerak.
     *  4. Manba ilova `AudioAttributes` ni `ALLOW_CAPTURE_BY_ALL` qilmasa (YouTube
     *     kabi ko'p ilova qilmaydi) tizim JIM audio qaytaradi — bizning xatomiz emas.
     *
     * (2) tufayli mustaqillik **ikki trekni navbatlashtirish** bilan olinadi —
     * jadval va qaror [ScreenAudioPlan] da. Bu funksiya ikkinchi trekni e'lon qiladi,
     * qaysi biri ovozli bo'lishini [applyAudioPlan] hal qiladi.
     */
    suspend fun startScreenAudio(): Boolean {
        // Allaqachon ishlab turgan bo'lsa qayta yasamaymiz — aks holda ikkinchi
        // `AudioRecord` ochilib, birinchisi oqib ketardi.
        if (screenAudioCapturer != null) return true

        // API DARVOZASI — ataylab shu yerda takrorlangan.
        // `ScreenAudioCapturer` `@RequiresApi(Q)`; tekshiruv faqat `ScreenAudioPolicy`
        // ichida bo'lsa lint uni ko'rmaydi (funksiya chegarasidan o'tmaydi) va
        // `NewApi` XATO beradi — ya'ni Android 8/9 dagi crash xavfi kodda ko'rinmay
        // qoladi. Sabab matni siyosatda, API shartnomasi esa shu qatorda.
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.Q) return false

        if (!ScreenAudioPolicy.canCapture(
                hasRecordAudio = ContextCompat.checkSelfPermission(
                    ctx,
                    Manifest.permission.RECORD_AUDIO,
                ) == PackageManager.PERMISSION_GRANTED,
                sharing = _screenShareOn.value,
                // Mikrofon TREKI kerak (ADM shu tufayli ishlaydi), lekin uning MUTE
                // holati endi to'siq emas — mustaqillikning butun maqsadi shu.
                micOn = room.localParticipant.getTrackPublication(Track.Source.MICROPHONE) != null,
            )
        ) {
            return false
        }

        val screenTrack = room.localParticipant
            .getTrackPublication(Track.Source.SCREEN_SHARE)?.track ?: return false
        val micTrack = room.localParticipant
            .getTrackPublication(Track.Source.MICROPHONE)?.track as? LocalAudioTrack ?: return false

        val capturer = ScreenAudioCapturer.createFromScreenShareTrack(screenTrack) ?: return false
        screenAudioCapturer = capturer

        // Callback ADM darajasida global — qaysi trekda o'rnatilishi ahamiyatsiz;
        // mikrofon treki eng barqaror egasi (butun dars davomida mavjud).
        micTrack.setAudioBufferCallback(ScreenAudioMixer(capturer) { silenceMicSamples })

        applyAudioPlan()
        return true
    }

    /**
     * Ekran audiosi resurslarini bo'shatadi: `AudioRecord`, global bufer callback'i
     * va **ikkinchi trek** (C-6).
     *
     * `releaseAudioResources()` majburiy — SDK `AudioRecord` ni o'zi boshqarmaydi
     * (B-5: bo'shatilmasa mikrofon o'chirilganda ham yozib olish davom etardi).
     */
    private fun releaseScreenAudio() {
        // Bayroq HAR QANDAY holatda tushiriladi: UI haqiqatni ko'rsatishi kerak,
        // capturer bo'lgan-bo'lmaganidan qat'i nazar (B-4).
        _screenAudioOn.value = false
        silenceMicSamples = false

        val capturer = screenAudioCapturer ?: return
        (room.localParticipant.getTrackPublication(Track.Source.MICROPHONE)?.track as? LocalAudioTrack)
            ?.setAudioBufferCallback(null)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            capturer.releaseAudioResources()
        }
        screenAudioCapturer = null
    }

    fun release() {
        releaseScreenAudio()
        room.disconnect()
        room.release()
    }

    private companion object {
        /** Ekran ovozi yolg'iz ketganda — to'liq balandlik. */
        const val SCREEN_AUDIO_GAIN_SOLO = 1.0f

        /** Mikrofon bilan birga ketganda — ustoz ovozi ustidan eshitilishi uchun. */
        const val SCREEN_AUDIO_GAIN_MIXED = 0.6f

        /**
         * Ekran ulashish manbasining o'lchami — qurilma nisbatidan (№23).
         *
         * Nisbat qurilma bo'yicha o'zgarmas (burilish faqat tomonlarni
         * almashtiradi, uni SDK o'zi qiladi), shuning uchun bir marta —
         * sessiya yaratilganda — hisoblansa yetarli.
         */
        fun screenCaptureParams(ctx: Context): VideoCaptureParameter {
            val (w, h) = realDisplaySize(ctx)
            return ScreenCaptureSize.forDisplay(
                displayWidth = w,
                displayHeight = h,
                // Kadr chastotasi o'lchamdan mustaqil qaror — u `MediaTuning` da
                // qoladi (narvonning tepa pog'onasi bilan bir xil).
                maxFps = MediaTuning.SCREEN_HIGH.capture.maxFps,
            )
        }

        /**
         * Ekranning HAQIQIY piksel o'lchami.
         *
         * ATAYLAB eskirgan `defaultDisplay.getRealMetrics` ishlatilgan:
         * LiveKit SDK `LocalScreencastVideoTrack.startCapture()` da aynan shu
         * manbadan o'qiydi va bizning qiymat bilan solishtiradi
         * (`displayWidth > displayHeight` → tomonlarni almashtiradi). Boshqa
         * API (`WindowMetrics`, `resources.displayMetrics`) ba'zi qurilmada
         * boshqacha son beradi — ikki manbaning farqi esa aynan orientatsiya
         * chegarasida jimgina noto'g'ri kadrga olib kelardi.
         */
        @Suppress("DEPRECATION")
        fun realDisplaySize(ctx: Context): Pair<Int, Int> {
            val wm = ctx.getSystemService(Context.WINDOW_SERVICE) as? WindowManager
                ?: return 0 to 0
            val metrics = DisplayMetrics()
            wm.defaultDisplay.getRealMetrics(metrics)
            return metrics.widthPixels to metrics.heightPixels
        }
    }
}
