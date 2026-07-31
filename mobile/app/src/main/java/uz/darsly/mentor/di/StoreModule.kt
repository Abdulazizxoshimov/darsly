package uz.darsly.mentor.di

import android.content.Context
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import uz.darsly.mentor.data.repo.ChatAttachments
import uz.darsly.mentor.data.repo.ContentChatAttachments
import uz.darsly.mentor.data.store.LessonsCache
import uz.darsly.mentor.data.store.PrefsLessonsCache
import uz.darsly.mentor.data.store.SecureTokenStore
import uz.darsly.mentor.data.store.TokenStore
import uz.darsly.mentor.data.store.PrefsUiPrefs
import uz.darsly.mentor.data.store.UiPrefs
import javax.inject.Singleton

/**
 * Diskda saqlanadigan narsalar: token, darslar keshi, UI sozlamalari.
 *
 * ## Nega Hilt kerak bo'ldi
 *
 * Avval bular `object Session`, `PrefsLessonsCache.create(ctx)` kabi global
 * nuqtalar orqali olinardi. Natijada ViewModel'da bog'liqlikni ALMASHTIRIB
 * bo'lmasdi va 37 test faylida **bitta ham ViewModel testi yo'q edi** —
 * bu tasodif emas, arxitektura oqibati.
 *
 * `Session` da hatto `installForTest()` teshigi ochilgan edi: global holatni
 * test uchun almashtirishga urinish — DI yo'qligining klassik alomati.
 */
@Module
@InstallIn(SingletonComponent::class)
object StoreModule {

    /**
     * Token saqlagichi — Android Keystore bilan shifrlangan.
     *
     * `SecureTokenStore.create` OEM Keystore nosozliklarini o'zi qoplaydi
     * (buzilsa tozalab bir marta qayta urinadi, u ham bo'lmasa xotiradagi
     * saqlagich). Shu sabab bu yerda qo'shimcha himoya kerak emas.
     */
    @Provides
    @Singleton
    fun tokenStore(@ApplicationContext ctx: Context): TokenStore =
        SecureTokenStore.create(ctx)

    @Provides
    @Singleton
    fun lessonsCache(@ApplicationContext ctx: Context): LessonsCache =
        PrefsLessonsCache.create(ctx)

    @Provides
    @Singleton
    fun uiPrefs(@ApplicationContext ctx: Context): UiPrefs = PrefsUiPrefs.create(ctx)

    /**
     * Chatga biriktiriladigan fayllarni `content://` URI'dan o'qiydi (№15).
     *
     * Interfeys sifatida beriladi — implementatsiya `ContentResolver` ga
     * bog'langan va JVM testida ishlamaydi; repozitoriyning HTTP tomoni esa
     * sinaladi (`ChatPollRepositoryTest`).
     */
    @Provides
    @Singleton
    fun chatAttachments(@ApplicationContext ctx: Context): ChatAttachments =
        ContentChatAttachments(ctx)
}
