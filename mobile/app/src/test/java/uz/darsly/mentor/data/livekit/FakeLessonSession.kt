package uz.darsly.mentor.data.livekit

import android.app.Notification
import android.content.Intent
import io.livekit.android.events.RoomEvent
import io.livekit.android.room.Room
import io.livekit.android.room.track.AudioTrack
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import uz.darsly.mentor.data.api.RoomToken
import java.io.File
import java.util.concurrent.atomic.AtomicInteger

/**
 * LiveKit'siz sessiya — [LessonSessionStore], `RoomViewModel`,
 * `ScreenShareController` va `ReconnectController` testlari uchun.
 *
 * [room] ATAYLAB otadi: `Room` JVM'da yaratilmaydi va sinalayotgan mantiq
 * unga tegmasligi kerak — tegsa test shu zahoti buni ko'rsatadi.
 */
class FakeLessonSession(
    override val lessonId: String = "l1",
    override val roomToken: RoomToken = token(lessonId),
    /** `startLocalRecording` uchun ekran treki bormi. */
    var screenTrackAvailable: Boolean = true,
    /** `connect()` har chaqiruvda otadigan xato (`null` — muvaffaqiyat). */
    var connectFailure: (() -> Throwable?)? = null,
    /** `release()` otadigan xato — SDK yiqilganda ham servis to'xtashini sinash uchun. */
    var releaseFailure: Throwable? = null,
) : LessonSession {

    override val room: Room
        get() = error("FakeLessonSession.room — Room JVM'da yo'q; sinalayotgan kod unga tegmasligi kerak")

    override val events = MutableSharedFlow<RoomEvent>()

    private val _screenShareOn = MutableStateFlow(false)
    override val screenShareOn: StateFlow<Boolean> = _screenShareOn
    private val _screenAudioOn = MutableStateFlow(false)
    override val screenAudioOn: StateFlow<Boolean> = _screenAudioOn
    private val _cameraFront = MutableStateFlow(true)
    override val cameraFront: StateFlow<Boolean> = _cameraFront

    override var onUserStoppedShare: (() -> Unit)? = null

    val connectCalls = AtomicInteger(0)
    val releaseCalls = AtomicInteger(0)
    val stopRecordingCalls = AtomicInteger(0)
    var startShareCalls = 0
    var stopShareCalls = 0
    var screenAudioStarted = false
    var micEnabled: Boolean? = null
    var reconcileResult = false
    val remoteAudio = mutableListOf<Pair<AudioTrack, Boolean>>()

    private var recordingFile: File? = null
    val released: Boolean get() = releaseCalls.get() > 0
    val recordingStarted: Int get() = startedRecordings
    private var startedRecordings = 0

    override suspend fun connect() {
        connectCalls.incrementAndGet()
        connectFailure?.invoke()?.let { throw it }
    }

    override suspend fun setCameraEnabled(enabled: Boolean) = Unit
    override suspend fun setMicrophoneEnabled(enabled: Boolean) { micEnabled = enabled }
    override fun flipCamera(): Boolean = false

    override suspend fun startScreenShare(resultData: Intent, notification: Notification?) {
        startShareCalls++
        _screenShareOn.value = true
    }

    override suspend fun stopScreenShare() {
        stopShareCalls++
        _screenShareOn.value = false
    }

    override suspend fun startScreenAudio(): Boolean { screenAudioStarted = true; return true }

    override fun reconcileScreenShare(): Boolean {
        _screenShareOn.value = reconcileResult
        return reconcileResult
    }

    /** Tizim "Stop sharing" / trek yo'qolishi — testdan qo'zg'atiladi. */
    fun simulateShareLost(byUser: Boolean) {
        _screenShareOn.value = false
        if (byUser) onUserStoppedShare?.invoke()
    }

    override fun startLocalRecording(bitrate: Int): File? {
        if (!screenTrackAvailable) return null
        recordingFile?.let { return it }
        startedRecordings++
        return File.createTempFile("fake-rec", ".mp4").also {
            it.writeBytes(ByteArray(16))
            recordingFile = it
        }
    }

    override suspend fun stopLocalRecording(): File? {
        stopRecordingCalls.incrementAndGet()
        return recordingFile.also { recordingFile = null }
    }

    override fun isLocalRecording(): Boolean = recordingFile != null

    override fun localRecordingOnRemoteAudio(track: AudioTrack, added: Boolean) {
        remoteAudio += track to added
    }

    override suspend fun release() {
        releaseCalls.incrementAndGet()
        // Haqiqiy sessiya ham yozuvni yakunlaydi (C2) — soxtasi ham shunday.
        recordingFile = null
        releaseFailure?.let { throw it }
    }

    companion object {
        fun token(lessonId: String) = RoomToken(
            token = "jwt-$lessonId",
            wsUrl = "wss://lk.test",
            roomName = "lesson_$lessonId",
            identity = "mentor",
            role = "host",
            lessonId = lessonId,
        )
    }
}

/** Sessiya yasovchi — yasalgan sessiyalarni eslab qoladi. */
class FakeSessionFactory(
    private val build: (String, RoomToken) -> FakeLessonSession = { id, t -> FakeLessonSession(id, t) },
) : LessonSessionFactory {
    val created = mutableListOf<FakeLessonSession>()
    override fun create(lessonId: String, token: RoomToken): LessonSession =
        build(lessonId, token).also { created += it }
}
