package uz.darsly.mentor.ui.notifications

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.Notification
import uz.darsly.mentor.data.repo.NotificationsBadge
import uz.darsly.mentor.data.repo.NotificationsRepository
import uz.darsly.mentor.data.ws.Realtime
import uz.darsly.mentor.data.ws.RealtimeClient
import uz.darsly.mentor.data.ws.RealtimeEvent
import uz.darsly.mentor.util.NotificationFormat

data class NotificationsUiState(
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    val items: List<Notification> = emptyList(),
    val error: String? = null,
    val offline: Boolean = false,
    val markingAll: Boolean = false,
    val notice: String? = null,
    /**
     * Serverdan kamida bir marta muvaffaqiyatli o'qildimi.
     *
     * Nishon sonini shu bayroqsiz sinxronlash bo'lmaydi: ViewModel yaratilgan
     * lahzada ro'yxat bo'sh, ya'ni "0 o'qilmagan" bo'lib ko'rinadi va ilova
     * darajasidagi to'g'ri sonni **o'chirib** yuborardi — nishon bir zumga
     * yo'qolib, keyin qayta paydo bo'lardi.
     */
    val loaded: Boolean = false,
) {
    val unread: Int get() = NotificationFormat.unreadCount(items)
}

/**
 * Bildirishnomalar ro'yxati.
 *
 * ## Nima o'zgardi
 * Real-time kanal ([RealtimeClient]) R1'dan beri ishlab turibdi va `notification`
 * hodisalarini qabul qilardi — lekin **hech kim ularni ko'rsatmasdi**: xabar
 * kelardi va jimgina yo'qolardi. Bu ekran ikkalasini birlashtiradi: server
 * tarixini beradi va uning ustiga jonli xabarlarni qo'shadi.
 *
 * ## Nega WS obunasi ekran umriga bog'langan
 * Kanalning o'zi `DarslyApp` da, sessiya bilan yashaydi. Bu yerdagi obuna esa
 * faqat ro'yxatni yangilash uchun — ekran yopilganda `viewModelScope` bilan
 * birga bekor qilinadi va soket ochiq qolaveradi (xona ekrani undan foydalanadi).
 */
class NotificationsViewModel(
    private val repo: NotificationsRepository = NotificationsRepository.create(),
    realtime: RealtimeClient = Realtime.client,
) : ViewModel() {

    private val _state = MutableStateFlow(NotificationsUiState())
    val state: StateFlow<NotificationsUiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            realtime.events.collect { event ->
                if (event is RealtimeEvent.Notification) onRealtime(event)
            }
        }
        // Nishon soni ro'yxat holatidan ANIQ hisoblanadi va `DarslyApp` ning
        // taxminiy `increment()` ini to'g'rilaydi. Bitta yozuv joyi bo'lgani
        // uchun "belgiladim, nishon turibdi" holati bo'lishi mumkin emas.
        viewModelScope.launch {
            _state.collect { if (it.loaded) NotificationsBadge.set(it.unread) }
        }
    }

    fun start() {
        if (_state.value.loading || _state.value.refreshing) return
        refresh(userInitiated = false)
    }

    fun refresh(userInitiated: Boolean = true) {
        val current = _state.value
        if (current.loading || current.refreshing) return
        val hasContent = current.items.isNotEmpty()
        _state.update {
            it.copy(loading = !hasContent, refreshing = hasContent && userInitiated, error = null)
        }
        viewModelScope.launch {
            repo.refresh()
                .onSuccess { page ->
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            items = page.items,
                            error = null,
                            offline = false,
                            loaded = true,
                            notice = if (page.truncated) {
                                "Juda ko'p bildirishnoma — oxirgi ${page.items.size} tasi ko'rsatildi"
                            } else {
                                it.notice
                            },
                        )
                    }
                }
                .onFailure { t ->
                    val offline = NotificationsRepository.isOffline(t)
                    val message = ApiErrors.humanError(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && it.items.isNotEmpty(),
                            // Ro'yxat bor ekan — uni xato ekrani bilan almashtirmaymiz.
                            error = if (it.items.isEmpty()) message else null,
                            notice = if (it.items.isEmpty()) null else message,
                        )
                    }
                }
        }
    }

    /**
     * Jonli kelgan bildirishnoma.
     *
     * Payload to'liq modelga aylantirilsa (`id` bor) ro'yxatga **darhol**
     * qo'shiladi — server so'rovisiz. Aylantirib bo'lmasa (eski server yoki
     * kutilmagan shakl) ro'yxat serverdan yangilanadi: "bildirishnoma keldi"
     * fakti baribir yo'qolmasligi kerak.
     */
    private fun onRealtime(event: RealtimeEvent.Notification) {
        val model = event.toModel()
        if (model == null) {
            refresh(userInitiated = false)
            return
        }
        _state.update { it.copy(items = NotificationFormat.merge(it.items, model)) }
    }

    /**
     * O'qilgan deb belgilash — **optimistik**.
     *
     * Ro'yxat darhol yangilanadi, so'rov fonda ketadi. Xato bo'lsa belgi qaytariladi
     * va sabab aytiladi: jimgina qaytarish "bosdim, hech nima bo'lmadi" degan
     * chalkashlik tug'diradi.
     */
    fun markRead(item: Notification) {
        if (!item.isUnread) return
        val readNow = nowIso()
        _state.update { st ->
            st.copy(items = st.items.map { if (it.id == item.id) it.copy(readAt = readNow) else it })
        }
        viewModelScope.launch {
            repo.markRead(item.id).onFailure { t ->
                _state.update { st ->
                    st.copy(
                        items = st.items.map { if (it.id == item.id) it.copy(readAt = null) else it },
                        notice = ApiErrors.humanError(t),
                    )
                }
            }
        }
    }

    /**
     * Hammasini o'qilgan deb belgilash.
     *
     * Bu yerda optimistik yangilash ATAYLAB ishlatilmadi: amal ko'p yozuvga
     * tegadi va qaytarish uchun har birining oldingi `read_at` ini eslab qolish
     * kerak bo'lardi. Buning o'rniga tugma "band" holatga o'tadi va muvaffaqiyatda
     * ro'yxat serverdan qayta olinadi — haqiqat manbai bitta qoladi.
     */
    fun markAllRead() {
        val current = _state.value
        if (current.markingAll || current.unread == 0) return
        _state.update { it.copy(markingAll = true) }
        viewModelScope.launch {
            repo.markAllRead()
                .onSuccess {
                    _state.update { it.copy(markingAll = false, notice = "Hammasi o'qilgan deb belgilandi") }
                    refresh(userInitiated = false)
                }
                .onFailure { t ->
                    _state.update { it.copy(markingAll = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }

    /**
     * Optimistik belgi uchun vaqt tamg'asi.
     *
     * Aniq qiymat muhim emas — UI faqat "bo'shmi yoki yo'qmi" ni tekshiradi
     * ([Notification.isUnread]). Keyingi yangilashda server qiymati keladi.
     */
    private fun nowIso(): String = java.time.Instant.now().toString()
}
