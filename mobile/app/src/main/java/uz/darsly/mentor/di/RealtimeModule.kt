package uz.darsly.mentor.di

import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.components.SingletonComponent
import uz.darsly.mentor.data.api.DarslyApi
import uz.darsly.mentor.data.api.SessionManager
import uz.darsly.mentor.data.ws.RealtimeClient
import javax.inject.Singleton

/**
 * Real-time (WebSocket) kanal.
 *
 * ## Nega `@Provides`, `@Inject constructor` emas
 *
 * [RealtimeClient] konstruktorida to'rtta default parametr bor (`client`,
 * `parser`, `baseUrl`, `sleep`) va ularning hech biri bog'liqlik emas — ular
 * TEST uchun almashtiriladigan nuqtalar (masalan `sleep` haqiqiy kutishni
 * qisqartiradi). Hilt `@Inject constructor` da default qiymatlarni e'tiborsiz
 * qoldirib har birini graph'dan yechishga urinardi va build yiqilardi.
 *
 * ## Nega bitta nusxa
 *
 * Soket ekranga emas, **sessiyaga** bog'langan: ustoz darslar ro'yxatidan
 * xonaga o'tganda kanal uzilmasligi kerak, aks holda o'sha lahzada kelgan
 * kutish so'rovi yo'qolardi.
 */
@Module
@InstallIn(SingletonComponent::class)
object RealtimeModule {

    @Provides
    @Singleton
    fun realtimeClient(session: SessionManager, api: DarslyApi): RealtimeClient =
        RealtimeClient(
            accessToken = { session.accessToken },
            // Oddiy authed so'rov — OkHttp authenticator'i uni ushlab refresh qiladi.
            refreshToken = { api.me() },
        )
}
