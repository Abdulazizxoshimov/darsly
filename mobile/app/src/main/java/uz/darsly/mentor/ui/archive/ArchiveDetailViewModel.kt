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
import uz.darsly.mentor.data.api.ArchiveChatMessageDto
import uz.darsly.mentor.data.api.LessonArchiveDto
import uz.darsly.mentor.data.repo.ArchiveRepository

data class ArchiveDetailUiState(
    val lessonId: String = "",
    val lessonTitle: String = "",
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    val archive: LessonArchiveDto? = null,
    val error: String? = null,
    val offline: Boolean = false,
    /** Chat qidiruvi (bo'sh → barcha xabarlar). */
    val chatQuery: String = "",
    /** Tashqarida ochilishi kerak havola (yuklab olish / Telegram) — ekran iste'mol qiladi. */
    val openUrl: String? = null,
) {
    /** Qidiruvga mos chat xabarlari (matn, yuboruvchi yoki fayl nomi bo'yicha). */
    val filteredChat: List<ArchiveChatMessageDto>
        get() {
            val q = chatQuery.trim()
            val all = archive?.chat.orEmpty()
            if (q.isBlank()) return all
            return all.filter {
                it.body.contains(q, ignoreCase = true) ||
                    it.senderName.contains(q, ignoreCase = true) ||
                    (it.file?.name?.contains(q, ignoreCase = true) == true)
            }
        }
}

/**
 * Dars arxivi detali — video (ExoPlayer) + Telegram-uslub chat paneli.
 *
 * VM faqat MA'LUMOTni boshqaradi (arxivni yuklash + chat qidiruvi + tashqi
 * ochish hodisalari). To'liq-ekran va pleyer tezligi — sof UI holati, ekranda
 * (`ArchiveDetailScreen`) turadi.
 */
@HiltViewModel
class ArchiveDetailViewModel @Inject constructor(
    private val repo: ArchiveRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(ArchiveDetailUiState())
    val state: StateFlow<ArchiveDetailUiState> = _state.asStateFlow()

    fun start(lessonId: String, lessonTitle: String) {
        if (_state.value.lessonId == lessonId && _state.value.archive != null) return
        _state.value = ArchiveDetailUiState(lessonId = lessonId, lessonTitle = lessonTitle)
        load(userInitiated = false)
    }

    fun refresh() = load(userInitiated = true)

    private fun load(userInitiated: Boolean) {
        val cur = _state.value
        if (cur.lessonId.isBlank() || cur.loading || cur.refreshing) return
        val has = cur.archive != null
        _state.update { it.copy(loading = !has, refreshing = has && userInitiated, error = null) }
        viewModelScope.launch {
            repo.archive(cur.lessonId)
                .onSuccess { a ->
                    _state.update {
                        it.copy(loading = false, refreshing = false, archive = a, error = null, offline = false)
                    }
                }
                .onFailure { t ->
                    val offline = ArchiveRepository.isOffline(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && it.archive != null,
                            error = if (it.archive == null) ApiErrors.humanError(t) else null,
                        )
                    }
                }
        }
    }

    fun setChatQuery(q: String) = _state.update { it.copy(chatQuery = q) }

    fun requestOpen(url: String) = _state.update { it.copy(openUrl = url) }

    /** Ekran havolani ochgach chaqiradi — takror ochilmasin. */
    fun openHandled() = _state.update { it.copy(openUrl = null) }
}
