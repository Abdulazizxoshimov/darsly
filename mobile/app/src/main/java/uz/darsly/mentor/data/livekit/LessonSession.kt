package uz.darsly.mentor.data.livekit

import android.app.Notification
import android.content.Intent
import io.livekit.android.events.RoomEvent
import io.livekit.android.room.Room
import io.livekit.android.room.track.AudioTrack
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.StateFlow
import uz.darsly.mentor.data.api.RoomToken
import java.io.File

/**
 * Bir dars sessiyasi — LiveKit xonasi va uning media holati.
 *
 * ## Nega interfeys
 *
 * Amaldagi sinf ([LiveKitLessonSession]) `LiveKit.create(...)` ni konstruktorda
 * chaqiradi — uni JVM testida yaratib bo'lmaydi. Natijada sessiyaga tegishli
 * har bir buyurtma ([LessonSessionStore], `RoomViewModel`, `ScreenShareController`,
 * `ReconnectController`) sinovdan tashqarida qolgan edi va aynan o'sha joylarda
 * eng qimmat xatolar yashirinardi: yakunlashda bo'shatilmay qolgan yozuv,
 * o'lik trekka bog'lanib qolgan recorder, bekor qilingan qamrovda yarim
 * qolgan tozalash.
 *
 * Interfeys chegarasi shu sinflarni LiveKit'siz sinash imkonini beradi.
 * [room] SDK turi bo'lib qoladi — video plitkalari renderer'ni trekka ulashi
 * kerak; testdagi soxta sessiya unga tegmaydi.
 */
interface LessonSession {
    val lessonId: String
    val roomToken: RoomToken

    /** LiveKit xonasi — faqat UI (sahna) va hodisa kuzatuvchisi uchun. */
    val room: Room

    /** Xona hodisalari — `room.events` ning oqim ko'rinishi (soxta sessiyada bo'sh). */
    val events: Flow<RoomEvent>

    val screenShareOn: StateFlow<Boolean>
    val screenAudioOn: StateFlow<Boolean>
    val cameraFront: StateFlow<Boolean>

    /** Ustoz ulashishni TIZIM panelidan ("Stop sharing") to'xtatdi. */
    var onUserStoppedShare: (() -> Unit)?

    suspend fun connect()
    suspend fun setCameraEnabled(enabled: Boolean)
    suspend fun setMicrophoneEnabled(enabled: Boolean)

    /** @return almashtirish bajarildimi (kamera o'chiq bo'lsa `false`). */
    fun flipCamera(): Boolean

    suspend fun startScreenShare(resultData: Intent, notification: Notification?)
    suspend fun stopScreenShare()
    suspend fun startScreenAudio(): Boolean

    /** @return hozir ekran treki haqiqatan e'lon qilinganmi (bayroq shunga tenglashtiriladi). */
    fun reconcileScreenShare(): Boolean

    /** @return yozuv fayli yoki `null` (ekran treki yo'q / qo'llab-quvvatlanmadi). */
    fun startLocalRecording(bitrate: Int): File?

    /** Yozuvni yakunlaydi (`moov` yoziladi) va yuklashga yaroqli faylni qaytaradi. */
    suspend fun stopLocalRecording(): File?

    fun isLocalRecording(): Boolean

    /** O'quvchi dars o'rtasida qo'shilsa/chiqsa — lokal yozuvga ovozini ulash/uzish. */
    fun localRecordingOnRemoteAudio(track: AudioTrack, added: Boolean)

    /**
     * Barcha resurslarni bo'shatadi: yozuv yakunlanadi, ekran audiosi va xona
     * yopiladi. **Idempotent** — takroriy chaqiruv zararsiz.
     */
    suspend fun release()
}

/** Sessiya yasovchi — [LessonSessionStore] uchun; testda soxta sessiya beradi. */
fun interface LessonSessionFactory {
    fun create(lessonId: String, token: RoomToken): LessonSession
}
