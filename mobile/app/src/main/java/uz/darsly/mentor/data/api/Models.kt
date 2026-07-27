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

/** `POST /auth/logout` so'rovi (M3) — sessiyani serverda ham yopadi. */
@JsonClass(generateAdapter = true)
data class LogoutReq(
    @Json(name = "refresh_token") val refreshToken: String,
)

@JsonClass(generateAdapter = true)
data class User(
    @Json(name = "id") val id: String,
    @Json(name = "email") val email: String,
    @Json(name = "full_name") val fullName: String,
    @Json(name = "role") val role: String,
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
)
