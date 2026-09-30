package uz.darsly.mentor.di

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import okhttp3.OkHttpClient
import uz.darsly.mentor.data.livekit.LessonSessionFactory
import uz.darsly.mentor.data.livekit.LiveKitLessonSession
import uz.darsly.mentor.data.livekit.LocalRecorder
import uz.darsly.mentor.data.livekit.NetworkMonitor
import uz.darsly.mentor.data.livekit.RecorderFactory
import uz.darsly.mentor.data.livekit.TransportSource
import uz.darsly.mentor.service.AndroidLessonPlatform
import uz.darsly.mentor.service.LessonPlatform
import java.util.concurrent.TimeUnit
import javax.inject.Qualifier
import javax.inject.Singleton

/**
 * Ilova umri qamrovi — ekran yopilsa ham tugashi SHART bo'lgan ishlar uchun:
 * sessiya bo'shatish, yozuvni yuklash. `GlobalScope` o'rniga (u bekor
 * qilinmaydi, kuzatilmaydi va testda almashtirilmaydi).
 */
@Qualifier
@Retention(AnnotationRetention.BINARY)
annotation class ApplicationScope

/**
 * Presigned PUT (MinIO) uchun klient: `Authorization` header'i YO'Q (imzoni
 * buzadi) va uzun yozish muddati (~1 GB fayl sekin tarmoqda).
 */
@Qualifier
@Retention(AnnotationRetention.BINARY)
annotation class Upload

/**
 * Jonli dars: sessiya egasi, tizim chegarasi, yozuvchi va tarmoq kuzatuvi.
 *
 * Bu yerdagi har bir bog'lanish JVM testida soxta bilan almashtiriladi —
 * `LessonSessionStore`, `RoomViewModel`, `ScreenShareController` va
 * `ReconnectController` endi LiveKit/Android'siz sinaladi.
 */
@Module
@InstallIn(SingletonComponent::class)
object LessonModule {

    @Provides
    @Singleton
    @ApplicationScope
    fun applicationScope(): CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    @Provides
    @Singleton
    fun lessonPlatform(@ApplicationContext ctx: Context): LessonPlatform = AndroidLessonPlatform(ctx)

    @Provides
    @Singleton
    fun recorderFactory(): RecorderFactory = RecorderFactory { file -> LocalRecorder(file) }

    @Provides
    @Singleton
    fun lessonSessionFactory(
        @ApplicationContext ctx: Context,
        recorders: RecorderFactory,
    ): LessonSessionFactory = LessonSessionFactory { lessonId, token ->
        LiveKitLessonSession(ctx, lessonId, token, recorders)
    }

    @Provides
    @Singleton
    fun transportSource(@ApplicationContext ctx: Context): TransportSource =
        TransportSource { NetworkMonitor.transports(ctx) }

    /**
     * M9: avval `LocalRecordingRepository` o'zining TO'RTINCHI `OkHttpClient` ini
     * yasardi (alohida ulanish havzasi va dispatcher). Endi havza va dispatcher
     * `@Bare` klient bilan UMUMIY; interceptor'lar esa ATAYLAB yo'q — presigned
     * URL MinIO hostiga qaraydi va dev-server override'i uni buzardi.
     */
    @Provides
    @Singleton
    @Upload
    fun uploadClient(@Bare bare: OkHttpClient): OkHttpClient = OkHttpClient.Builder()
        .connectionPool(bare.connectionPool)
        .dispatcher(bare.dispatcher)
        .connectTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(60, TimeUnit.MINUTES)
        .build()
}
