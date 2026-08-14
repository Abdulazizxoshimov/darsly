package uz.darsly.mentor.data.livekit

import android.media.MediaCodec
import android.media.MediaCodecInfo
import android.media.MediaFormat
import android.media.MediaMuxer
import android.os.Handler
import android.os.HandlerThread
import android.util.Log
import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.VideoTrack
import livekit.org.webrtc.AudioTrackSink
import livekit.org.webrtc.VideoFrame
import livekit.org.webrtc.VideoSink
import java.io.File
import java.nio.ByteBuffer
import java.util.concurrent.ConcurrentHashMap
import java.util.concurrent.ConcurrentLinkedQueue
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Client-side (lokal) yozib olish — «Zoom local recording».
 *
 * ## Video — VideoSink orqali (Android 14+ tuzog'i)
 * Dastlab ekran ulashish MediaProjection'idan IKKINCHI VirtualDisplay olish
 * rejalashtirilgan edi, LEKIN Android 14+ (API 34) bitta MediaProjection'dan
 * ikkinchi VirtualDisplay yasashni BLOKLAYDI (LiveKit birinchisini olgan).
 * Shuning uchun ekran-ulashish VIDEO trekiga [VideoSink] ulanib, LiveKit
 * allaqachon ushlagan kadrlar ([VideoFrame]) olinadi va MediaCodec'ga I420
 * sifatida beriladi — bitta ushlash, ikki VirtualDisplaysiz.
 *
 * ## Audio — AudioSink orqali
 * LiveKit audio treklaridan (ustoz + o'quvchilar) PCM olinib [AudioMixer]
 * bilan aralashtiriladi (AudioPlaybackCapture call-ovozini ushlolmaydi).
 */
class LocalRecorder(private val outputFile: File) {

    private val running = AtomicBoolean(false)

    // ── Video ────────────────────────────────────────────────────────────────
    private var videoEncoder: MediaCodec? = null
    private var videoTrack: VideoTrack? = null
    private val videoSink = FrameSink()
    private val pendingFrames = ConcurrentLinkedQueue<ByteArray>() // I420 paketlar
    @Volatile private var frameW = 0
    @Volatile private var frameH = 0
    @Volatile private var configuredW = 0
    @Volatile private var bitrate = 2_000_000
    private var videoStarted = AtomicBoolean(false)

    // ── Audio ────────────────────────────────────────────────────────────────
    private var audioEncoder: MediaCodec? = null
    private val mixer = AudioMixer(SAMPLE_RATE, CHANNELS)
    private val audioSinks = ConcurrentHashMap<AudioTrack, TrackSink>()

    // ── Muxer ────────────────────────────────────────────────────────────────
    private var muxer: MediaMuxer? = null
    private var videoTrackIndex = -1
    private var audioTrackIndex = -1
    private var muxerStarted = false
    private val muxerLock = Any()

    private lateinit var encThread: HandlerThread
    private lateinit var encHandler: Handler
    private var startNanos = 0L

    /**
     * Yozishni boshlaydi.
     *
     * @param screenVideo ekran-ulashish video treki (LiveKit LocalScreencastVideoTrack)
     * @param videoBitrate video bitreyt (bps)
     * @param localAudio ustoz mikrofoni treki (null bo'lishi mumkin)
     * @param remoteAudio o'quvchilar audio treklari
     * @return muvaffaqiyat (false → egress fallback)
     */
    fun start(
        screenVideo: VideoTrack,
        videoBitrate: Int,
        localAudio: AudioTrack?,
        remoteAudio: List<AudioTrack>,
    ): Boolean {
        if (running.getAndSet(true)) return true
        return try {
            encThread = HandlerThread("local-rec-enc").apply { start() }
            encHandler = Handler(encThread.looper)
            startNanos = System.nanoTime()
            bitrate = videoBitrate

            muxer = MediaMuxer(outputFile.absolutePath, MediaMuxer.OutputFormat.MUXER_OUTPUT_MPEG_4)

            setupAudio()
            localAudio?.let { attachSink(it) }
            remoteAudio.forEach { attachSink(it) }
            mixer.start { }

            // Video encoder birinchi kadr kelganda sozlanadi (o'lchamni bilish uchun).
            videoTrack = screenVideo
            screenVideo.addRenderer(videoSink)

            Log.i(TAG, "local recording started: ${outputFile.name}")
            true
        } catch (t: Throwable) {
            Log.e(TAG, "local recording start failed — egress fallback", t)
            safeStop()
            running.set(false)
            false
        }
    }

    fun addRemoteAudio(track: AudioTrack) {
        if (running.get()) attachSink(track)
    }

    fun removeRemoteAudio(track: AudioTrack) {
        audioSinks.remove(track)?.let { track.removeSink(it) }
    }

    fun stop() {
        if (!running.getAndSet(false)) return
        safeStop()
        Log.i(TAG, "local recording stopped: ${outputFile.name} (${outputFile.length()} bytes)")
    }

    // ─── Video: VideoSink → I420 → MediaCodec ───────────────────────────────────

    private inner class FrameSink : VideoSink {
        override fun onFrame(frame: VideoFrame) {
            if (!running.get()) return
            val i420 = frame.buffer.toI420() ?: return
            try {
                val w = i420.width
                val h = i420.height
                if (w <= 0 || h <= 0) return
                frameW = w; frameH = h
                // Encoder hali sozlanmagan bo'lsa — birinchi kadrda sozlaymiz.
                if (videoStarted.compareAndSet(false, true)) {
                    encHandler.post { setupVideo(w, h) }
                }
                // I420 → tekis bayt massivi (Y + U + V), stride'larsiz zich.
                pendingFrames.offer(packI420(i420))
                // Navbat cheksiz o'smasin (encoder sekin bo'lsa eski kadr tashlansin).
                while (pendingFrames.size > 4) pendingFrames.poll()
            } finally {
                i420.release()
            }
        }
    }

    private fun setupVideo(w: Int, h: Int) {
        try {
            val format = MediaFormat.createVideoFormat(MediaFormat.MIMETYPE_VIDEO_AVC, w, h).apply {
                setInteger(MediaFormat.KEY_COLOR_FORMAT, MediaCodecInfo.CodecCapabilities.COLOR_FormatYUV420Flexible)
                setInteger(MediaFormat.KEY_BIT_RATE, bitrate)
                setInteger(MediaFormat.KEY_FRAME_RATE, VIDEO_FPS)
                setInteger(MediaFormat.KEY_I_FRAME_INTERVAL, 2)
            }
            val enc = MediaCodec.createEncoderByType(MediaFormat.MIMETYPE_VIDEO_AVC)
            enc.setCallback(VideoCallback(), encHandler)
            enc.configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE)
            enc.start()
            videoEncoder = enc
            configuredW = w
        } catch (t: Throwable) {
            Log.e(TAG, "video encoder setup failed", t)
        }
    }

    private inner class VideoCallback : MediaCodec.Callback() {
        override fun onInputBufferAvailable(codec: MediaCodec, index: Int) {
            val frame = pendingFrames.poll()
            if (frame == null) {
                // Kadr yo'q — bufer indeksini keyinroq ishlatishni kutish o'rniga
                // bo'sh 0-baytli kadr bermaymiz (encoder buni yoqtirmaydi);
                // qisqa kutib qayta beramiz.
                encHandler.postDelayed({ if (running.get()) runCatching { onInputBufferAvailable(codec, index) } }, 10)
                return
            }
            try {
                val image = codec.getInputImage(index)
                val size = if (image != null) {
                    fillImageI420(image, frame, frameW, frameH); frame.size
                } else {
                    val buf = codec.getInputBuffer(index) ?: return
                    buf.clear(); buf.put(frame); frame.size
                }
                val pts = (System.nanoTime() - startNanos) / 1000
                codec.queueInputBuffer(index, 0, size, pts, 0)
            } catch (t: Throwable) {
                Log.e(TAG, "video input error", t)
            }
        }
        override fun onOutputBufferAvailable(codec: MediaCodec, index: Int, info: MediaCodec.BufferInfo) {
            drain(codec, index, info, video = true)
        }
        override fun onError(codec: MediaCodec, e: MediaCodec.CodecException) {
            Log.e(TAG, "video encoder error", e)
        }
        override fun onOutputFormatChanged(codec: MediaCodec, format: MediaFormat) {
            synchronized(muxerLock) {
                videoTrackIndex = muxer!!.addTrack(format)
                maybeStartMuxer()
            }
        }
    }

    /** I420Buffer → zich (stride'siz) Y+U+V bayt massivi. */
    private fun packI420(b: livekit.org.webrtc.VideoFrame.I420Buffer): ByteArray {
        val w = b.width; val h = b.height
        val cw = (w + 1) / 2; val ch = (h + 1) / 2
        val out = ByteArray(w * h + cw * ch * 2)
        copyPlane(b.dataY, b.strideY, out, 0, w, h)
        copyPlane(b.dataU, b.strideU, out, w * h, cw, ch)
        copyPlane(b.dataV, b.strideV, out, w * h + cw * ch, cw, ch)
        return out
    }

    private fun copyPlane(src: ByteBuffer, stride: Int, dst: ByteArray, dstOff: Int, w: Int, h: Int) {
        val row = ByteArray(stride)
        var o = dstOff
        for (y in 0 until h) {
            src.position(y * stride)
            val n = minOf(stride, src.remaining())
            src.get(row, 0, n)
            System.arraycopy(row, 0, dst, o, w)
            o += w
        }
    }

    /** Zich I420 → MediaCodec Image plane'lariga (rowStride/pixelStride bilan). */
    private fun fillImageI420(image: android.media.Image, i420: ByteArray, w: Int, h: Int) {
        val cw = (w + 1) / 2; val ch = (h + 1) / 2
        val yPlane = image.planes[0]
        val uPlane = image.planes[1]
        val vPlane = image.planes[2]
        // Y
        putPlane(yPlane, i420, 0, w, h)
        // U, V (I420: U keyin V)
        putPlane(uPlane, i420, w * h, cw, ch)
        putPlane(vPlane, i420, w * h + cw * ch, cw, ch)
    }

    private fun putPlane(plane: android.media.Image.Plane, src: ByteArray, srcOff: Int, w: Int, h: Int) {
        val buf = plane.buffer
        val rowStride = plane.rowStride
        val pixStride = plane.pixelStride
        var s = srcOff
        if (pixStride == 1 && rowStride == w) {
            buf.put(src, srcOff, w * h)
            return
        }
        val rowTmp = ByteArray(w)
        for (y in 0 until h) {
            System.arraycopy(src, s, rowTmp, 0, w); s += w
            buf.position(y * rowStride)
            if (pixStride == 1) {
                buf.put(rowTmp, 0, w)
            } else {
                for (x in 0 until w) buf.put(y * rowStride + x * pixStride, rowTmp[x])
            }
        }
    }

    // ─── Audio ──────────────────────────────────────────────────────────────────

    private fun setupAudio() {
        val format = MediaFormat.createAudioFormat(MediaFormat.MIMETYPE_AUDIO_AAC, SAMPLE_RATE, CHANNELS).apply {
            setInteger(MediaFormat.KEY_AAC_PROFILE, MediaCodecInfo.CodecProfileLevel.AACObjectLC)
            setInteger(MediaFormat.KEY_BIT_RATE, AUDIO_BITRATE)
        }
        val enc = MediaCodec.createEncoderByType(MediaFormat.MIMETYPE_AUDIO_AAC)
        enc.setCallback(AudioCallback(), encHandler)
        enc.configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE)
        enc.start()
        audioEncoder = enc
    }

    private inner class AudioCallback : MediaCodec.Callback() {
        override fun onInputBufferAvailable(codec: MediaCodec, index: Int) {
            val buf = codec.getInputBuffer(index) ?: return
            val pcm = mixer.poll(buf.capacity())
            val pts = (System.nanoTime() - startNanos) / 1000
            if (pcm != null && pcm.isNotEmpty()) {
                buf.clear(); buf.put(pcm)
                codec.queueInputBuffer(index, 0, pcm.size, pts, 0)
            } else {
                encHandler.postDelayed({ if (running.get()) runCatching { onInputBufferAvailable(codec, index) } }, 15)
            }
        }
        override fun onOutputBufferAvailable(codec: MediaCodec, index: Int, info: MediaCodec.BufferInfo) {
            drain(codec, index, info, video = false)
        }
        override fun onError(codec: MediaCodec, e: MediaCodec.CodecException) {
            Log.e(TAG, "audio encoder error", e)
        }
        override fun onOutputFormatChanged(codec: MediaCodec, format: MediaFormat) {
            synchronized(muxerLock) {
                audioTrackIndex = muxer!!.addTrack(format)
                maybeStartMuxer()
            }
        }
    }

    private fun attachSink(track: AudioTrack) {
        if (audioSinks.containsKey(track)) return
        val sink = TrackSink(track)
        audioSinks[track] = sink
        track.addSink(sink)
    }

    private inner class TrackSink(private val track: AudioTrack) : AudioTrackSink {
        override fun onData(
            audioData: ByteBuffer, bitsPerSample: Int, sampleRate: Int,
            numberOfChannels: Int, numberOfFrames: Int, timestampMs: Long,
        ) {
            mixer.feed(track, audioData, bitsPerSample, sampleRate, numberOfChannels, numberOfFrames)
        }
    }

    // ─── Muxer ────────────────────────────────────────────────────────────────

    private fun maybeStartMuxer() {
        if (!muxerStarted && videoTrackIndex >= 0 && audioTrackIndex >= 0) {
            muxer!!.start()
            muxerStarted = true
        }
    }

    private fun drain(codec: MediaCodec, index: Int, info: MediaCodec.BufferInfo, video: Boolean) {
        if (info.flags and MediaCodec.BUFFER_FLAG_CODEC_CONFIG != 0) {
            codec.releaseOutputBuffer(index, false); return
        }
        val out = codec.getOutputBuffer(index)
        if (out != null && info.size > 0) {
            synchronized(muxerLock) {
                if (muxerStarted) {
                    out.position(info.offset)
                    out.limit(info.offset + info.size)
                    muxer!!.writeSampleData(if (video) videoTrackIndex else audioTrackIndex, out, info)
                }
            }
        }
        codec.releaseOutputBuffer(index, false)
    }

    private fun safeStop() {
        runCatching { videoTrack?.removeRenderer(videoSink) }; videoTrack = null
        audioSinks.forEach { (t, s) -> runCatching { t.removeSink(s) } }
        audioSinks.clear()
        runCatching { mixer.stop() }
        pendingFrames.clear()
        runCatching { videoEncoder?.stop(); videoEncoder?.release() }; videoEncoder = null
        runCatching { audioEncoder?.stop(); audioEncoder?.release() }; audioEncoder = null
        synchronized(muxerLock) {
            if (muxerStarted) runCatching { muxer?.stop() }
            runCatching { muxer?.release() }
            muxer = null; muxerStarted = false
            videoTrackIndex = -1; audioTrackIndex = -1
        }
        if (::encThread.isInitialized) runCatching { encThread.quitSafely() }
    }

    companion object {
        private const val TAG = "LocalRecorder"
        const val VIDEO_FPS = 24
        const val SAMPLE_RATE = 48000
        const val CHANNELS = 1
        const val AUDIO_BITRATE = 96_000
    }
}
