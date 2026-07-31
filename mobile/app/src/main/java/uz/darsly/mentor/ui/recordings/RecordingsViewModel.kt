package uz.darsly.mentor.ui.recordings

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
import uz.darsly.mentor.data.api.Recording
import uz.darsly.mentor.data.repo.RecordingsRepository
import uz.darsly.mentor.util.RecordingFormat

data class RecordingsUiState(
    val lessonId: String = "",
    val lessonTitle: String = "",
    val loading: Boolean = false,
    val refreshing: Boolean = false,
    val items: List<Recording> = emptyList(),
    val error: String? = null,
    val offline: Boolean = false,
    /** Boshlanmoqda/to'xtatilmoqda — tugmalar bloklanadi. */
    val busy: Boolean = false,
    /** Havola so'ralayotgan yozuv ID'si (o'sha qatorda aylana ko'rsatiladi). */
    val preparingId: String? = null,
    /** Brauzerda ochilishi kerak bo'lgan havola — ekran uni iste'mol qiladi. */
    val openUrl: String? = null,
    val notice: String? = null,
) {
    /** Faol yozuv bormi — "Boshlash" va "To'xtatish" tugmalari shunga qarab almashadi. */
    val active: Recording? get() = items.firstOrNull { RecordingFormat.canStop(it) }
}

/**
 * Dars yozuvlari: ro'yxat, boshlash/to'xtatish, yuklab olish.
 *
 * ## Mahsulot qarori
 * Yozib olish — Zoom'da dars **ichida** bosiladigan tugma, oldindan sozlanadigan
 * belgi emas. Shuning uchun bu ekran nafaqat tarixni ko'rsatadi, balki yozishni
 * boshlash/to'xtatish imkonini ham beradi. Dars yaratishdagi "Yozib olish"
 * belgisi ham qoladi (u avtomatik boshlanishni bildiradi).
 *
 * ## Nega yuklab olish tizim brauzerida
 * Server **presigned** (imzolangan, muddatli) havola beradi. Faylni ilova ichida
 * yuklab olish uchun `DownloadManager`, saqlash ruxsatlari va progress UI kerak
 * bo'lardi — dars video fayli yuz megabaytlarcha. Tizim brauzeri/yuklab olish
 * menejeri buni allaqachon to'g'ri qiladi: fon rejimi, davom ettirish, xabarnoma.
 */
@HiltViewModel
class RecordingsViewModel @Inject constructor(
    private val repo: RecordingsRepository,
) : ViewModel() {

    private val _state = MutableStateFlow(RecordingsUiState())
    val state: StateFlow<RecordingsUiState> = _state.asStateFlow()

    fun start(lessonId: String, lessonTitle: String) {
        if (_state.value.lessonId == lessonId) return
        _state.value = RecordingsUiState(lessonId = lessonId, lessonTitle = lessonTitle)
        refresh(userInitiated = false)
    }

    fun refresh(userInitiated: Boolean = true) {
        val current = _state.value
        if (current.lessonId.isBlank() || current.loading || current.refreshing) return
        val hasContent = current.items.isNotEmpty()
        _state.update {
            it.copy(loading = !hasContent, refreshing = hasContent && userInitiated, error = null)
        }
        viewModelScope.launch {
            repo.list(current.lessonId)
                .onSuccess { items ->
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            items = RecordingFormat.sortForDisplay(items),
                            error = null,
                            offline = false,
                        )
                    }
                }
                .onFailure { t ->
                    val offline = RecordingsRepository.isOffline(t)
                    val message = ApiErrors.humanError(t)
                    _state.update {
                        it.copy(
                            loading = false,
                            refreshing = false,
                            offline = offline && it.items.isNotEmpty(),
                            error = if (it.items.isEmpty()) message else null,
                            notice = if (it.items.isEmpty()) null else message,
                        )
                    }
                }
        }
    }

    /**
     * Yozib olishni boshlash.
     *
     * DIQQAT: backend buni **jonli** dars uchun bajaradi (LiveKit xonasi ochiq
     * bo'lishi kerak). Dars boshlanmagan bo'lsa server xato beradi va biz uni
     * o'zbekcha ko'rsatamiz — bu yerda oldindan tekshirmaymiz, chunki "jonli"
     * holatning yagona haqiqiy manbai serverda.
     */
    fun startRecording() {
        val current = _state.value
        if (current.busy || current.lessonId.isBlank()) return
        _state.update { it.copy(busy = true) }
        viewModelScope.launch {
            repo.start(current.lessonId)
                .onSuccess { rec ->
                    _state.update {
                        it.copy(
                            busy = false,
                            items = RecordingFormat.sortForDisplay(
                                listOf(rec) + it.items.filterNot { r -> r.id == rec.id },
                            ),
                            notice = "Yozib olish boshlandi",
                        )
                    }
                }
                .onFailure { t ->
                    _state.update { it.copy(busy = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * To'xtatish. Muvaffaqiyatda ro'yxat serverdan qayta olinadi.
     *
     * Lokal ravishda statusni `processing` qilib qo'ymaymiz: server 204 qaytaradi
     * va yangi holatni bermaydi, ya'ni bizning taxminimiz haqiqatdan ajralib
     * ketishi mumkin (masalan egress allaqachon tugagan bo'lsa).
     */
    fun stopRecording(recording: Recording) {
        val current = _state.value
        if (current.busy) return
        _state.update { it.copy(busy = true) }
        viewModelScope.launch {
            repo.stop(recording.id)
                .onSuccess {
                    _state.update { it.copy(busy = false, notice = "Yozib olish to'xtatildi") }
                    refresh(userInitiated = false)
                }
                .onFailure { t ->
                    _state.update { it.copy(busy = false, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /**
     * Yuklab olish havolasini so'raydi.
     *
     * Havola saqlanmaydi — u muddatli. Har bosishda yangisi olinadi, aks holda
     * bir necha soatdan keyin bosilgan tugma 403 berardi va bu "yozuv yo'qolgan"
     * dek ko'rinardi.
     */
    fun requestDownload(recording: Recording) {
        if (_state.value.preparingId != null) return
        _state.update { it.copy(preparingId = recording.id) }
        viewModelScope.launch {
            repo.downloadUrl(recording.id)
                .onSuccess { dl ->
                    _state.update {
                        it.copy(
                            preparingId = null,
                            openUrl = dl.url,
                            notice = RecordingFormat.expiryLabel(dl.expiresInS),
                        )
                    }
                }
                .onFailure { t ->
                    _state.update { it.copy(preparingId = null, notice = ApiErrors.humanError(t)) }
                }
        }
    }

    /** Ekran havolani ochgach chaqiradi — takror ochilmasin. */
    fun openUrlHandled() = _state.update { it.copy(openUrl = null) }

    fun noticeShown() = _state.update { it.copy(notice = null) }
}
