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
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicBoolean
import uz.darsly.mentor.data.livekit.RecorderPipeline.DrainAction

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
 * ## Orientatsiya — DOIM gorizontal ([OUT_W]×[OUT_H])
 * Manba tik (portret telefon) bo'lsa, yozuv ham tik chiqardi va pleyerda
 * to'liq ekranga (kino kabi) ochilmasdi — yon tomonda katta bo'sh joy qolardi.
 * Endi encoder DOIM 16:9 gorizontal tuvalga sozlanadi va har kadr nisbatini
 * saqlab tuval markaziga joylanadi ([LetterboxFit], qora chegara bilan). Shu
 * sabab: (1) fayl har doim gorizontal — Zoom kabi to'liq ekranga ochiladi;
 * (2) dars o'rtasida qurilma burilsa (manba o'lchami o'zgarsa) ham encoder
 * qayta sozlashsiz ishlayveradi.
 *
 * ## Audio — AudioSink orqali
 * LiveKit audio treklaridan (ustoz + o'quvchilar) PCM olinib [AudioMixer]
 * bilan aralashtiriladi (AudioPlaybackCapture call-ovozini ushlolmaydi).
 *
 * ## Oqimlar (M3)
 * Kodek callback'lari `local-rec-enc` oqimida. [stop] ham butun bo'shatishni
 * AYNAN o'sha oqimga yuboradi va tugashini kutadi: shunda `drain()` bilan
 * `muxer.stop()` hech qachon bir vaqtda ishlamaydi (avval bu asosiy oqimda
 * edi va bo'shatilgan kodek ustidan kelgan `drain` `IllegalStateException`
 * bilan ilovani yiqitardi).
 */
class LocalRecorder(private val outputFile: File) : Recorder {

    // Boshlash/to'xtatish idempotentligi — sof, atomik hayot sikli (JVM testida qulflangan).
    private val lifecycle = RecorderLifecycle()

    // ── Video ────────────────────────────────────────────────────────────────
    private var videoEncoder: MediaCodec? = null
    private var videoTrack: VideoTrack? = null
    private val videoSink = FrameSink()
    private val pendingFrames = ConcurrentLinkedQueue<ByteArray>() // OUT_W×OUT_H I420 tuvallar
    // Tuval buferlari qayta ishlatiladi (M8): har kadrda 1.38 MB ajratish
    // ushlash oqimini GC bilan to'xtatib turardi.
    private val framePool = FramePool(OUT_W * OUT_H * 3 / 2)
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
    override fun start(
        screenVideo: VideoTrack,
        videoBitrate: Int,
        localAudio: AudioTrack?,
        remoteAudio: List<AudioTrack>,
    ): Boolean {
        if (!lifecycle.beginStart()) return true
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

            // Video encoder birinchi kadr kelganda sozlanadi (gorizontal tuval —
            // o'lcham qat'iy [OUT_W]×[OUT_H], manbaga bog'liq emas).
            videoTrack = screenVideo
            screenVideo.addRenderer(videoSink)

            Log.i(TAG, "local recording started: ${outputFile.name}")
            true
        } catch (t: Throwable) {
            Log.e(TAG, "local recording start failed — egress fallback", t)
            // Boshlanishda yiqildi — kodek oqimi hali callback bermaydi, shu
            // yerda to'g'ridan-to'g'ri bo'shatish xavfsiz.
            safeStop()
            lifecycle.failStart()
            false
        }
    }

    override fun addRemoteAudio(track: AudioTrack) {
        if (lifecycle.isRunning()) attachSink(track)
    }

    override fun removeRemoteAudio(track: AudioTrack) {
        audioSinks.remove(track)?.let { track.removeSink(it) }
    }

    /**
     * Yozuvni yakunlaydi. Bo'shatish kodek oqimida bajariladi va shu yerda
     * kutiladi (eng ko'pi [STOP_TIMEOUT_MS]) — chaqiruvchi fayl tayyor deb
     * ishonishi mumkin. BLOKLAYDI: asosiy oqimdan chaqirilmasin
     * ([LocalRecordingController.stop] `Dispatchers.Default` da chaqiradi).
     */
    override fun stop() {
        if (!lifecycle.beginStop()) return
        val done = CountDownLatch(1)
        val posted = ::encHandler.isInitialized && encHandler.post {
            safeStop()
            done.countDown()
        }
        if (!posted) {
            safeStop()
            return
        }
        if (!done.await(STOP_TIMEOUT_MS, TimeUnit.MILLISECONDS)) {
            Log.w(TAG, "local recording stop timed out — muxer may be unfinished")
        }
        Log.i(TAG, "local recording stopped: ${outputFile.name} (${outputFile.length()} bytes)")
    }

    // ─── Video: VideoSink → I420 → MediaCodec ───────────────────────────────────

    private inner class FrameSink : VideoSink {
        override fun onFrame(frame: VideoFrame) {
            if (!lifecycle.isRunning()) return
            val w = frame.buffer.width
            val h = frame.buffer.height
            if (w <= 0 || h <= 0) return
            // Encoder hali sozlanmagan bo'lsa — birinchi kadrda sozlaymiz
            // (o'lcham qat'iy gorizontal tuval, manbaga bog'liq emas).
            if (videoStarted.compareAndSet(false, true)) {
                encHandler.post { setupVideo() }
            }
            // Kadrni nisbatini saqlab gorizontal tuval markaziga joylaymiz
            // (letterbox). Natija — doim OUT_W×OUT_H zich I420 bayt massivi.
            // Bo'sh bufer yo'q — encoder orqada, kadr tashlanadi (M8).
            val canvas = framePool.acquire() ?: return
            if (!composeLandscape(frame.buffer, w, h, canvas)) {
                framePool.release(canvas)
                return
            }
            pendingFrames.offer(canvas)
            // Navbat cheksiz o'smasin (encoder sekin bo'lsa eski kadr tashlansin).
            while (RecorderPipeline.shouldDropOldest(pendingFrames.size)) {
                pendingFrames.poll()?.let { framePool.release(it) }
            }
        }
    }

    private fun setupVideo() {
        try {
            val format = MediaFormat.createVideoFormat(MediaFormat.MIMETYPE_VIDEO_AVC, OUT_W, OUT_H).apply {
                setInteger(MediaFormat.KEY_COLOR_FORMAT, MediaCodecInfo.CodecCapabilities.COLOR_FormatYUV420Flexible)
                setInteger(MediaFormat.KEY_BIT_RATE, bitrate)
                setInteger(MediaFormat.KEY_FRAME_RATE, VIDEO_FPS)
                setInteger(MediaFormat.KEY_I_FRAME_INTERVAL, 2)
                // VBR — statik kadrда (doska/slayd o'zgarmasa) bitni isrof qilmaydi,
                // harakatли joyда oshiradi. CBR har kadrга teng bit yozib faylni
                // behuda kattalashtirardi. Yakuniy hajmni post-siqish (dars tugagach,
                // telefonda H.264 CRF) hal qiladi; bu — jonli oraliq faylni yengillashtirish.
                setInteger(
                    MediaFormat.KEY_BITRATE_MODE,
                    MediaCodecInfo.EncoderCapabilities.BITRATE_MODE_VBR,
                )
            }
            val enc = MediaCodec.createEncoderByType(MediaFormat.MIMETYPE_VIDEO_AVC)
            enc.setCallback(VideoCallback(), encHandler)
            enc.configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE)
            enc.start()
            videoEncoder = enc
        } catch (t: Throwable) {
            Log.e(TAG, "video encoder setup failed", t)
        }
    }

    /**
     * Manba kadrini ([src], [srcW]×[srcH]) gorizontal tuval ([OUT_W]×[OUT_H])
     * ichiga nisbatini saqlab [canvas] ga joylashtiradi — zich I420 (Y+U+V,
     * stride'siz — [VideoCallback] shu tartibda kutadi). Bo'sh joy qora.
     *
     * Joylashuv hisobi [LetterboxFit] da (sof, JVM test ostida). Masshtablash
     * LiveKit/libwebrtc `cropAndScale` bilan (native, tez).
     *
     * @return `false` — kadr o'girilmadi (tuval qaytariladi).
     */
    private fun composeLandscape(src: VideoFrame.Buffer, srcW: Int, srcH: Int, canvas: ByteArray): Boolean {
        val fit = LetterboxFit.fit(srcW, srcH, OUT_W, OUT_H)
        val scaled = try {
            src.cropAndScale(0, 0, srcW, srcH, fit.w, fit.h)
        } catch (t: Throwable) {
            Log.e(TAG, "cropAndScale failed", t); return false
        }
        val i420 = scaled.toI420()
        return try {
            if (i420 == null) return false
            fillBlack(canvas)
            blit(i420, canvas, fit.x, fit.y)
            true
        } finally {
            i420?.release()
            scaled.release()
        }
    }

    /** Tuvalni "video qora" bilan to'ldiradi (Y=16, U=V=128). */
    private fun fillBlack(canvas: ByteArray) {
        val ySize = OUT_W * OUT_H
        java.util.Arrays.fill(canvas, 0, ySize, 16.toByte())
        java.util.Arrays.fill(canvas, ySize, canvas.size, 128.toByte())
    }

    /** Masshtablangan I420 ([src]) ni tuval ([canvas]) ichiga ([ox],[oy]) burchakdan ko'chiradi. */
    private fun blit(src: livekit.org.webrtc.VideoFrame.I420Buffer, canvas: ByteArray, ox: Int, oy: Int) {
        val dw = src.width; val dh = src.height
        val cOutW = OUT_W / 2; val cOutH = OUT_H / 2
        val ySize = OUT_W * OUT_H
        copyInto(src.dataY, src.strideY, dw, dh, canvas, 0, OUT_W, ox, oy)
        copyInto(src.dataU, src.strideU, (dw + 1) / 2, (dh + 1) / 2, canvas, ySize, cOutW, ox / 2, oy / 2)
        copyInto(src.dataV, src.strideV, (dw + 1) / 2, (dh + 1) / 2, canvas, ySize + cOutW * cOutH, cOutW, ox / 2, oy / 2)
    }

    /** Bitta plane'ni (stride bilan) tuval bayt massiviga ([ox],[oy]) dan zich ko'chiradi. */
    private fun copyInto(
        src: ByteBuffer, srcStride: Int, w: Int, h: Int,
        dst: ByteArray, dstBase: Int, dstStride: Int, ox: Int, oy: Int,
    ) {
        val row = ByteArray(w)
        for (y in 0 until h) {
            src.position(y * srcStride)
            val n = minOf(w, src.remaining())
            src.get(row, 0, n)
            System.arraycopy(row, 0, dst, dstBase + (oy + y) * dstStride + ox, w)
        }
    }

    private inner class VideoCallback : MediaCodec.Callback() {
        override fun onInputBufferAvailable(codec: MediaCodec, index: Int) {
            val frame = pendingFrames.poll()
            if (frame == null) {
                // Kadr yo'q — bufer indeksini keyinroq ishlatishni kutish o'rniga
                // bo'sh 0-baytli kadr bermaymiz (encoder buni yoqtirmaydi);
                // qisqa kutib qayta beramiz.
                encHandler.postDelayed({ if (lifecycle.isRunning()) runCatching { onInputBufferAvailable(codec, index) } }, 10)
                return
            }
            try {
                val image = codec.getInputImage(index)
                val size = if (image != null) {
                    fillImageI420(image, frame, OUT_W, OUT_H); frame.size
                } else {
                    val buf = codec.getInputBuffer(index) ?: return
                    buf.clear(); buf.put(frame); frame.size
                }
                val pts = RecorderPipeline.ptsMicros(System.nanoTime(), startNanos)
                codec.queueInputBuffer(index, 0, size, pts, 0)
            } catch (t: Throwable) {
                Log.e(TAG, "video input error", t)
            } finally {
                // Kadr kodekka ko'chirildi — tuval keyingi kadr uchun bo'sh (M8).
                framePool.release(frame)
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
                videoTrackIndex = muxer?.addTrack(format) ?: return
                maybeStartMuxer()
            }
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
            val pts = RecorderPipeline.ptsMicros(System.nanoTime(), startNanos)
            if (pcm != null && pcm.isNotEmpty()) {
                buf.clear(); buf.put(pcm)
                codec.queueInputBuffer(index, 0, pcm.size, pts, 0)
            } else {
                encHandler.postDelayed({ if (lifecycle.isRunning()) runCatching { onInputBufferAvailable(codec, index) } }, 15)
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
                audioTrackIndex = muxer?.addTrack(format) ?: return
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
        val m = muxer ?: return
        if (!muxerStarted && RecorderPipeline.muxerReady(videoTrackIndex, audioTrackIndex)) {
            m.start()
            muxerStarted = true
        }
    }

    /**
     * Kodek chiqishini muxerga yozadi.
     *
     * Bo'shatishga QARSHI QO'RIQLANGAN (M3): [stop] kodek oqimida bajarilgani
     * uchun bu funksiya u bilan bir vaqtda ishlamaydi, lekin navbatda qolgan
     * callback bo'shatishdan KEYIN kelishi mumkin — u holda kodek allaqachon
     * o'lik va har chaqiruv istisno beradi. Shu sabab hayot sikli tekshiriladi
     * va SDK chaqiruvlari `runCatching` ichida.
     */
    private fun drain(codec: MediaCodec, index: Int, info: MediaCodec.BufferInfo, video: Boolean) {
        if (!lifecycle.isRunning()) return
        runCatching {
            val isConfig = info.flags and MediaCodec.BUFFER_FLAG_CODEC_CONFIG != 0
            val out = if (isConfig) null else codec.getOutputBuffer(index)
            synchronized(muxerLock) {
                // Qaror sof [RecorderPipeline] da: config buferi YOZILMAYDI, muxer
                // boshlanmagan yoki bo'sh bufer ham (aks holda fayl buziladi).
                val action = RecorderPipeline.drainAction(isConfig, out?.let { info.size } ?: 0, muxerStarted)
                val m = muxer
                if (action == DrainAction.WRITE && out != null && m != null) {
                    out.position(info.offset)
                    out.limit(info.offset + info.size)
                    m.writeSampleData(if (video) videoTrackIndex else audioTrackIndex, out, info)
                }
            }
            codec.releaseOutputBuffer(index, false)
        }.onFailure { Log.w(TAG, "drain after teardown ignored (${if (video) "video" else "audio"})", it) }
    }

    /** Barcha resurslarni bo'shatadi. Kodek oqimida chaqiriladi ([stop]) — `drain` bilan poyga yo'q. */
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

        /** Muxer yakunlanishini kutish chegarasi — undan keyin fayl "qanday bo'lsa shunday". */
        private const val STOP_TIMEOUT_MS = 5_000L

        /**
         * Yozuv DOIM shu gorizontal tuvalga (16:9) chiqadi. Manba tik bo'lsa
         * [LetterboxFit] bilan markazga joylanadi — fayl baribir gorizontal,
         * pleyerda kino kabi to'liq ekranga ochiladi. 1280 — ekran-ulashish
         * yuqori qatlami bilan bir xil ([ScreenCaptureSize] · MediaTuning).
         */
        const val OUT_W = 1280
        const val OUT_H = 720

        const val SAMPLE_RATE = 48000
        const val CHANNELS = 1
        const val AUDIO_BITRATE = 96_000
    }
}
