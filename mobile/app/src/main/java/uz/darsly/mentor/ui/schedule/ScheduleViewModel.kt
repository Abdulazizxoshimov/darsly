package uz.darsly.mentor.ui.schedule

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
import uz.darsly.mentor.util.ScheduleFormat
import java.time.LocalDate

data class ScheduleUiState(
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    /** Kelajakdagi kunlar (o'sish tartibida). */
    val days: List<ScheduleFormat.Day> = emptyList(),
    /** Vaqti belgilanmagan darslar — pastdagi alohida bo'lim. */
    val undated: List<Lesson> = emptyList(),
    /** O'tgan darslarni ham ko'rsatishmi (default — faqat kelajak). */
    val includePast: Boolean = false,
    val error: String? = null,
    val offline: Boolean = false,
    val notice: String? = null,
) {
    val isEmpty: Boolean get() = days.isEmpty() && undated.isEmpty()
}

/**
 * Jadval — darslar **vaqt bo'yicha**, kunlarga bo'lingan holda.
 *
 * Bir xil `LessonsRepository` dan foydalanadi (kesh ham umumiy): jadval uchun
 * alohida endpoint yo'q va kerak ham emas — backend `GET /lessons` da barcha
 * darslarni beradi, guruhlash esa sof klient mantiqi
 * ([uz.darsly.mentor.util.ScheduleFormat]). Shu tufayli darslar ro'yxati va
 * jadval **hech qachon bir-biriga zid** ma'lumot ko'rsatmaydi va offline'da
 * ikkalasi ham ishlaydi.
 */
@HiltViewModel
class ScheduleViewModel @Inject constructor(
    private val repo: LessonsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(ScheduleUiState())
    val state: StateFlow<ScheduleUiState> = _state.asStateFlow()

    /** Oxirgi ma'lum ro'yxat — `includePast` o'zgarganda qayta guruhlash uchun. */
    private var lessons: List<Lesson> = emptyList()

    fun start() {
        if (_state.value.loading || _state.value.refreshing) return
        // Kesh o'qish diskka tegadi — korutinada (`LessonsViewModel` bilan bir xil sabab).
        viewModelScope.launch {
            if (lessons.isEmpty()) {
                repo.cached()?.let { cached ->
                    lessons = cached.lessons
                    regroup()
                }
            }
            refresh(userInitiated = false)
        }
    }

    fun refresh(userInitiated: Boolean = true) {
        val current = _state.value
        if (current.loading || current.refreshing) return
        val hasContent = !current.isEmpty
        _state.update {
            it.copy(loading = !hasContent, refreshing = hasContent && userInitiated, error = null)
        }
        viewModelScope.launch {
            repo.refresh()
                .onSuccess { page ->
                    lessons = page.lessons
                    _state.update { it.copy(loading = false, refreshing = false, error = null, offline = false) }
                    regroup()
                }
                .onFailure { t ->
                    val offline = LessonsRepository.isOffline(t)
                    val message = ApiErrors.humanError(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && !it.isEmpty,
                            error = if (it.isEmpty) message else null,
                            notice = if (it.isEmpty) null else message,
                        )
                    }
                }
        }
    }

    /**
     * "O'tgan darslarni ko'rsatish" almashtirgichi.
     *
     * Tarmoqqa chiqmaydi — bir xil ma'lumot boshqacha guruhlanadi. Ustoz
     * o'tgan haftadagi darsni topmoqchi bo'lganda yangi so'rov kutmasligi kerak.
     */
    fun setIncludePast(value: Boolean) {
        if (_state.value.includePast == value) return
        _state.update { it.copy(includePast = value) }
        regroup()
    }

    private fun regroup() {
        val includePast = _state.value.includePast
        val days = ScheduleFormat.groupByDay(
            lessons = lessons,
            fromDate = if (includePast) null else LocalDate.now(),
        )
        // `includePast` IKKALA bo'limga ham uzatiladi: avval faqat sanali
        // bo'limga berilardi va tumbler vaqtsiz darslarga ta'sir qilmasdi.
        _state.update {
            it.copy(days = days, undated = ScheduleFormat.undated(lessons, includePast))
        }
    }

    fun noticeShown() = _state.update { it.copy(notice = null) }

    /** Ekran hodisalari (masalan, dars rejalashtirildi) uchun qisqa xabar. */
    fun showNotice(text: String) = _state.update { it.copy(notice = text) }
}
