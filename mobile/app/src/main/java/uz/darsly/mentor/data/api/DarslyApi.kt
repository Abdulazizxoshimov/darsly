package uz.darsly.mentor.data.api

import okhttp3.MultipartBody
import okhttp3.RequestBody
import retrofit2.Call
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Part
import retrofit2.http.Path
import retrofit2.http.Query

/**
 * Backend REST kontrakti (docs/api-contract.md).
 *
 * Har bir yo'l `backend/api/router.go` dan **ko'chirilgan**, taxmin qilinmagan;
 * javob shakllari `api/http_status/response.go` konvensiyasiga bo'ysunadi:
 * bitta obyekt → [Envelope], sahifalangan ro'yxat → [ListEnvelope], 204 → tipsiz.
 *
 * ⚠️ `hs.Success(c, items)` bilan qaytariladigan ro'yxatlar (kutish xonasi,
 * yozuvlar) `total` MAYDONISIZ keladi — ular `Envelope<List<T>>`, `ListEnvelope`
 * emas. Bu farq backend handlerlarida ko'rinadi va shu yerda saqlanadi.
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

    /**
     * Darsni tahrirlash. **PATCH** — ya'ni faqat yuborilgan maydonlar o'zgaradi
     * (`UpdateLessonReq` da `null` bo'lganlari JSON'ga tushmaydi).
     */
    @PATCH("api/v1/lessons/{id}")
    suspend fun updateLesson(
        @Path("id") lessonId: String,
        @Body req: UpdateLessonReq,
    ): Envelope<Lesson>

    /** Darsni o'chirish (backend soft-delete qiladi). 204 qaytaradi. */
    @DELETE("api/v1/lessons/{id}")
    suspend fun deleteLesson(@Path("id") lessonId: String)

    /** Mentor (host) uchun LiveKit tokeni — xonani ochadi. */
    @POST("api/v1/lessons/{id}/token")
    suspend fun hostToken(@Path("id") lessonId: String): Envelope<RoomToken>

    /**
     * Xona holati — ko'tarilgan qo'llar. Room-token bilan (JWT emas), chunki
     * bu endpoint xonadagi HAR QANDAY ishtirokchi uchun ochiq.
     *
     * Kech ulangan (yoki qayta ulangan) ustoz mavjud navbatni shu bilan tiklaydi —
     * aks holda u "hech kim qo'l ko'tarmagan" degan noto'g'ri manzarani ko'rardi.
     */
    @GET("api/v1/rooms/{lessonId}/state")
    suspend fun roomState(
        @Path("lessonId") lessonId: String,
        @Query("token") token: String,
    ): Envelope<RoomStateResp>

    // ── Xona moderatsiyasi (host, JWT) ──────────────────────────────────────
    // Mobil ustoz shu paytgacha bularni faqat webda qila olardi: telefondan dars
    // o'tayotgan ustoz shovqin qilayotgan o'quvchini mute ham qila olmasdi.

    @GET("api/v1/lessons/{id}/participants")
    suspend fun participants(@Path("id") lessonId: String): ListEnvelope<RoomParticipantDto>

    @POST("api/v1/lessons/{id}/mute-all")
    suspend fun muteAll(
        @Path("id") lessonId: String,
        @Body body: MuteAllReq,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/participants/{identity}/mute")
    suspend fun muteParticipant(
        @Path("id") lessonId: String,
        @Path("identity") identity: String,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/participants/{identity}/remove")
    suspend fun removeParticipant(
        @Path("id") lessonId: String,
        @Path("identity") identity: String,
        @Body body: RemoveParticipantReq,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/participants/{identity}/allow-speak")
    suspend fun allowSpeak(
        @Path("id") lessonId: String,
        @Path("identity") identity: String,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/participants/{identity}/revoke-speak")
    suspend fun revokeSpeak(
        @Path("id") lessonId: String,
        @Path("identity") identity: String,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/hands/lower")
    suspend fun lowerHand(
        @Path("id") lessonId: String,
        @Body body: LowerHandReq,
    ): retrofit2.Response<Unit>

    @POST("api/v1/lessons/{id}/hands/lower-all")
    suspend fun lowerAllHands(@Path("id") lessonId: String): retrofit2.Response<Unit>

    // ── Chat (room-token — host ham, o'quvchi ham ayni yo'ldan) ──────────────

    @GET("api/v1/rooms/{lessonId}/chat")
    suspend fun roomChat(
        @Path("lessonId") lessonId: String,
        @Query("token") token: String,
    ): ListEnvelope<ChatMessageDto>

    @POST("api/v1/rooms/{lessonId}/chat")
    suspend fun sendRoomChat(
        @Path("lessonId") lessonId: String,
        @Body body: SendRoomChatReq,
    ): Envelope<ChatMessageDto>

    /**
     * Xabarni o'chirish — FAQAT dars egasi (moderatsiya, №6).
     *
     * Yumshoq o'chirish: qator DB'da qoladi (`deleted_at`/`deleted_by` — iz),
     * lekin hech bir tarix so'rovida qaytmaydi va qabrtosh («xabar o'chirilgan»)
     * QOLDIRILMAYDI. Jonli xonadagilar `chat_deleted` hodisasini oladi.
     *
     * Takroriy o'chirish → **404** (atomik `WHERE deleted_at IS NULL`), begona
     * mentor → 403. Shu sabab `Response<Unit>`: 204 tanasiz keladi.
     */
    @DELETE("api/v1/lessons/{id}/chat/{messageId}")
    suspend fun deleteChatMessage(
        @Path("id") lessonId: String,
        @Path("messageId") messageId: String,
    ): retrofit2.Response<Unit>

    /**
     * Fayl ulashish (№15) — `multipart/form-data`, maydon nomi **`file`**.
     *
     * JWT yo'li (mentor): tokenni interceptor qo'yadi, query'da token kerak emas.
     * Server cheklovlari: 20 MB, kengaytma allowlist + mazmun sniff, identity
     * bo'yicha daqiqasiga 5 ta. Klient yuborgan `Content-Type` ISHONCHSIZ deb
     * qaraladi — MinIO'ga serverning kanonik MIME'i yoziladi.
     */
    @Multipart
    @POST("api/v1/lessons/{id}/chat/upload")
    suspend fun uploadChatFile(
        @Path("id") lessonId: String,
        @Part file: MultipartBody.Part,
        @Part("body") body: RequestBody?,
        @Part("to") to: RequestBody?,
    ): Envelope<ChatMessageDto>

    /**
     * Emoji reaksiya (№14) — room-token bilan, chunki endpoint xonadagi HAR
     * QANDAY ishtirokchi uchun (ustoz ham shu yo'ldan yuradi).
     *
     * Server saqlamaydi, faqat data-channel orqali tarqatadi; ruxsatsiz emoji
     * 400, 10 soniyada 5 tadan ko'p bo'lsa 429.
     */
    @POST("api/v1/rooms/{lessonId}/reaction")
    suspend fun sendReaction(
        @Path("lessonId") lessonId: String,
        @Body body: SendReactionReq,
    ): retrofit2.Response<Unit>

    // ── So'rovnoma (host — JWT) ─────────────────────────────────────────────

    /** `hs.Success(items)` — sahifalanmagan ro'yxat, `total` YO'Q. */
    @GET("api/v1/lessons/{id}/polls")
    suspend fun polls(@Path("id") lessonId: String): Envelope<List<Poll>>

    @POST("api/v1/lessons/{id}/polls")
    suspend fun createPoll(
        @Path("id") lessonId: String,
        @Body req: CreatePollReq,
    ): Envelope<Poll>

    /**
     * Natijani o'quvchilarga ochadi va xonaga `poll_published` yuboradi.
     *
     * DIQQAT: `results_visibility=mentor_only` so'rovnomada **400** — bu rejim
     * yaratishda tanlangan va o'zgarmas. Idempotent: takroriy bosish e'lon
     * vaqtini surmaydi.
     */
    @POST("api/v1/lessons/{id}/polls/{pollId}/publish")
    suspend fun publishPoll(
        @Path("id") lessonId: String,
        @Path("pollId") pollId: String,
    ): Envelope<PollResults>

    /** Ovoz berishni to'xtatadi. **Yopish ≠ e'lon qilish** — natija ochilmaydi. */
    @POST("api/v1/polls/{id}/close")
    suspend fun closePoll(@Path("id") pollId: String): Envelope<PollResults>

    /**
     * Natijalar — room-token bilan (ochiq endpoint, JWT emas).
     *
     * Mentor (host token) DOIM 200 oladi; o'quvchi esa faqat `public` +
     * e'lon qilingan bo'lsa. Shuning uchun ustoz jonli natijani shu yerdan
     * ko'radi va buning uchun so'rovnomani yopishi shart emas.
     */
    @GET("api/v1/polls/{id}/results")
    suspend fun pollResults(
        @Path("id") pollId: String,
        @Query("token") token: String,
    ): Envelope<PollResults>

    @POST("api/v1/lessons/{id}/end")
    suspend fun endLesson(@Path("id") lessonId: String)

    // ─── Shaxsiy kabinet ──────────────────────────────────────────────────────

    @GET("api/v1/users/me")
    suspend fun profile(): Envelope<User>

    @PUT("api/v1/users/me")
    suspend fun updateProfile(@Body req: UpdateProfileReq): Envelope<User>

    /**
     * Parolni o'zgartirish. 204 qaytaradi.
     *
     * DIQQAT: backend bu yerda sessiyalarni bekor QILMAYDI (faqat
     * `ResetPassword` qiladi) — ustoz dars o'rtasida chiqib ketmaydi.
     */
    @PUT("api/v1/users/me/password")
    suspend fun changePassword(@Body req: ChangePasswordReq)

    // ─── Parolni tiklash (ochiq endpointlar) ──────────────────────────────────

    /**
     * Tiklash havolasini emailga yuboradi. Backend **har doim 204** qaytaradi —
     * email ro'yxatdan o'tganmi yoki yo'qmi, farqi yo'q (`auth.go:196` — hisob
     * borligini aniqlash oracle'i bo'lmasligi uchun). Ilova ham shunga mos
     * "agar bunday email bo'lsa, xat yuborildi" deydi.
     */
    @POST("api/v1/auth/forgot-password")
    suspend fun forgotPassword(@Body req: ForgotPasswordReq)

    /** Emaildagi token bilan yangi parol o'rnatish. 204; barcha sessiyalar o'ladi. */
    @POST("api/v1/auth/reset-password")
    suspend fun resetPassword(@Body req: ResetPasswordReq)

    // ─── Bildirishnomalar ─────────────────────────────────────────────────────

    /** Sahifalangan ro'yxat (`hs.List`). [unread] `true` — faqat o'qilmaganlar. */
    @GET("api/v1/notifications")
    suspend fun notifications(
        @Query("limit") limit: Int = 50,
        @Query("page") page: Int = 1,
        @Query("unread") unread: Boolean? = null,
    ): ListEnvelope<Notification>

    @GET("api/v1/notifications/unread-count")
    suspend fun unreadCount(): Envelope<UnreadCount>

    @POST("api/v1/notifications/{id}/read")
    suspend fun markNotificationRead(@Path("id") id: String)

    @POST("api/v1/notifications/read-all")
    suspend fun markAllNotificationsRead()

    // ─── Yozuvlar (Egress → MinIO) ────────────────────────────────────────────

    /** `hs.Success(items)` — sahifalanmagan ro'yxat, `total` yo'q. */
    @GET("api/v1/lessons/{id}/recordings")
    suspend fun recordings(@Path("id") lessonId: String): Envelope<List<Recording>>

    /** Yozib olishni boshlaydi — 201 + yaratilgan yozuv. */
    @POST("api/v1/lessons/{id}/recording/start")
    suspend fun startRecording(@Path("id") lessonId: String): Envelope<Recording>

    /** To'xtatadi (204). Status `processing` ga o'tadi, keyin webhook `ready` qiladi. */
    @POST("api/v1/recordings/{id}/stop")
    suspend fun stopRecording(@Path("id") recordingId: String)

    /** Vaqtinchalik (presigned) yuklab olish havolasi. */
    @GET("api/v1/recordings/{id}/download")
    suspend fun recordingDownload(@Path("id") recordingId: String): Envelope<RecordingDownload>

    // ─── Kutish xonasi (M27) ──────────────────────────────────────────────────

    /** Shu darsning kutayotgan so'rovlari (`hs.Success` — `total` yo'q). */
    @GET("api/v1/lessons/{id}/waitingroom")
    suspend fun waitingRoom(@Path("id") lessonId: String): Envelope<List<WaitingRoomRequest>>

    /**
     * So'rovni qabul qilish. Javobda **guest** tokeni keladi (u WS orqali
     * o'quvchiga ham yuboriladi) — ustozga kerak emas, shuning uchun
     * [uz.darsly.mentor.data.repo.WaitingRoomRepository] uni tashlaydi.
     *
     * DIQQAT: ikkinchi marta admit qilinsa server **409** beradi (atomik
     * `TransitionFromPending`) — bu xato emas, "allaqachon hal qilingan".
     */
    @POST("api/v1/waitingroom/{id}/admit")
    suspend fun admitWaitingRoom(@Path("id") requestId: String): Envelope<RoomToken>

    @POST("api/v1/waitingroom/{id}/reject")
    suspend fun rejectWaitingRoom(@Path("id") requestId: String)
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
