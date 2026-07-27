// security-crypto 1.1.0: `MasterKey` va `EncryptedSharedPreferences` deprecated deb
// belgilangan (Google to'g'ridan-to'g'ri Keystore'ni tavsiya qiladi), lekin bu reliz
// BARQAROR va hozircha eng ishonchli yechim. Sabab va almashtirish rejasi: README §9.1.
@file:Suppress("DEPRECATION")

package uz.darsly.mentor.data.store

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey
import io.livekit.android.util.LKLog
import java.security.KeyStore

/**
 * Shifrlangan token saqlagichini yasaydi (M1).
 *
 * Kalit **Android Keystore** ichida (`AES256_GCM` master key), qiymatlar
 * `EncryptedSharedPreferences` bilan shifrlanadi. Ya'ni root'siz qurilmada boshqa
 * ilova ham, `adb backup` ham tokenni o'qiy olmaydi.
 *
 * ⚠️ MA'LUM XATARLAR VA ULARNI QOPLASH
 *  1. `androidx.security:security-crypto` API'lari 1.1.0-beta01 dan boshlab
 *     **deprecated** (Google to'g'ridan-to'g'ri Keystore ishlatishni tavsiya qiladi).
 *     Shunga qaramay 1.1.0 — **barqaror** reliz va hozircha sanoat standarti;
 *     almashtirish keyingi bloklarga rejalashtirilgan (`mobile/README.md §9.1`).
 *  2. Ba'zi OEM (Samsung/Huawei/Xiaomi) qurilmalarida Keystore buzilib qolishi
 *     ma'lum muammo: `EncryptedSharedPreferences.create` `InvalidProtocolBufferException`
 *     yoki `KeyStoreException` beradi va ilova **ishga tushmayoq crash** bo'ladi.
 *     Shuning uchun quyida: xato → keyset+fayl o'chiriladi → BIR marta qayta urinish →
 *     u ham bo'lmasa xotiradagi saqlagich (ilova ishlaydi, faqat sessiya saqlanmaydi).
 *     Ustozni dars oldida crash bilan qoldirishdan ko'ra qayta login qildirish yaxshiroq.
 */
object SecureTokenStore {

    private const val PREFS_NAME = "darsly_session"

    /** `MasterKey.DEFAULT_MASTER_KEY_ALIAS` — recovery uchun bizga aniq nom kerak. */
    private const val MASTER_KEY_ALIAS = "_androidx_security_master_key_"

    fun create(context: Context): TokenStore {
        val prefs = openEncrypted(context, firstAttempt = true)
            ?: run {
                // Buzilgan holatni tozalab bir marta qayta urinamiz.
                wipe(context)
                openEncrypted(context, firstAttempt = false)
            }
        return if (prefs != null) {
            PrefsTokenStore(prefs)
        } else {
            LKLog.e { "Shifrlangan saqlagich yasalmadi — sessiya faqat xotirada qoladi" }
            InMemoryTokenStore()
        }
    }

    private fun openEncrypted(context: Context, firstAttempt: Boolean): SharedPreferences? =
        try {
            val masterKey = MasterKey.Builder(context, MASTER_KEY_ALIAS)
                .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                .build()
            EncryptedSharedPreferences.create(
                context,
                PREFS_NAME,
                masterKey,
                EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM,
            )
        } catch (t: Throwable) {
            // Ataylab `Throwable`: Tink `GeneralSecurityException`, `IOException`,
            // OEM'larda esa `IllegalStateException`/`ProviderException` ham uchraydi.
            LKLog.e(t) { "EncryptedSharedPreferences ochilmadi (firstAttempt=$firstAttempt)" }
            null
        }

    /** Buzilgan keyset va preferens faylini o'chiradi (sessiya yo'qoladi — qayta login). */
    private fun wipe(context: Context) {
        runCatching { context.deleteSharedPreferences(PREFS_NAME) }
            .onFailure { LKLog.w(it) { "preferens fayli o'chirilmadi" } }
        runCatching {
            KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
                .deleteEntry(MASTER_KEY_ALIAS)
        }.onFailure { LKLog.w(it) { "master key o'chirilmadi" } }
    }
}
