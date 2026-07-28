package uz.darsly.mentor.ui.room

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.WaitingRoomRequest
import uz.darsly.mentor.data.repo.WaitingRoomRepository
import uz.darsly.mentor.data.ws.Realtime
import uz.darsly.mentor.data.ws.RealtimeClient
import uz.darsly.mentor.data.ws.RealtimeEvent

data class WaitingRoomUiState(
    val lessonId: String = "",
    val requests: List<WaitingRoomRequest> = emptyList(),
    /** Hozir qaror yuborilayotgan so'rovlar — tugmalar bloklanadi. */
    val deciding: Set<String> = emptySet(),
    val notice: String? = null,
) {
    val count: Int get() = requests.size
    val isEmpty: Boolean get() = requests.isEmpty()

    fun isDeciding(requestId: String): Boolean = requestId in deciding
}

/**
 * Kutish xonasi — ustoz tomoni (M27).
 *
 * ## Nega xona ekrani ichida
 * Kutish xonasi — dars **davomidagi** amal: o'quvchi eshikni taqillatadi, ustoz
 * kiritadi. Alohida ekran bo'lsa ustoz uni ochib turishga majbur bo'lardi va
 * dars sahnasini ko'rmasdi. Zoom ham xuddi shunday: xona ichidagi qalqib
 * chiquvchi panel.
 *
 * ## Ikki manba, bitta ro'yxat
 * Ekran ochilganda REST'dan surat olinadi, so'ng WebSocket jonli qo'shadi.
 * Ikkalasining birlashishi — [WaitingRoomState] da, sof funksiyalarda va
 * testlar ostida (dublikat, boshqa darsning so'rovi, tartib).
 */
class WaitingRoomViewModel(
    private val repo: WaitingRoomRepository = WaitingRoomRepository.create(),
    private val realtime: RealtimeClient = Realtime.client,
) : ViewModel() {

    private val _state = MutableStateFlow(WaitingRoomUiState())
    val state: StateFlow<WaitingRoomUiState> = _state.asStateFlow()

    init {
        viewModelScope.launch {
            realtime.events.collect { event ->
                if (event is RealtimeEvent.WaitingRoomRequest) onLiveRequest(event)
            }
        }
    }

    /** Xonaga ulanilgach chaqiriladi. Bir xil dars uchun takroriy chaqiruv zararsiz. */
    fun start(lessonId: String) {
        if (_state.value.lessonId == lessonId) return
        _state.value = WaitingRoomUiState(lessonId = lessonId)
        loadSnapshot()
    }

    /**
     * Serverdagi kutayotganlar suratini oladi.
     *
     * Xato JIM o'tkaziladi — ataylab. Bu ekranning asosiy vazifasi dars o'tish;
     * kutish xonasi ro'yxati yuklanmagani uchun ustozga xato dialogi ko'rsatish
     * darsni buzardi. WebSocket baribir jonli so'rovlarni olib keladi, va
     * backend mentor ulanganda kutayotganlarni **qayta yuboradi**
     * (`DeliverPendingSnapshot`) — ya'ni bu so'rov yiqilsa ham ro'yxat to'ladi.
     */
    private fun loadSnapshot() {
        val lessonId = _state.value.lessonId
        if (lessonId.isBlank()) return
        viewModelScope.launch {
            repo.pending(lessonId).onSuccess { snapshot ->
                _state.update { st ->
                    if (st.lessonId != lessonId) return@update st // dars almashdi
                    st.copy(
                        requests = WaitingRoomState.sortForDisplay(
                            WaitingRoomState.merge(snapshot, st.requests),
                        ),
                    )
                }
            }
        }
    }

    private fun onLiveRequest(event: RealtimeEvent.WaitingRoomRequest) {
        val lessonId = _state.value.lessonId
        if (lessonId.isBlank()) return
        val incoming = WaitingRoomState.toRequest(event)
        _state.update { st ->
            st.copy(
                requests = WaitingRoomState.sortForDisplay(
                    WaitingRoomState.add(st.requests, incoming, lessonId),
                ),
            )
        }
    }

    /**
     * O'quvchini kiritish.
     *
     * So'rov muvaffaqiyatli bo'lsa (yoki 409 — allaqachon hal qilingan bo'lsa,
     * repozitoriy izohiga qara) qator ro'yxatdan olib tashlanadi. Xatoda qator
     * **qoladi** va sabab aytiladi: ustoz qayta urina olishi kerak, o'quvchi esa
     * jimgina eshik ortida qolib ketmasligi kerak.
     */
    fun admit(request: WaitingRoomRequest) = decide(request, admitting = true)

    fun reject(request: WaitingRoomRequest) = decide(request, admitting = false)

    private fun decide(request: WaitingRoomRequest, admitting: Boolean) {
        if (_state.value.isDeciding(request.id)) return
        _state.update { it.copy(deciding = it.deciding + request.id) }

        viewModelScope.launch {
            val result = if (admitting) repo.admit(request.id) else repo.reject(request.id)
            result
                .onSuccess {
                    _state.update {
                        it.copy(
                            requests = WaitingRoomState.remove(it.requests, request.id),
                            deciding = it.deciding - request.id,
                            notice = if (admitting) {
                                "${request.requesterName} darsga qo'shildi"
                            } else {
                                "${request.requesterName} rad etildi"
                            },
                        )
                    }
                }
                .onFailure { t ->
                    _state.update {
                        it.copy(
                            deciding = it.deciding - request.id,
                            notice = ApiErrors.humanError(t),
                        )
                    }
                }
        }
    }

    /** "Hammasini qabul qilish" — Zoom'dagi "Admit all". */
    fun admitAll() {
        val pending = _state.value.requests
        if (pending.isEmpty()) return
        // Har birini alohida yuboramiz: backend'da ommaviy endpoint yo'q va
        // qismi yiqilsa qolgani baribir kiritilishi kerak (hammasi-yoki-hech
        // nima bu yerda foydalanuvchiga zarar).
        pending.forEach { admit(it) }
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }
}
