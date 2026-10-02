package uz.darsly.mentor.data.livekit

import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.VideoTrack
import io.livekit.android.webrtc.peerconnection.RTCThreadToken
import livekit.org.webrtc.AudioTrackSink
import livekit.org.webrtc.VideoSink

/**
 * JVM'da yaratiladigan trek nusxalari.
 *
 * `livekit.org.webrtc.*Track(long)` konstruktorlari faqat native ko'rsatkichni
 * saqlaydi (JNI chaqirmaydi) — shuning uchun soxta ko'rsatkich bilan
 * yaratiladi; native metodlar ([AudioTrack.addSink] kabi) bu yerda
 * ustidan yozilgan, tekshirilayotgan kod ularga tegmaydi.
 */
private object NoThread : RTCThreadToken {
    override val isDisposed: Boolean get() = false
}

class FakeAudioTrack(name: String = "audio") :
    AudioTrack(name, livekit.org.webrtc.AudioTrack(1L), NoThread) {
    val sinks = mutableListOf<AudioTrackSink>()
    override fun addSink(sink: AudioTrackSink) { sinks += sink }
    override fun removeSink(sink: AudioTrackSink) { sinks -= sink }
}

class FakeVideoTrack(name: String = "screen") :
    VideoTrack(name, livekit.org.webrtc.VideoTrack(1L), NoThread) {
    val renderers = mutableListOf<VideoSink>()
    override fun addRenderer(renderer: VideoSink) { renderers += renderer }
    override fun removeRenderer(renderer: VideoSink) { renderers -= renderer }
}
