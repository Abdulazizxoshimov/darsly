package uz.darsly.mentor.data.api

import com.squareup.moshi.Json
import com.squareup.moshi.JsonClass

/**
 * Backend konvensiyasi (docs/api-contract.md):
 *   muvaffaqiyat  → { "data": T }
 *   ro'yxat       → { data, total, page, limit, total_pages }
 *   xato          → { "code": string, "message": string }
 */
@JsonClass(generateAdapter = true)
data class Envelope<T>(
    @Json(name = "data") val data: T?,
)

@JsonClass(generateAdapter = true)
data class ListEnvelope<T>(
    @Json(name = "data") val data: List<T>?,
    @Json(name = "total") val total: Int = 0,
    @Json(name = "page") val page: Int = 1,
    @Json(name = "limit") val limit: Int = 0,
    @Json(name = "total_pages") val totalPages: Int = 0,
)

/**
 * Xato javobi.
 *
 * BE-4 TUZATILDI (2026-07-25): backend endi **barcha** xatolarni (401/403/429 ham)
 * yagona `{"code","message"}` konvertida qaytaradi. Eski `{"error","code"}` shakli
 * qolmagan, lekin eski serverga qarshi ishlab qolmaslik uchun `error` maydoni
 * zaxira sifatida saqlanadi (parse xarajati nol).
 */
@JsonClass(generateAdapter = true)
data class ApiError(
    @Json(name = "code") val code: String? = null,
    @Json(name = "message") val message: String? = null,
    @Json(name = "error") val error: String? = null,
) {
    fun humanMessage(): String = message ?: error ?: code ?: "Noma'lum xato"
}

@JsonClass(generateAdapter = true)
data class MuteAllReq(
    /** null = bayroqqa tegilmaydi; false = «o'quvchilar o'zi ocholmasin» ham yoqiladi. */
    @Json(name = "allow_self_unmute") val allowSelfUnmute: Boolean?,
)

@JsonClass(generateAdapter = true)
data class RemoveParticipantReq(
    /** "lesson" — faqat shu darsdan; "mentor" — doimiy qora ro'yxat. */
    @Json(name = "scope") val scope: String,
)

@JsonClass(generateAdapter = true)
data class LoginReq(
    @Json(name = "email") val email: String,
    @Json(name = "password") val password: String,
)

@JsonClass(generateAdapter = true)
data class TokenPair(
    @Json(name = "access_token") val accessToken: String,
    @Json(name = "refresh_token") val refreshToken: String,
)

/** `POST /auth/refresh` so'rovi (M2). */
@JsonClass(generateAdapter = true)
data class RefreshReq(
    @Json(name = "refresh_token") val refreshToken: String,
)

/**
 * `POST /auth/logout` so'rovi (M3) — sessiyani serverda ham yopadi.
 *
 * [refreshToken] IXTIYORIY (backend 2026-09-30 kontrakti): server joriy
 * sessiyani `Authorization` header'idagi access token bo'yicha HAR DOIM bekor
 * qiladi; refresh berilsa uning JTI'si ham o'chadi. `null` — Moshi maydonni
 * yozmaydi, tana `{}` ketadi (server `omitempty`).
 */
@JsonClass(generateAdapter = true)
data class LogoutReq(
    @Json(name = "refresh_token") val refreshToken: String? = null,
)

/**
 * Foydalanuvchi — `entity.User` ning mobil ilova ishlatadigan qismi.
 *
 * `password_hash`/`deleted_at` backend'da `json:"-"` bilan yopilgan, ya'ni ular
 * hech qachon kelmaydi. Qolgan maydonlar shaxsiy kabinet uchun kerak
 * ([uz.darsly.mentor.ui.profile.ProfileScreen]): ism va til/mintaqa tahrirlanadi,
 * `role`/`created_at` esa faqat ko'rsatiladi.
 */
@JsonClass(generateAdapter = true)
data class User(
    @Json(name = "id") val id: String,
    @Json(name = "email") val email: String,
    @Json(name = "full_name") val fullName: String,
    @Json(name = "role") val role: String,
    @Json(name = "avatar_url") val avatarUrl: String? = null,
    /** Avatar o'rnidagi rang (`#RRGGBB`) — web bilan bir xil ko'rinish uchun. */
    @Json(name = "color") val color: String? = null,
    @Json(name = "timezone") val timezone: String? = null,
    @Json(name = "language") val language: String? = null,
    @Json(name = "is_active") val isActive: Boolean = true,
    @Json(name = "last_login_at") val lastLoginAt: String? = null,
    @Json(name = "created_at") val createdAt: String? = null,
)

/**
 * `PUT /api/v1/users/me` — entity.UpdateUserReq ning ustoz o'zgartira oladigan qismi.
 *
 * `role` ATAYLAB YO'Q: backend uni `PUT /users/me` da baribir `nil` qiladi
 * (`v1/user.go:198` — privilege escalation himoyasi). Uni bu yerga qo'shish
 * "o'zgartirsa bo'ladi" degan yolg'on va'da bo'lardi.
 *
 * `null` maydonlar JSON'ga umuman qo'shilmaydi → o'zgartirilmaydi.
 */
@JsonClass(generateAdapter = true)
data class UpdateProfileReq(
    @Json(name = "full_name") val fullName: String? = null,
    @Json(name = "timezone") val timezone: String? = null,
    @Json(name = "language") val language: String? = null,
)

/** `PUT /api/v1/users/me/password` — entity.ChangePasswordReq. */
@JsonClass(generateAdapter = true)
data class ChangePasswordReq(
    @Json(name = "current_password") val currentPassword: String,
    @Json(name = "new_password") val newPassword: String,
)

/** `POST /api/v1/auth/forgot-password` — entity.ForgotPasswordReq. */
@JsonClass(generateAdapter = true)
data class ForgotPasswordReq(
    @Json(name = "email") val email: String,
)

/**
 * `POST /api/v1/auth/reset-password` — entity.ResetPasswordReq.
 *
 * `token` emaildagi havoladan (`/reset-password?token=…`) olinadi. Ilova uni
 * qo'lda kiritishga ham, havolani yopishtirishga ham ruxsat beradi
 * ([uz.darsly.mentor.ui.auth.PasswordResetForm.extractToken]).
 */
@JsonClass(generateAdapter = true)
data class ResetPasswordReq(
    @Json(name = "token") val token: String,
    @Json(name = "new_password") val newPassword: String,
)

@JsonClass(generateAdapter = true)
data class Lesson(
    @Json(name = "id") val id: String,
    @Json(name = "title") val title: String,
    @Json(name = "description") val description: String? = null,
    @Json(name = "scheduled_at") val scheduledAt: String? = null,
    @Json(name = "duration_min") val durationMin: Int = 0,
    @Json(name = "join_slug") val joinSlug: String? = null,
    @Json(name = "status") val status: String = "scheduled",
    @Json(name = "is_waiting_room_enabled") val isWaitingRoomEnabled: Boolean = false,
    @Json(name = "is_recording_enabled") val isRecordingEnabled: Boolean = false,
    /** Parol bilan himoyalanganmi (backend hisoblaydi, hash hech qachon kelmaydi). */
    @Json(name = "has_passcode") val hasPasscode: Boolean = false,
    @Json(name = "is_locked") val isLocked: Boolean = false,
    /**
     * Ovoz sozlamalari (№11) — server webhook'da qo'llaydi.
     *
     * Default'lar server default'i bilan bir xil (`true`/`true`): eski backend
     * bu maydonlarsiz javob qaytarsa ham forma "mikrofon o'chiq, o'quvchi o'zi
     * yoqa oladi" degan HAQIQIY holatni ko'rsatadi, teskarisini emas.
     */
    @Json(name = "mute_on_entry") val muteOnEntry: Boolean = true,
    @Json(name = "allow_self_unmute") val allowSelfUnmute: Boolean = true,
    /** Ro'yxatni barqaror tartiblash uchun (`scheduled_at` bo'sh darslar). */
    @Json(name = "created_at") val createdAt: String? = null,
)

/**
 * `POST /api/v1/lessons` so'rovi (M7) — entity.CreateLessonReq.
 *
 * DIQQAT: backend `validate` teglari — `title` 2..255, `duration_min` 5..1440
 * (0 → server 60 qo'yadi), `passcode` 4..20, `description` ≤2000.
 * Shu chegaralar klientda [uz.darsly.mentor.ui.lessons.LessonForm] da takrorlangan:
 * ustoz xatoni **serverga bormasdan**, o'zbekcha ko'radi (B-7).
 *
 * `null` maydonlar Moshi tomonidan JSON'ga umuman qo'shilmaydi → server default'i ishlaydi.
 */
@JsonClass(generateAdapter = true)
data class CreateLessonReq(
    @Json(name = "title") val title: String,
    @Json(name = "description") val description: String? = null,
    /** RFC3339 (`2026-07-27T14:30:00Z`) yoki `null` — "vaqti belgilanmagan". */
    @Json(name = "scheduled_at") val scheduledAt: String? = null,
    @Json(name = "duration_min") val durationMin: Int = 60,
    @Json(name = "passcode") val passcode: String? = null,
    @Json(name = "is_recording_enabled") val isRecordingEnabled: Boolean = false,
    @Json(name = "is_waiting_room_enabled") val isWaitingRoomEnabled: Boolean = false,
    /**
     * Ovoz sozlamalari (№11). Server default'i ikkalasi uchun ham `true`,
     * shuning uchun bu yerdagi default'lar ham `true` — forma yubormasa ham
     * natija bir xil bo'ladi (yolg'on farq yo'q).
     */
    @Json(name = "mute_on_entry") val muteOnEntry: Boolean = true,
    @Json(name = "allow_self_unmute") val allowSelfUnmute: Boolean = true,
)

/**
 * `PATCH /api/v1/lessons/{id}` — entity.UpdateLessonReq.
 *
 * HAMMA maydon `null` bo'la oladi va `null` maydon JSON'ga **qo'shilmaydi** —
 * ya'ni "tegilmadi" degani. Bu yerda [CreateLessonReq] dan farqli o'laroq bu
 * xatti-harakat majburiy: bitta sarlavhani o'zgartirganda darsning paroli yoki
 * kutish xonasi jimgina nolga tushib qolmasligi kerak.
 *
 * PAROL uch holatli, shuning uchun ikkita maydon:
 *  · `passcode = null`, `removePasscode = false` → parol **tegilmaydi**
 *  · `passcode = "1234"`                          → yangi parol o'rnatiladi
 *  · `removePasscode = true`                      → parol olib tashlanadi
 */
@JsonClass(generateAdapter = true)
data class UpdateLessonReq(
    @Json(name = "title") val title: String? = null,
    @Json(name = "description") val description: String? = null,
    @Json(name = "scheduled_at") val scheduledAt: String? = null,
    @Json(name = "duration_min") val durationMin: Int? = null,
    @Json(name = "passcode") val passcode: String? = null,
    @Json(name = "remove_passcode") val removePasscode: Boolean = false,
    @Json(name = "is_locked") val isLocked: Boolean? = null,
    @Json(name = "is_recording_enabled") val isRecordingEnabled: Boolean? = null,
    @Json(name = "is_waiting_room_enabled") val isWaitingRoomEnabled: Boolean? = null,
    /**
     * Ovoz sozlamalari (№11) — jonli darsda ham o'zgartirsa bo'ladi
     * (`api-contract.md`: bayroqni `PATCH` bilan ham yangilash mumkin).
     */
    @Json(name = "mute_on_entry") val muteOnEntry: Boolean? = null,
    @Json(name = "allow_self_unmute") val allowSelfUnmute: Boolean? = null,
)

/**
 * Bildirishnoma — entity.Notification.
 *
 * Aynan shu shakl **ikki yo'ldan** keladi: `GET /notifications` javobida va
 * WebSocket `notification` xabarining `payload` ida (`notification.go:42` —
 * `hub.Send(userID, ws.NewNotificationMsg(userID, n))` butun entity'ni yuboradi).
 * Shu sabab jonli kelgan bildirishnomani ro'yxatga qo'shish uchun qayta so'rov
 * kerak emas.
 */
@JsonClass(generateAdapter = true)
data class Notification(
    @Json(name = "id") val id: String,
    @Json(name = "type") val type: String = "system",
    @Json(name = "title") val title: String = "",
    @Json(name = "body") val body: String = "",
    @Json(name = "lesson_id") val lessonId: String? = null,
    /** `null` — o'qilmagan. Backend `read_at` ni faqat o'qilganda yuboradi. */
    @Json(name = "read_at") val readAt: String? = null,
    @Json(name = "created_at") val createdAt: String? = null,
) {
    val isUnread: Boolean get() = readAt.isNullOrBlank()
}

/** `GET /api/v1/notifications/unread-count` javobi — `{data:{count:N}}`. */
@JsonClass(generateAdapter = true)
data class UnreadCount(
    @Json(name = "count") val count: Int = 0,
)

/**
 * Dars yozuvi — entity.Recording.
 *
 * `object_key` backend'da `json:"-"` — MinIO ichidagi yo'l hech qachon klientga
 * kelmaydi.
 */
@JsonClass(generateAdapter = true)
data class Recording(
    @Json(name = "id") val id: String,
    @Json(name = "lesson_id") val lessonId: String = "",
    /** `recording` | `processing` | `ready` | `failed` (entity/recording.go). */
    @Json(name = "status") val status: String = "processing",
    @Json(name = "duration_sec") val durationSec: Int = 0,
    @Json(name = "size_bytes") val sizeBytes: Long = 0,
    @Json(name = "started_at") val startedAt: String? = null,
    @Json(name = "ended_at") val endedAt: String? = null,
    @Json(name = "created_at") val createdAt: String? = null,
    /**
     * Yozuv qachon avtomatik o'chiriladi (retention — PRODUCT.md №5).
     *
     * Faqat `ready` yozuvda keladi (backend uni `ended_at + RECORDING_RETENTION_DAYS`
     * dan HISOBLAYDI, DB'da saqlamaydi). `expired` bo'lgach maydon umuman
     * qaytmaydi — shuning uchun uning yo'qligi "muddat yo'q" degani, "muddat
     * o'tmagan" degani EMAS.
     */
    @Json(name = "expires_at") val expiresAt: String? = null,
)

/** `POST /recordings/{id}/upload-url` — telefon→MinIO to'g'ridan PUT havolasi. */
@JsonClass(generateAdapter = true)
data class UploadUrl(
    @Json(name = "url") val url: String,
)

/** `POST /recordings/{id}/complete` so'rovi (lokal yozuv yakuni). */
@JsonClass(generateAdapter = true)
data class CompleteRecordingReq(
    @Json(name = "duration_sec") val durationSec: Int,
    @Json(name = "ended_at") val endedAt: String? = null,
)

// ─── Dars arxivi (video + chat + materiallar) — entity.LessonArchive ──────────

/** `GET /api/v1/lessons/{id}/archive` — dars arxivining YAGONA so'rovdagi surati. */
@JsonClass(generateAdapter = true)
data class LessonArchiveDto(
    @Json(name = "lesson") val lesson: Lesson,
    /** Yozuv topilmasa `null` (klient «yozuv yo'q» ko'rsatadi). */
    @Json(name = "recording") val recording: ArchiveRecordingDto? = null,
    /** ESKIDAN YANGIGA tartiblangan, o'chirilganlarsiz. */
    @Json(name = "chat") val chat: List<ArchiveChatMessageDto> = emptyList(),
    @Json(name = "materials") val materials: List<ArchiveMaterialDto> = emptyList(),
)

/** Arxivdagi yozuv — pleyer + holat. `url`/`telegramUrl` presigned/deep-link. */
@JsonClass(generateAdapter = true)
data class ArchiveRecordingDto(
    @Json(name = "id") val id: String,
    @Json(name = "status") val status: String = "processing",
    @Json(name = "duration_sec") val durationSec: Int = 0,
    @Json(name = "size_bytes") val sizeBytes: Long = 0,
    /** Faqat `ready` da to'ladi (presigned). `expired` da `null`. */
    @Json(name = "url") val url: String? = null,
    @Json(name = "expires_at") val expiresAt: String? = null,
    /** Arxiv guruhidagi videoga `t.me/c/...` havola; `null` → tugma ko'rsatilmaydi. */
    @Json(name = "telegram_url") val telegramUrl: String? = null,
)

/** Arxivdagi chat xabari (`ChatFileDto` fayl uchun qayta ishlatiladi). */
@JsonClass(generateAdapter = true)
data class ArchiveChatMessageDto(
    @Json(name = "id") val id: String,
    @Json(name = "sender_identity") val senderIdentity: String = "",
    @Json(name = "sender_name") val senderName: String = "",
    @Json(name = "body") val body: String = "",
    @Json(name = "to_identity") val toIdentity: String? = null,
    @Json(name = "file") val file: ChatFileDto? = null,
    @Json(name = "created_at") val createdAt: String = "",
    /** Dars boshlanishidan necha soniya keyin (web sinxroni uchun; mobil ishlatmaydi). */
    @Json(name = "offset_sec") val offsetSec: Int = 0,
)

/** Arxivda ulashilgan material (chatdagi fayl xabaridan). */
@JsonClass(generateAdapter = true)
data class ArchiveMaterialDto(
    @Json(name = "name") val name: String = "",
    @Json(name = "size") val size: Long = 0,
    @Json(name = "mime") val mime: String = "",
    @Json(name = "url") val url: String = "",
    @Json(name = "created_at") val createdAt: String = "",
)

/**
 * Kutish xonasidagi kirish so'rovi — entity.WaitingRoomRequest (M27).
 *
 * DIQQAT: REST (`GET /lessons/:id/waitingroom`) `id` kaliti bilan keladi,
 * WebSocket xabari esa `request_id` bilan (`waitingroom.go:68`). Ikki nom bitta
 * narsani anglatadi — WS tomoni [uz.darsly.mentor.data.ws.RealtimeParser] da
 * shu turga o'giriladi, ya'ni farq bitta joyda qoladi.
 */
@JsonClass(generateAdapter = true)
data class WaitingRoomRequest(
    @Json(name = "id") val id: String,
    @Json(name = "lesson_id") val lessonId: String = "",
    @Json(name = "requester_name") val requesterName: String = "",
    @Json(name = "status") val status: String = "pending",
    @Json(name = "created_at") val createdAt: String? = null,
)

/**
 * `GET /api/v1/app-config` javobi (M42 · BE-5) — entity.AppConfig.
 * Ochiq endpoint, token talab qilmaydi.
 */
@JsonClass(generateAdapter = true)
data class AppConfig(
    @Json(name = "android") val android: AndroidConfig? = null,
)

/** Bitta platforma uchun versiya siyosati — entity.AppPlatformConfig. */
@JsonClass(generateAdapter = true)
data class AndroidConfig(
    /** Shu versiyadan past klient **ishlamasligi** kerak (semver). */
    @Json(name = "min_version") val minVersion: String = "",
    /** Mavjud eng yangi versiya — yumshoq taklif uchun. */
    @Json(name = "latest_version") val latestVersion: String = "",
    /** APK yuklab olish havolasi. */
    @Json(name = "apk_url") val apkUrl: String = "",
    /** Favqulodda to'xtatish: versiyadan qat'i nazar majburlash. */
    @Json(name = "force_update") val forceUpdate: Boolean = false,
    @Json(name = "release_notes") val releaseNotes: String = "",
)

/** `POST /lessons/:id/token` javobi — entity.RoomToken. */
@JsonClass(generateAdapter = true)
data class RoomToken(
    @Json(name = "token") val token: String,
    @Json(name = "ws_url") val wsUrl: String,
    @Json(name = "room_name") val roomName: String,
    @Json(name = "identity") val identity: String,
    @Json(name = "role") val role: String,
    /**
     * Dars ID'si — xona holati endpointlari (`/rooms/{lessonId}/...`) uchun.
     * Default bo'sh: eski backend bilan ham ishlasin (maydonsiz javob parse
     * bo'ladi va shunchaki holat yuklanmaydi).
     */
    @Json(name = "lesson_id") val lessonId: String = "",
)

/** Xonadagi ishtirokchi (`GET /lessons/{id}/participants`). */
@JsonClass(generateAdapter = true)
data class RoomParticipantDto(
    @Json(name = "identity") val identity: String,
    @Json(name = "name") val name: String = "",
    @Json(name = "active") val active: Boolean = true,
    @Json(name = "audio_muted") val audioMuted: Boolean = true,
    @Json(name = "video_muted") val videoMuted: Boolean = true,
)

/** Dars chati xabari. `toIdentity != null` → shaxsiy. */
@JsonClass(generateAdapter = true)
data class ChatMessageDto(
    @Json(name = "id") val id: String,
    @Json(name = "sender_identity") val senderIdentity: String = "",
    @Json(name = "sender_name") val senderName: String = "",
    @Json(name = "body") val body: String = "",
    @Json(name = "to_identity") val toIdentity: String? = null,
    @Json(name = "created_at") val createdAt: String = "",
    /** Ilova qilingan fayl (`null` — oddiy matnli xabar). */
    @Json(name = "file") val file: ChatFileDto? = null,
)

/**
 * Chat xabariga ilova qilingan fayl — `entity.ChatFile`.
 *
 * ⚠️ [url] **presigned va muddatli** (1 soat) havola: backend uni bazada
 * saqlamaydi, HAR javobda qaytadan imzolaydi. Shu sabab uni keshlab qo'yish
 * mumkin emas — bir soatdan keyin bosilgan havola 403 berardi va bu "fayl
 * yo'qolgan"dek ko'rinardi. Obyekt kaliti (`Key`) esa umuman kelmaydi.
 */
@JsonClass(generateAdapter = true)
data class ChatFileDto(
    @Json(name = "name") val name: String = "",
    @Json(name = "size") val size: Long = 0,
    @Json(name = "mime") val mime: String = "",
    @Json(name = "url") val url: String = "",
    @Json(name = "expires_in_s") val expiresInS: Int = 0,
)

/** Xona chatiga xabar (room-token bilan — host ham shu yo'ldan yuradi). */
@JsonClass(generateAdapter = true)
data class SendRoomChatReq(
    @Json(name = "token") val token: String,
    @Json(name = "body") val body: String,
    @Json(name = "to") val to: String = "",
)

/** Qo'lni tushirish (host). */
@JsonClass(generateAdapter = true)
data class LowerHandReq(
    @Json(name = "identity") val identity: String,
)

/**
 * Emoji reaksiya — `POST /rooms/{lessonID}/reaction` (room-token bilan).
 *
 * Server hech nima SAQLAMAYDI, faqat data-channel orqali tarqatadi. To'plam
 * server tomonda ham qulflangan ([uz.darsly.mentor.ui.room.Reactions.ALLOWED]),
 * ya'ni ro'yxatdan tashqari emoji 400 oladi.
 */
@JsonClass(generateAdapter = true)
data class SendReactionReq(
    @Json(name = "token") val token: String,
    @Json(name = "emoji") val emoji: String,
)

/**
 * So'rovnoma — `entity.Poll`.
 *
 * [resultsVisibility] YARATISHDA tanlanadi va o'zgarmas: `mentor_only` (default)
 * natijani hech qachon o'quvchiga ko'rsatmaydi, `public` esa mentor «E'lon
 * qilish» bosgandan keyin ko'rsatadi ([resultsPublishedAt] to'ladi).
 */
@JsonClass(generateAdapter = true)
data class Poll(
    @Json(name = "id") val id: String,
    @Json(name = "lesson_id") val lessonId: String = "",
    @Json(name = "question") val question: String = "",
    @Json(name = "options") val options: List<String> = emptyList(),
    @Json(name = "is_active") val isActive: Boolean = true,
    @Json(name = "created_at") val createdAt: String? = null,
    @Json(name = "closed_at") val closedAt: String? = null,
    @Json(name = "results_visibility") val resultsVisibility: String = "mentor_only",
    @Json(name = "results_published_at") val resultsPublishedAt: String? = null,
)

/** So'rovnoma natijasi — `entity.PollResults`. [counts] variantlar tartibida. */
@JsonClass(generateAdapter = true)
data class PollResults(
    @Json(name = "poll") val poll: Poll? = null,
    @Json(name = "counts") val counts: List<Int> = emptyList(),
    @Json(name = "total") val total: Int = 0,
)

/**
 * `POST /lessons/{id}/polls` — `entity.CreatePollReq`.
 *
 * Server chegaralari: savol 1..500, variantlar 2..10 ta (har biri 1..200).
 * Ular klientda ham takrorlangan ([uz.darsly.mentor.ui.room.PollForm]) —
 * ustoz xatoni serverga bormasdan, o'zbekcha ko'radi.
 */
@JsonClass(generateAdapter = true)
data class CreatePollReq(
    @Json(name = "question") val question: String,
    @Json(name = "options") val options: List<String>,
    @Json(name = "results_visibility") val resultsVisibility: String,
)

/** `GET /rooms/{lessonId}/state` javobi — ko'tarilgan qo'llar (navbat tartibida). */
@JsonClass(generateAdapter = true)
data class RoomStateResp(
    @Json(name = "hands") val hands: List<RaisedHandDto> = emptyList(),
)

/**
 * Qora ro'yxat yozuvi — `entity.BlocklistEntry` (№4).
 *
 * `mentor_id` backend'da `json:"-"` — kelmaydi (ro'yxat baribir faqat so'rovchi
 * ustozniki). Moslik kaliti — [displayName]: o'quvchida akkaunt yo'q va LiveKit
 * identity har kirishda yangi, shuning uchun ban **ism bo'yicha** (katta-kichik
 * harf farqsiz) ishlaydi. [identity] esa chiqarilgan paytdagi qiymat — audit uchun.
 */
@JsonClass(generateAdapter = true)
data class BlocklistEntry(
    @Json(name = "id") val id: String,
    @Json(name = "identity") val identity: String = "",
    @Json(name = "display_name") val displayName: String = "",
    @Json(name = "created_at") val createdAt: String? = null,
)

@JsonClass(generateAdapter = true)
data class RaisedHandDto(
    @Json(name = "identity") val identity: String,
    @Json(name = "name") val name: String = "",
    @Json(name = "raised_at") val raisedAt: String = "",
)
