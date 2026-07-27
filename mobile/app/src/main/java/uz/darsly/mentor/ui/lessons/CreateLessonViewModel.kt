package uz.darsly.mentor.ui.lessons

import android.app.Application
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import uz.darsly.mentor.data.api.ApiErrors
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.data.repo.LessonsRepository

data class CreateLessonUiState(
    val input: LessonForm.Input = LessonForm.Input(),
    // 🟢I: boshlang'ich xatolar HAM hisoblanadi. Avval bo'sh `Errors()` turardi va
    // `isValid` `true` bo'lib chiqardi — ya'ni mutlaqo bo'sh formada "Yaratish"
    // tugmasi yoqiq ko'rinardi (bosilganda faqat xato chiqardi).
    val errors: LessonForm.Errors = LessonForm.validate(input),
    /**
     * Xatolarni qachon ko'rsatish: forma ochilishi bilan hamma maydonni qizil
     * qilib qo'yish tajovuzkor ko'rinadi — "Yaratish" bosilgandan keyin ko'rsatamiz.
     */
    val showErrors: Boolean = false,
    val submitting: Boolean = false,
    /** Server rad etgan bo'lsa (masalan 429 yoki validatsiya) — o'zbekcha matn. */
    val serverError: String? = null,
    /** `null` bo'lmasa — yaratildi, ekran yopiladi. */
    val created: Lesson? = null,
)

/** Dars yaratish formasi (M7). */
class CreateLessonViewModel @JvmOverloads constructor(
    app: Application,
    // @JvmOverloads — `viewModel()` fabrikasi `(Application)` konstruktorini qidiradi.
    private val repo: LessonsRepository = LessonsRepository.create(app),
) : AndroidViewModel(app) {

    private val _state = MutableStateFlow(CreateLessonUiState())
    val state: StateFlow<CreateLessonUiState> = _state.asStateFlow()

    fun edit(transform: (LessonForm.Input) -> LessonForm.Input) {
        _state.update { st ->
            val input = transform(st.input)
            st.copy(
                input = input,
                // Xatolar HAR o'zgarishda qayta hisoblanadi: foydalanuvchi tuzatishi
                // bilan qizil matn darhol yo'qoladi (tugma ham darhol yoqiladi).
                errors = LessonForm.validate(input),
                serverError = null,
            )
        }
    }

    fun submit() {
        val current = _state.value
        if (current.submitting) return
        val errors = LessonForm.validate(current.input)
        if (!errors.isValid) {
            _state.update { it.copy(errors = errors, showErrors = true) }
            return
        }
        _state.update { it.copy(submitting = true, showErrors = true, serverError = null) }
        viewModelScope.launch {
            repo.create(LessonForm.toRequest(current.input))
                .onSuccess { lesson -> _state.update { it.copy(submitting = false, created = lesson) } }
                .onFailure { t ->
                    _state.update {
                        it.copy(submitting = false, serverError = ApiErrors.humanError(t))
                    }
                }
        }
    }

    /** Ekran yopilgandan keyin — bir xil VM qayta ishlatilsa eski natija qolmasin. */
    fun reset() {
        _state.value = CreateLessonUiState()
    }
}
