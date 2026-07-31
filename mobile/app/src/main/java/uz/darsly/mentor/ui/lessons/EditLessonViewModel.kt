package uz.darsly.mentor.ui.lessons

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

data class EditLessonUiState(
    /** Tahrirlanayotgan darsning serverdagi holati — diff shunga nisbatan hisoblanadi. */
    val original: Lesson? = null,
    val input: LessonForm.Input = LessonForm.Input(),
    val errors: LessonForm.Errors = LessonForm.Errors(),
    val showErrors: Boolean = false,
    /** Ustoz "Parolni olib tashlash" ni tanladi (yangi parol maydonini bloklaydi). */
    val removePasscode: Boolean = false,
    val submitting: Boolean = false,
    val deleting: Boolean = false,
    val serverError: String? = null,
    /** `null` bo'lmasa — saqlandi, ekran yopiladi. */
    val saved: Lesson? = null,
    /** `true` — dars o'chirildi, ekran yopiladi. */
    val deleted: Boolean = false,
) {
    /**
     * Saqlash tugmasi holati.
     *
     * Faqat validatsiya emas, **o'zgarish borligi** ham talab qilinadi: hech nima
     * o'zgarmaganda yoqiq turgan tugma bosilsa hech narsa bo'lmasdi va bu buzuq
     * ilova taassurotini berardi.
     */
    val canSubmit: Boolean
        get() = original != null && errors.isValid && !submitting && !deleting && hasChanges

    val hasChanges: Boolean
        get() = original?.let {
            LessonForm.toUpdateRequest(it, input, removePasscode) != null
        } ?: false
}

/**
 * Darsni tahrirlash va o'chirish.
 *
 * Boshlang'ich holat ro'yxatdan **tayyor** [Lesson] sifatida keladi (qo'shimcha
 * `GET /lessons/:id` yo'q): ustoz kartani bosgan zahoti forma to'la ochiladi.
 * Ro'yxat esa `LessonsViewModel.start()` da har safar serverdan yangilanadi,
 * ya'ni ma'lumot eskirmaydi.
 */
@HiltViewModel
class EditLessonViewModel @Inject constructor(
    private val repo: LessonsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(EditLessonUiState())
    val state: StateFlow<EditLessonUiState> = _state.asStateFlow()

    /** Ekran ochilganda bir marta. Takroriy chaqiruv kiritilgan matnni yo'qotmaydi. */
    fun load(lesson: Lesson) {
        if (_state.value.original?.id == lesson.id) return
        val input = LessonForm.fromLesson(lesson)
        _state.value = EditLessonUiState(
            original = lesson,
            input = input,
            errors = LessonForm.validate(input),
        )
    }

    fun edit(transform: (LessonForm.Input) -> LessonForm.Input) {
        _state.update { st ->
            val input = transform(st.input)
            st.copy(input = input, errors = LessonForm.validate(input), serverError = null)
        }
    }

    /**
     * Parolni olib tashlash tanlovi. Yoqilganda kiritilgan yangi parol tozalanadi —
     * ikkalasi bir vaqtda ma'noga ega emas va backend'da `RemovePasscode` baribir
     * ustunlik qiladi (`lesson.go:119-128`), ya'ni yozilgan parol jimgina
     * yo'qolardi.
     */
    fun setRemovePasscode(value: Boolean) {
        _state.update { st ->
            val input = if (value) st.input.copy(passcode = "") else st.input
            st.copy(removePasscode = value, input = input, errors = LessonForm.validate(input))
        }
    }

    fun submit() {
        val current = _state.value
        val original = current.original ?: return
        if (current.submitting || current.deleting) return

        val errors = LessonForm.validate(current.input)
        if (!errors.isValid) {
            _state.update { it.copy(errors = errors, showErrors = true) }
            return
        }
        val req = LessonForm.toUpdateRequest(original, current.input, current.removePasscode)
        if (req == null) {
            // Bu yerga UI odatda yo'l qo'ymaydi (tugma o'chiq), lekin poyga bo'lsa —
            // tarmoqqa chiqmaymiz va "saqlandi" deb yolg'on ham aytmaymiz.
            _state.update { it.copy(serverError = "O'zgarish kiritilmadi") }
            return
        }

        _state.update { it.copy(submitting = true, showErrors = true, serverError = null) }
        viewModelScope.launch {
            repo.update(original.id, req)
                .onSuccess { lesson -> _state.update { it.copy(submitting = false, saved = lesson) } }
                .onFailure { t ->
                    _state.update { it.copy(submitting = false, serverError = ApiErrors.humanError(t)) }
                }
        }
    }

    /** O'chirish. Tasdiq dialogi ekranda — bu yerga faqat tasdiqlangandan keyin kelinadi. */
    fun delete() {
        val current = _state.value
        val original = current.original ?: return
        if (current.submitting || current.deleting) return

        _state.update { it.copy(deleting = true, serverError = null) }
        viewModelScope.launch {
            repo.delete(original.id)
                .onSuccess { _state.update { it.copy(deleting = false, deleted = true) } }
                .onFailure { t ->
                    _state.update { it.copy(deleting = false, serverError = ApiErrors.humanError(t)) }
                }
        }
    }

    /** Ekran yopilgach — bir xil VM boshqa dars uchun qayta ishlatilsa toza boshlansin. */
    fun reset() {
        _state.value = EditLessonUiState()
    }
}
