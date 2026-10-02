package uz.darsly.mentor.data.livekit

import io.livekit.android.room.track.AudioTrack
import io.livekit.android.room.track.VideoTrack
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

/**
 * Lokal yozuv egaligi (C2 · H3 · H4) — kodeksiz, soxta yozuvchi bilan.
 */
class LocalRecordingControllerTest {

    @get:Rule val tmp = TemporaryFolder()

    /** Faylga bir necha bayt yozadi — "yaroqli" chiqish. */
    private class FakeRecorder(private val file: File, private val writeBytes: Int = 8) : Recorder {
        var stopCalls = 0
        val audio = mutableListOf<Pair<AudioTrack, Boolean>>()
        override fun start(screenVideo: VideoTrack, videoBitrate: Int, localAudio: AudioTrack?, remoteAudio: List<AudioTrack>): Boolean {
            if (writeBytes > 0) file.writeBytes(ByteArray(writeBytes))
            return true
        }
        override fun addRemoteAudio(track: AudioTrack) { audio += track to true }
        override fun removeRemoteAudio(track: AudioTrack) { audio += track to false }
        override fun stop() { stopCalls++ }
    }

    private val recorders = mutableListOf<FakeRecorder>()
    private var clock = 1_000L
    private fun controller(writeBytes: Int = 8) = LocalRecordingController(
        dir = File(tmp.root, "recordings"),
        factory = RecorderFactory { f -> FakeRecorder(f, writeBytes).also { recorders += it } },
        now = { clock },
    )

    private val video = FakeVideoTrack()

    @Test
    fun `start yangi faylga yozadi va uni qaytaradi`() {
        val c = controller()
        val f = c.start("l1", video, 1_000, null, emptyList())
        assertNotNull(f)
        assertEquals("l1_1000.mp4", f?.name)
        assertTrue(c.isRecording)
    }

    @Test
    fun `H4 — har yozuv ALOHIDA fayl, eski yol qayta ochilmaydi`() = runBlocking {
        val c = controller()
        val first = c.start("l1", video, 1_000, null, emptyList())
        c.stop()
        clock = 2_000L
        val second = c.start("l1", video, 1_000, null, emptyList())
        assertNotEquals("ikkinchi Record avvalgi (yuklanayotgan) faylni kesmasligi kerak", first, second)
    }

    @Test
    fun `ikkinchi start yangi yozuvchi ochmaydi`() {
        val c = controller()
        val f1 = c.start("l1", video, 1_000, null, emptyList())
        val f2 = c.start("l1", video, 1_000, null, emptyList())
        assertEquals(f1, f2)
        assertEquals(1, recorders.size)
    }

    @Test
    fun `C2 — stop yozuvchini yakunlaydi va idempotent`() = runBlocking {
        val c = controller()
        c.start("l1", video, 1_000, null, emptyList())
        val out = c.stop()
        assertNotNull("yaroqli fayl qaytadi", out)
        assertFalse(c.isRecording)
        assertNull("ikkinchi stop hech nima qilmaydi", c.stop())
        assertEquals("muxer.stop() aynan bir marta", 1, recorders.single().stopCalls)
    }

    @Test
    fun `bosh fayl yuklashga yaroqsiz — null`() = runBlocking {
        val c = controller(writeBytes = 0)
        c.start("l1", video, 1_000, null, emptyList())
        assertNull(c.stop())
        assertEquals("yozuvchi baribir yakunlanadi", 1, recorders.single().stopCalls)
    }

    @Test
    fun `H3 — oquvchi ovozi yozuv ketayotganda ulanadi, yoqligida etiborsiz`() {
        val c = controller()
        val track = FakeAudioTrack()
        c.onAudio(track, added = true) // yozuv yo'q — hech nima
        assertTrue(recorders.isEmpty())

        c.start("l1", video, 1_000, null, emptyList())
        c.onAudio(track, added = true)
        c.onAudio(track, added = false)
        assertEquals(listOf(track to true, track to false), recorders.single().audio)
    }
}

class RecordingFilesTest {

    @Test
    fun `nom noyob va dars IDsi ajratiladi`() {
        val name = RecordingFiles.fileName("3f2a-uuid", 1_700_000_000_000)
        assertEquals("3f2a-uuid_1700000000000.mp4", name)
        assertEquals("3f2a-uuid", RecordingFiles.lessonIdOf(name))
        assertEquals(1_700_000_000_000L, RecordingFiles.startedAtOf(name))
    }

    @Test
    fun `eski nom ham oqiladi`() {
        // Yangilanishdan oldin qolgan `<lessonId>.mp4` yuklanishda davom etsin.
        assertEquals("abc", RecordingFiles.lessonIdOf("abc.mp4"))
        assertNull(RecordingFiles.startedAtOf("abc.mp4"))
    }

    @Test
    fun `mp4 bolmagan yoki bosh nom null`() {
        assertNull(RecordingFiles.lessonIdOf("notes.txt"))
        assertNull(RecordingFiles.lessonIdOf(".mp4"))
        assertNull(RecordingFiles.lessonIdOf("_123.mp4"))
    }
}
