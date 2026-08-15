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
import uz.darsly.mentor.util.LessonFormat

data class LessonsUiState(
    /** Birinchi yuklash — ekranda hali hech narsa yo'q (kesh ham bo'sh). */
    val loading: Boolean = false,
    /** Pull-to-refresh aylanasi — ro'yxat ko'rinib turadi. */
    val refreshing: Boolean = false,
    val lessons: List<Lesson> = emptyList(),
    /** To'liq ekranli xato — FAQAT ko'rsatadigan hech narsa bo'lmaganda. */
    val error: String? = null,
    /** Tarmoq yo'q va ekrandagi ro'yxat keshdan (B-4 belgisi). */
    val offline: Boolean = false,
    /** Kesh qachon saqlangan (epoch millis, 0 = noma'lum). */
    val cachedAtMillis: Long = 0L,
    /** Bir martalik xabar (snackbar): "yangilanmadi", "dars yaratildi"… */
    val notice: String? = null,
    /** Hozir o'chirilayotgan darsning ID'si (kartada spinner uchun). */
    val deletingId: String? = null,
)

/**
 * Darslar ro'yxati (M6).
 *
 * OQIM: kesh → ekran (bir zumda) → tarmoq → ekran + kesh.
 * Tarmoq yiqilsa keshdagi ro'yxat **qoladi** va yuqorida "internet yo'q" chizig'i chiqadi:
 * ustoz koridorda ro'yxatni ko'ra olishi kerak.
 */
@HiltViewModel
class LessonsViewModel @Inject constructor(
    private val repo: LessonsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(LessonsUiState())
    val state: StateFlow<LessonsUiState> = _state.asStateFlow()

    /**
     * Ekran ochilganda: keshni ko'rsatib, so'ng serverdan **jimgina** yangilaydi.
     *
     * Darsdan qaytganda ham chaqiriladi (ekran qayta kompozitsiyaga kiradi) —
     * shu tufayli yakunlangan dars ro'yxatda `Tugagan` bo'lib ko'rinadi, ustoz
     * buning uchun pastga tortishi shart emas.
     */
    fun start() {
        if (_state.value.loading || _state.value.refreshing) return
        // Kesh o'qish DISKKA tegadi (shifr ochish + JSON parse) — shuning uchun
        // korutinada. Avval u asosiy oqimda bajarilardi va ko'p darsli ustozda
        // ekran ochilishida qotib qolish (ANR) xavfi bor edi.
        viewModelScope.launch {
            if (_state.value.lessons.isEmpty()) {
                repo.cached()?.let { cached ->
                    _state.update {
                        it.copy(
                            lessons = LessonFormat.sortForDisplay(cached.lessons),
                            cachedAtMillis = cached.savedAtMillis,
                        )
                    }
                }
            }
            refresh(userInitiated = false)
        }
    }

    /** Pull-to-refresh va "Qayta urinish" tugmasi. */
    fun refresh(userInitiated: Boolean = true) {
        val current = _state.value
        if (current.loading || current.refreshing) return
        val hasContent = current.lessons.isNotEmpty()
        _state.update {
            it.copy(
                loading = !hasContent,
                refreshing = hasContent && userInitiated,
                error = null,
            )
        }
        viewModelScope.launch {
            repo.refresh()
                .onSuccess { page ->
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            lessons = LessonFormat.sortForDisplay(page.lessons),
                            error = null,
                            offline = false,
                            cachedAtMillis = System.currentTimeMillis(),
                            // 🟡E: chegara oshib ketgan bo'lsa jim qolmaymiz.
                            notice = if (page.truncated) {
                                "Juda ko'p dars — birinchi ${page.lessons.size} tasi ko'rsatildi"
                            } else {
                                it.notice
                            },
                        )
                    }
                }
                .onFailure { t ->
                    val offline = LessonsRepository.isOffline(t)
                    val message = ApiErrors.humanError(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && it.lessons.isNotEmpty(),
                            // Keshdagi ro'yxat bor ekan — uni xato ekrani bilan
                            // almashtirmaymiz, faqat qisqa xabar beramiz.
                            error = if (it.lessons.isEmpty()) message else null,
                            notice = if (it.lessons.isEmpty()) null else message,
                        )
                    }
                }
        }
    }

    /** M7: yaratilgan dars ro'yxat boshiga qo'yiladi — server javobini kutmasdan. */
    fun onLessonCreated(lesson: Lesson) {
        _state.update {
            it.copy(
                lessons = LessonFormat.sortForDisplay(
                    listOf(lesson) + it.lessons.filterNot { l -> l.id == lesson.id },
                ),
                notice = "Dars yaratildi",
            )
        }
        refresh(userInitiated = false)
    }

    /**
     * Dars tahrirlandi — ro'yxatdagi nusxa almashtiriladi.
     *
     * Serverdan qayta so'ralmaydi: `PATCH` javobi darsning **to'liq** yangi
     * holati (`hs.Success(c, lesson)`), ya'ni qo'shimcha so'rov faqat kechikish
     * qo'shardi. Tartib qayta hisoblanadi — vaqt o'zgargan bo'lsa dars boshqa
     * bo'limga ko'chishi kerak.
     */
    fun onLessonUpdated(lesson: Lesson) {
        _state.update { st ->
            st.copy(
                lessons = LessonFormat.sortForDisplay(
                    st.lessons.map { if (it.id == lesson.id) lesson else it },
                ),
                notice = "Dars saqlandi",
            )
        }
    }

    /** Dars o'chirildi — ro'yxatdan olib tashlanadi (repozitoriy keshni ham yangilagan). */
    fun onLessonDeleted(lesson: Lesson) {
        _state.update { st ->
            st.copy(
                lessons = st.lessons.filterNot { it.id == lesson.id },
                notice = "Dars o'chirildi",
            )
        }
    }

    /**
     * Darsni o'chirish (3-nuqta menyusidan). Tasdiqlash EKRANDA (LessonCard) —
     * bu yerga faqat tasdiqdan keyin keladi. Muvaffaqiyatda ro'yxatdan olinadi.
     */
    fun delete(lesson: Lesson) {
        if (_state.value.deletingId != null) return
        _state.update { it.copy(deletingId = lesson.id) }
        viewModelScope.launch {
            val result = repo.delete(lesson.id)
            _state.update { it.copy(deletingId = null) }
            if (result.isSuccess) {
                onLessonDeleted(lesson)
            } else {
                showNotice("Darsni o'chirib bo'lmadi — qaytadan urinib ko'ring")
            }
        }
    }

    /** Ekran tomonidan yuboriladigan qisqa xabar ("Havola nusxalandi"). */
    fun showNotice(message: String) = _state.update { it.copy(notice = message) }

    fun noticeShown() = _state.update { it.copy(notice = null) }

    // M3 "Chiqish" bu yerdan SHAXSIY KABINETGA ko'chirildi
    // (`ui/profile/ProfileViewModel.logout`). Sabab: chiqish — sozlama, kunlik
    // amal emas; darslar ekranining sarlavhasida turgani uchun tasodifan
    // bosilardi. Kesh tozalash mantiqi o'zgarmadi — u `Session.loggedIn == false`
    // signali bo'yicha `DarslyApp.observeLogoutCleanup()` da markazlashtirilgan (🟡B).
}
