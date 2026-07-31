package uz.darsly.mentor.data.livekit

import io.livekit.android.room.participant.AudioTrackPublishDefaults
import io.livekit.android.room.participant.VideoTrackPublishDefaults
import io.livekit.android.room.track.ScreenSharePresets
import io.livekit.android.room.track.VideoPreset169
import livekit.org.webrtc.RtpParameters

/**
 * Media sifati sozlamalari — **past internetli hududlar uchun**.
 *
 * Alohida obyektga ajratilgani ataylab: bu qiymatlar mahsulot qarorlari
 * (kim nimani ko'radi, zaif tarmoqda nima birinchi qurbon bo'ladi), Android
 * kontekstisiz sinaladi va tasodifan o'zgartirib qo'yilmasligi uchun testlar
 * bilan qotirilgan.
 *
 * ## Barcha raqamlar SDK'dan o'qilgan, taxmin emas
 * `ScreenSharePresets` qiymatlari `livekit-android:2.27.0` artefaktidan
 * o'lchab olingan (loyiha qoidasi: "LiveKit API'lari taxmin qilinmaydi"):
 *
 * | Preset | O'lcham | FPS | Bitrate |
 * |---|---|---|---|
 * | `H360_FPS3`   | 640×360   | 3  | 200 kbps |
 * | `H360_FPS15`  | 640×360   | 15 | 400 kbps |
 * | `H720_FPS5`   | 1280×720  | 5  | 800 kbps |
 * | `H720_FPS15`  | 1280×720  | 15 | 1.5 Mbps |
 * | `H1080_FPS15` | 1920×1080 | 15 | 2.5 Mbps |
 */
object MediaTuning {

    /** Ekran ulashishning asosiy (yuqori) qatlami — matn keskinligi shu yerda. */
    val SCREEN_HIGH: ScreenSharePresets = ScreenSharePresets.H720_FPS15

    /**
     * Ekran ulashishning **past** qatlami — zaif internet uchun.
     *
     * 640×360 @ 3 fps / 200 kbps. Uch kadr sekundiga slayd va kod uchun yetarli:
     * kontent statik, ya'ni fps emas, **o'qilishi** muhim. 200 kbps esa kuchsiz
     * 3G'da ham o'tadi (qurilma sinovida operator kanali ~300 KB/s o'lchangan).
     */
    val SCREEN_LOW: ScreenSharePresets = ScreenSharePresets.H360_FPS3

    /**
     * ⭐ EKRAN ULASHISH — endi SIMULCAST bilan.
     *
     * ## Nima muammo edi
     * Avval `simulcast = false` edi va izohda sabab yozilgandi: "past qatlamlarga
     * bitrate ketmasin, matn keskin qolsin". Niyat to'g'ri, oqibati esa yomon —
     * server jurnali buni raqam bilan ko'rsatdi:
     *
     * ```
     * source: SCREEN_SHARE  layers: [HIGH 1280x720 @1.5 Mbps]   ← BITTA qatlam
     * source: CAMERA        layers: [LOW 320x180, MEDIUM, HIGH] ← uchta
     * ```
     *
     * Bitta qatlam degani — o'quvchi 1.5 Mbit/s ni ko'tara olmasa, pastroq
     * sifatga **tusha olmaydi**: u ekranni umuman ko'rmaydi. Ekran ulashish esa
     * bu mahsulotning asosiy mazmuni (mobil ilova aynan shuning uchun yozilgan).
     *
     * ## Nega endi xavfsiz
     * [VideoTrackPublishDefaults.simulcastLayers] qatlamlarni **o'zimiz tanlash**
     * imkonini beradi. Ya'ni asl e'tiroz ("SDK o'zi keraksiz qatlamlar yasaydi")
     * bartaraf: faqat ikki qatlam bo'ladi — yuqorisi tegilmagan 720p/1.5 Mbps,
     * pastkisi 200 kbps. Yaxshi internetdagi o'quvchi avvalgidek 720p oladi.
     *
     * ## `MAINTAIN_RESOLUTION` — matn uchun hal qiluvchi
     * Kanal torayganda WebRTC nimadandir voz kechishi kerak: **o'lchamdan** yoki
     * **kadr chastotasidan**. Matn uchun javob aniq — o'lcham saqlanadi, fps
     * tushadi. Sekundiga 2 kadr bilan o'qiladigan kod, 15 kadr bilan xiralashgan
     * kodga qaraganda foydaliroq. Default (`null`) bunda BALANCED xulqiga tayanadi
     * va o'lchamni ham tushirishi mumkin edi.
     */
    fun screenSharePublish(): VideoTrackPublishDefaults = VideoTrackPublishDefaults(
        videoEncoding = SCREEN_HIGH.encoding,
        simulcast = true,
        simulcastLayers = listOf(SCREEN_LOW),
        degradationPreference = RtpParameters.DegradationPreference.MAINTAIN_RESOLUTION,
    )

    /**
     * OVOZ — zaif tarmoqda oxirigacha qurbon bo'lmaydigan narsa.
     *
     * ## Nega qiymatlar OSHKORA yozilgan, garchi SDK default'i bilan bir xil bo'lsa ham
     * `AudioTrackPublishDefaults()` da `red=true`, `dtx=true` allaqachon yoqilgan —
     * ya'ni bu yerda "tuzatish" yo'q. Lekin ular **jimgina** yoqilgan: kimdir
     * kelajakda `AudioTrackPublishDefaults` ni boshqa sabab bilan o'rnatsa
     * (masalan bitrate'ni o'zgartirish uchun), RED va DTX beixtiyor o'chib
     * ketardi va buni **hech kim sezmasdi** — ovoz faqat yomon tarmoqda
     * buzila boshlardi, ya'ni aynan eng muhim paytda.
     *
     * · **RED** (RFC 2198) — har paketda oldingi paket nusxasi ketadi. Paket
     *   yo'qolsa ovoz uzilmaydi. Narxi ~2× audio bitrate, lekin audio umumiy
     *   oqimning kichik qismi (48 kbps ↔ ekranning 1.5 Mbps'i).
     * · **DTX** — jim paytda deyarli hech narsa yuborilmaydi. Ustoz gapirmayotgan
     *   vaqtda kanal ekran uchun bo'shaydi.
     *
     * `audioBitrate` ATAYLAB tegilmagan (SDK default'i 48 kbps): uni pasaytirish
     * ovozni yengillashtirardi, lekin ekran audiosi (C-6 — video/musiqa ulashish)
     * sifatini buzardi va **majburiy yozuv** ham shu ovozni saqlaydi.
     */
    /**
     * KAMERA — zaif tarmoqda BIRINCHI qurbon (mahsulot qoidasi: ovoz > ekran > kamera).
     *
     * ## Nega 360p va nega simulcast
     * Dars mazmuni EKRANDA (slayd, misol daftari), kamera esa "gapirayotgan odam"
     * konteksti. Unga HD trafik sarflash — ekrandan o'g'irlash degani. Shuning uchun
     * yuqori qatlam 360p bilan cheklanadi, past qatlam 180p: zaif o'quvchi ustozni
     * butunlay yo'qotmaydi, faqat mayda ko'radi.
     *
     * ## `BALANCED` — ekrандan farqli o'laroq ATAYLAB
     * Ekranда `MAINTAIN_RESOLUTION` (matn o'qilishi shart), kamerada esa aksincha:
     * yuz uchun ravon harakat o'lchamdan muhimroq, shuning uchun WebRTC ikkalasini
     * ham moslashtira olsin.
     *
     * ⚠️ Android SDK'da trek ustuvorligini (`RtpParameters.priority`) publish
     * default'lari orqali berish MUMKIN EMAS (web'da `VideoEncoding.priority` bor).
     * Amaldagi ustuvorlik shu bilan ta'minlanadi: kamera tepasi past (360p),
     * ekran yuqori (720p) va ovoz DTX+RED bilan himoyalangan.
     */
    val CAMERA_HIGH: VideoPreset169 = VideoPreset169.H360
    val CAMERA_LOW: VideoPreset169 = VideoPreset169.H180

    fun cameraPublish(): VideoTrackPublishDefaults = VideoTrackPublishDefaults(
        videoEncoding = CAMERA_HIGH.encoding,
        simulcast = true,
        simulcastLayers = listOf(CAMERA_LOW),
        degradationPreference = RtpParameters.DegradationPreference.BALANCED,
    )

    fun audioPublish(): AudioTrackPublishDefaults = AudioTrackPublishDefaults(
        dtx = true,
        red = true,
    )
}
