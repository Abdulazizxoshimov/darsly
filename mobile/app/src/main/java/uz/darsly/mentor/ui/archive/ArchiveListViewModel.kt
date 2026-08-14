package uz.darsly.mentor.ui.archive

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
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.repo.LessonsRepository

data class ArchiveListUiState(
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    /** Faqat YAKUNLANGAN darslar, eng yangisidan boshlab. */
    val lessons: List<Lesson> = emptyList(),
    val error: String? = null,
    val offline: Boolean = false,
) {
    val isEmpty: Boolean get() = lessons.isEmpty()
}

/**
 * Arxiv ro'yxati — yakunlangan darslar (video + chatga kirish nuqtasi).
 *
 * Jadval bilan bir xil naqsh: alohida endpoint yo'q, `GET /lessons` barcha
 * darslarni beradi, `ended` filtri esa sof klient mantiqi (kesh ham umumiy —
 * offline'da ham ishlaydi, ro'yxat bilan hech qachon zid bo'lmaydi).
 */
@HiltViewModel
class ArchiveListViewModel @Inject constructor(
    private val repo: LessonsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(ArchiveListUiState())
    val state: StateFlow<ArchiveListUiState> = _state.asStateFlow()

    fun start() {
        if (_state.value.loading || _state.value.refreshing) return
        viewModelScope.launch {
            if (_state.value.lessons.isEmpty()) {
                repo.cached()?.let { cached -> _state.update { it.copy(lessons = endedOf(cached.lessons)) } }
            }
            refresh(userInitiated = false)
        }
    }

    fun refresh(userInitiated: Boolean = true) {
        val cur = _state.value
        if (cur.loading || cur.refreshing) return
        val has = !cur.isEmpty
        _state.update { it.copy(loading = !has, refreshing = has && userInitiated, error = null) }
        viewModelScope.launch {
            repo.refresh()
                .onSuccess { page ->
                    _state.update {
                        it.copy(loading = false, refreshing = false, lessons = endedOf(page.lessons), error = null, offline = false)
                    }
                }
                .onFailure { t ->
                    val offline = LessonsRepository.isOffline(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && !it.isEmpty,
                            error = if (it.isEmpty) ApiErrors.humanError(t) else null,
                        )
                    }
                }
        }
    }

    /** Faqat `ended`, eng yangi sana yuqorida (ISO satrlar leksikografik = xronologik). */
    private fun endedOf(all: List<Lesson>): List<Lesson> =
        all.filter { it.status == "ended" }
            .sortedByDescending { it.scheduledAt ?: it.createdAt ?: "" }
}
