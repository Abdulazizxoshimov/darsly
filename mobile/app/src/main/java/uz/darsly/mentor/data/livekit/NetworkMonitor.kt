package uz.darsly.mentor.data.livekit

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import kotlinx.coroutines.flow.distinctUntilChanged

/**
 * Transport oqimining manbai — `ReconnectController` uchun almashtiriladigan nuqta.
 * Amalda [NetworkMonitor.transports], testda qo'lda boshqariladigan oqim.
 */
fun interface TransportSource {
    fun transports(): Flow<Transport>
}

/**
 * Qurilmaning joriy transportini (Wi-Fi / uyali) kuzatadi.
 *
 * Faqat "qaysi turdagi tarmoq" savoliga javob beradi — qaror
 * [NetworkSwitchPolicy] da (u sof va test ostida). Bu ajratish ataylab:
 * `ConnectivityManager` ni JVM testida ishlatib bo'lmaydi, qaror esa aynan
 * sinovga muhtoj qism.
 *
 * `ACCESS_NETWORK_STATE` ruxsati manifestda allaqachon bor.
 */
object NetworkMonitor {

    /**
     * Transport oqimi. Faqat O'ZGARISHLAR chiqadi (`distinctUntilChanged`):
     * `ConnectivityManager` bir xil holat uchun ham bir necha callback berishi
     * mumkin va ularning har birida qayta ulanish "davolash" emas, buzish bo'lardi.
     */
    fun transports(ctx: Context): Flow<Transport> = callbackFlow {
        val cm = ctx.getSystemService(ConnectivityManager::class.java)
        if (cm == null) {
            trySend(Transport.OTHER)
            awaitClose { }
            return@callbackFlow
        }

        fun emitFor(caps: NetworkCapabilities?) {
            trySend(caps.toTransport())
        }

        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onCapabilitiesChanged(network: Network, caps: NetworkCapabilities) {
                emitFor(caps)
            }

            override fun onLost(network: Network) {
                // DEFAULT tarmoq yo'qoldi. Boshqasi bor bo'lsa tizim darhol
                // `onCapabilitiesChanged` beradi va NONE ustiga yozilib ketadi.
                trySend(Transport.NONE)
            }
        }

        // Boshlang'ich qiymat: callback faqat o'zgarishda keladi, biz esa
        // "hozir qaysi tarmoqdamiz" ni darhol bilishimiz kerak.
        emitFor(cm.getNetworkCapabilities(cm.activeNetwork))

        // ⚠️ `registerDefaultNetworkCallback`, `registerNetworkCallback` EMAS.
        //
        // Qurilma sinovida (2026-07-28) aniqlandi: Wi-Fi va LTE bir vaqtda yoqiq
        // bo'lsa umumiy `registerNetworkCallback` IKKALA tarmoq uchun ham signal
        // beradi. Natijada oqim WIFI ↔ CELLULAR bo'lib tebranadi, `distinctUntilChanged`
        // esa buni haqiqiy almashuv deb o'tkazadi — va dars hech qanday sabab
        // yo'qligiga qaramay MAJBURIY qayta ulanadi. Ekranda ham "Mobil internetga
        // o'tildi" degan yolg'on xabar chiqadi.
        //
        // Bizga faqat DEFAULT tarmoq kerak: trafik qaysi interfeysdan ketayotgani.
        cm.registerDefaultNetworkCallback(callback)

        awaitClose { runCatching { cm.unregisterNetworkCallback(callback) } }
    }.distinctUntilChanged()

    /**
     * ⚠️ TASDIQLANMAGAN tarmoq `NONE` deb qaraladi.
     *
     * Qurilma sinovida (2026-07-28, LTE) aniqlangan poyga: transport o'zgarishi
     * signali tarmoq HALI ISHLAMAYOTGAN paytda keladi. O'sha lahzada qayta
     * ulanishga urinilsa DNS yiqiladi:
     *   `UnknownHostException: Unable to resolve host "livekit...sslip.io"`
     * va dars o'lik qoladi (bir necha soniyadan keyin o'sha host bemalol
     * yechilsa ham).
     *
     * `NET_CAPABILITY_VALIDATED` — Android'ning "bu tarmoqdan internetga
     * haqiqatan chiqish mumkin" tasdig'i. Faqat shundan keyin almashuv deb
     * hisoblaymiz: bu kutish emas, TO'G'RI paytni tanlash.
     */
    private fun NetworkCapabilities?.toTransport(): Transport = when {
        this == null -> Transport.NONE
        !hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED) -> Transport.NONE
        hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> Transport.WIFI
        hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> Transport.CELLULAR
        else -> Transport.OTHER
    }
}
