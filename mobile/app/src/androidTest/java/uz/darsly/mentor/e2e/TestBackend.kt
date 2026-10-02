package uz.darsly.mentor.e2e

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import uz.darsly.mentor.BuildConfig
import java.util.concurrent.TimeUnit

/**
 * To'liq-flow E2E uchun real backend yordamchisi (BuildConfig.API_BASE_URL).
 * Seed admin orqali test-mentor yaratadi — web `e2e-live/fixtures.js` bilan bir naqsh.
 * Diqqat: «bitta akkaunt = bitta sessiya» — mentor API-tokeni UI-logindan OLDIN
 * ishlatiladi va keyin tashlanadi.
 */
object TestBackend {
    private val base = BuildConfig.API_BASE_URL.trimEnd('/')
    private val json = "application/json; charset=utf-8".toMediaType()
    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .build()

    data class Creds(val email: String, val password: String, val id: String)

    private fun post(path: String, body: JSONObject, token: String? = null): JSONObject {
        val req = Request.Builder()
            .url("$base$path")
            .post(body.toString().toRequestBody(json))
            .apply { if (token != null) header("Authorization", "Bearer $token") }
            .build()
        http.newCall(req).execute().use { resp ->
            val text = resp.body?.string().orEmpty()
            check(resp.code in 200..299) { "POST $path -> ${resp.code}: ${text.take(200)}" }
            return if (text.isBlank()) JSONObject() else JSONObject(text)
        }
    }

    fun login(email: String, password: String): String {
        val r = post("/api/v1/auth/login", JSONObject().put("email", email).put("password", password))
        return r.getJSONObject("data").getString("access_token")
    }

    fun adminToken(): String = login("admin@darsly.uz", "Admin12345")

    /** Unikal mentor hisobi (admin API orqali). */
    fun createMentor(): Creds {
        val email = "e2e+${System.currentTimeMillis().toString(36)}@darsly.uz"
        val password = "parol12345"
        val r = post(
            "/api/v1/users",
            JSONObject()
                .put("email", email)
                .put("password", password)
                .put("full_name", "E2E Mentor")
                .put("role", "mentor"),
            adminToken(),
        )
        return Creds(email, password, r.getJSONObject("data").getString("id"))
    }

    /** Mentor nomidan dars seed qiladi (UI-logindan OLDIN chaqirilsin). */
    fun seedLesson(creds: Creds, title: String): JSONObject {
        val tok = login(creds.email, creds.password)
        val r = post(
            "/api/v1/lessons",
            JSONObject()
                .put("title", title)
                .put("duration_min", 30)
                .put("is_recording_enabled", false)
                .put("is_waiting_room_enabled", false),
            tok,
        )
        return r.getJSONObject("data")
    }
}
