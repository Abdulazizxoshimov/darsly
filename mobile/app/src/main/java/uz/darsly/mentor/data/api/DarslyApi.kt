package uz.darsly.mentor.data.api

import retrofit2.Call
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Backend REST kontrakti (docs/api-contract.md).
 * R1 · 1-blok: login/logout → app-config → darslar → host tokeni.
 */
interface DarslyApi {

    @POST("api/v1/auth/login")
    suspend fun login(@Body req: LoginReq): Envelope<TokenPair>

    /**
     * Sessiyani serverda yopadi (M3). Himoyalangan endpoint — `Authorization`
     * header'i kerak, tanasida esa bekor qilinadigan refresh token.
     * 204 qaytaradi, shuning uchun javob tipi yo'q.
     */
    @POST("api/v1/auth/logout")
    suspend fun logout(@Body req: LogoutReq)

    @GET("api/v1/auth/me")
    suspend fun me(): Envelope<User>

    /** Ochiq endpoint (BE-5 · M42) — majburiy yangilanish tekshiruvi. */
    @GET("api/v1/app-config")
    suspend fun appConfig(): Envelope<AppConfig>

    @GET("api/v1/lessons")
    suspend fun lessons(
        @Query("limit") limit: Int = 50,
        @Query("page") page: Int = 1,
    ): ListEnvelope<Lesson>

    /**
     * Yangi dars (M7). Server 201 + `{data: Lesson}` qaytaradi, `join_slug` ichida —
     * ya'ni ulashish havolasi uchun qo'shimcha so'rov kerak emas.
     */
    @POST("api/v1/lessons")
    suspend fun createLesson(@Body req: CreateLessonReq): Envelope<Lesson>

    /** Mentor (host) uchun LiveKit tokeni — xonani ochadi. */
    @POST("api/v1/lessons/{id}/token")
    suspend fun hostToken(@Path("id") lessonId: String): Envelope<RoomToken>

    @POST("api/v1/lessons/{id}/end")
    suspend fun endLesson(@Path("id") lessonId: String)
}

/**
 * Refresh uchun ALOHIDA interfeys (M2).
 *
 * Ataylab `suspend` emas, `Call` — [TokenAuthenticator.authenticate] bloklovchi
 * kontekstda ishlaydi va `execute()` shu yerda eng tabiiy. Bu interfeys
 * **authenticator'siz** klientga ulanadi, aks holda refresh 401 bersa rekursiya
 * bo'lardi.
 */
interface AuthRefreshApi {
    @POST("api/v1/auth/refresh")
    fun refresh(@Body req: RefreshReq): Call<Envelope<TokenPair>>
}
