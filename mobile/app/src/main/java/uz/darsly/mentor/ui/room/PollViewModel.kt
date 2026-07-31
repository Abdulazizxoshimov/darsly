package uz.darsly.mentor.ui.room

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.Poll
import uz.darsly.mentor.data.api.PollResults
import uz.darsly.mentor.data.repo.PollsRepository

data class PollUiState(
    val lessonId: String = "",
    val loading: Boolean = false,
    val polls: List<Poll> = emptyList(),
    /** So'rovnoma ID → oxirgi ma'lum natija. */
    val results: Map<String, PollResults> = emptyMap(),
    /** Hozir amal bajarilayotgan so'rovnoma (tugmalar bloklanadi). */
    val busyId: String? = null,
    val creating: Boolean = false,
    val error: String? = null,
    val notice: String? = null,
)

/**
 * Dars ichidagi so'rovnomalar paneli (№7).
 *
 * ## Nega alohida ViewModel
 * `RoomViewModel` allaqachon eng murakkab sinf (LiveKit, MediaProjection,
 * foreground servis, chat, moderatsiya). So'rovnoma esa mustaqil: u xona
 * media holatiga umuman bog'liq emas va faqat REST bilan ishlaydi.
 * `WaitingRoomViewModel` ham xuddi shu sababdan ajratilgan.
 *
 * ## Nega natija uchun ROOM-TOKEN kerak
 * `GET /polls/{id}/results` ochiq endpoint va "xonada bo'lganlik isboti"ni
 * LiveKit tokeni orqali talab qiladi (JWT emas). Mentor host-tokeni bilan
 * doim 200 oladi — shuning uchun panel jonli natijani ko'rsata oladi va
 * buning uchun so'rovnomani yopish shart emas.
 */
@HiltViewModel
class PollViewModel @Inject constructor(
    private val repo: PollsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(PollUiState())
    val state: StateFlow<PollUiState> = _state.asStateFlow()

    /** Xona tokeni — natijalarni o'qish uchun (`null` bo'lsa natija so'ralmaydi). */
    private var roomToken: String? = null

    fun start(lessonId: String, roomToken: String?) {
        this.roomToken = roomToken?.takeIf { it.isNotBlank() }
        if (_state.value.lessonId == lessonId && _state.value.polls.isNotEmpty()) {
            refresh()
            return
        }
        _state.value = PollUiState(lessonId = lessonId)
        refresh()
    }

    fun refresh() {
        val lessonId = _state.value.lessonId
        if (lessonId.isBlank() || _state.value.loading) return
        _state.update { it.copy(loading = true, error = null) }
        viewModelScope.launch {
            repo.list(lessonId)
                .onSuccess { items ->
                    _state.update {
                        it.copy(loading = false, polls = PollForm.sortForDisplay(items), error = null)
                    }
                    // Natijalar ALOHIDA so'rovlar bilan: ro'yxat endpointi
                    // ovozlarni bermaydi (u faqat so'rovnomalarni qaytaradi).
                    items.forEach { loadResults(it.id) }
                }
                .onFailure { t ->
                    _state.update { it.copy(loading = false, error = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * Jonli natija. Xato JIM yutiladi: natija — qo'shimcha ma'lumot, uning
     * yo'qligi panelni ishlatib bo'lmaydigan qilmasligi kerak (ro'yxat va
     * tugmalar baribir ishlaydi).
     */
    private fun loadResults(pollId: String) {
        val token = roomToken ?: return
        viewModelScope.launch {
            repo.results(pollId, token).onSuccess { res -> putResults(pollId, res) }
        }
    }

    fun create(question: String, options: List<String>, publicResults: Boolean) {
        val lessonId = _state.value.lessonId
        if (lessonId.isBlank() || _state.value.creating) return
        if (!PollForm.canSubmit(question, options)) return
        _state.update { it.copy(creating = true) }
        viewModelScope.launch {
            repo.create(lessonId, PollForm.request(question, options, publicResults))
                .onSuccess { poll ->
                    _state.update {
                        it.copy(
                            creating = false,
                            polls = PollForm.sortForDisplay(listOf(poll) + it.polls.filterNot { p -> p.id == poll.id }),
                            notice = "So'rovnoma yuborildi",
                        )
                    }
                }
                .onFailure { t ->
                    _state.update { it.copy(creating = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * Natijani o'quvchilarga ochish.
     *
     * `mentor_only` so'rovnomada server 400 beradi — UI bu tugmani o'sha
     * holatda umuman ko'rsatmaydi ([PollForm.canPublish]), lekin himoya shu
     * yerda ham qoladi: panel eski ma'lumot bilan ochilib qolishi mumkin.
     */
    fun publish(poll: Poll) {
        val lessonId = _state.value.lessonId
        if (lessonId.isBlank() || _state.value.busyId != null) return
        if (!PollForm.canPublish(poll)) return
        _state.update { it.copy(busyId = poll.id) }
        viewModelScope.launch {
            repo.publish(lessonId, poll.id)
                .onSuccess { res ->
                    applyResults(poll.id, res, "Natija e'lon qilindi")
                }
                .onFailure { t ->
                    _state.update { it.copy(busyId = null, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /** Ovoz berishni to'xtatadi. Natijani OCHMAYDI — bu alohida amal. */
    fun close(poll: Poll) {
        if (_state.value.busyId != null || !PollForm.canClose(poll)) return
        _state.update { it.copy(busyId = poll.id) }
        viewModelScope.launch {
            repo.close(poll.id)
                .onSuccess { res -> applyResults(poll.id, res, "So'rovnoma yopildi") }
                .onFailure { t ->
                    _state.update { it.copy(busyId = null, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }

    /** Javobdagi so'rovnoma nusxasi ham yangilanadi (`is_active`, e'lon vaqti). */
    private fun applyResults(pollId: String, res: PollResults, notice: String) {
        _state.update { st ->
            val fresh = res.poll
            st.copy(
                busyId = null,
                notice = notice,
                polls = if (fresh == null) st.polls else {
                    PollForm.sortForDisplay(st.polls.map { if (it.id == pollId) fresh else it })
                },
                results = st.results + (pollId to res),
            )
        }
    }

    private fun putResults(pollId: String, res: PollResults) {
        _state.update { it.copy(results = it.results + (pollId to res)) }
    }
}
