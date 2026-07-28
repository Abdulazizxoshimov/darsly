@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.recordings

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.Download
import androidx.compose.material.icons.filled.FiberManualRecord
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.data.api.Recording
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.util.LessonFormat
import uz.darsly.mentor.util.RecordingFormat
import uz.darsly.mentor.util.Share

/**
 * Dars yozuvlari.
 *
 * Yuqorida — **hozirgi holat** (yozilyaptimi yoki yo'q) va bitta asosiy tugma;
 * pastida tarix. Ustozning bu ekrandagi savoli deyarli har doim ikkitadan biri:
 * "hozir yozilyaptimi?" va "o'tgan darsni qayerdan olaman?".
 */
@Composable
fun RecordingsScreen(
    lessonId: String,
    lessonTitle: String,
    onBack: () -> Unit,
    vm: RecordingsViewModel = viewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    val snackbar = remember { SnackbarHostState() }

    LaunchedEffect(lessonId) { vm.start(lessonId, lessonTitle) }

    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbar.showSnackbar(it)
            vm.noticeShown()
        }
    }

    // Havola tayyor bo'lganda tizim brauzeriga uzatiladi.
    LaunchedEffect(state.openUrl) {
        state.openUrl?.let { url ->
            Share.openUrl(context, url)
            vm.openUrlHandled()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = {
            TopAppBar(
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Orqaga")
                    }
                },
                title = {
                    Column {
                        Text("Yozuvlar", style = MaterialTheme.typography.titleMedium)
                        Text(
                            state.lessonTitle,
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                        )
                    }
                },
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (state.offline) OfflineBar()

            ControlCard(
                active = state.active,
                busy = state.busy,
                onStart = { vm.startRecording() },
                onStop = { state.active?.let(vm::stopRecording) },
            )

            PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = { vm.refresh() },
                modifier = Modifier.fillMaxSize(),
            ) {
                when {
                    state.loading && state.items.isEmpty() -> Box(
                        Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center,
                    ) { CircularProgressIndicator() }

                    state.error != null && state.items.isEmpty() ->
                        ErrorState(state.error!!) { vm.refresh() }

                    state.items.isEmpty() -> EmptyState()

                    else -> LazyColumn(
                        Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        items(state.items, key = { it.id }) { rec ->
                            RecordingCard(
                                recording = rec,
                                preparing = state.preparingId == rec.id,
                                onDownload = { vm.requestDownload(rec) },
                            )
                        }
                    }
                }
            }
        }
    }
}

/**
 * Yuqoridagi boshqaruv kartasi — "hozir nima bo'layapti" degan savolga javob.
 *
 * Odatda ustoz bu yerda hech narsa bosmaydi: yozuv dars boshlanishi bilan
 * SERVER tomonidan avtomatik boshlanadi (default yoniq). Tugmalar chekka
 * holatlar uchun qoladi — masalan ustoz yozuvni dars o'rtasida to'xtatmoqchi
 * bo'lsa yoki avtomatik boshlash biror sababga ko'ra ishlamagan bo'lsa.
 */
@Composable
private fun ControlCard(
    active: Recording?,
    busy: Boolean,
    onStart: () -> Unit,
    onStop: () -> Unit,
) {
    val isActive = active != null
    Card(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp)) {
        Row(
            Modifier.padding(14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                Icons.Default.FiberManualRecord,
                contentDescription = null,
                tint = if (isActive) MaterialTheme.colorScheme.error else DarslyTheme.colors.textMuted,
                modifier = Modifier.size(16.dp),
            )
            Column(Modifier.weight(1f)) {
                Text(
                    if (isActive) "Hozir yozilmoqda" else "Yozib olish o'chiq",
                    style = MaterialTheme.typography.titleSmall,
                )
                Text(
                    if (isActive) {
                        "Dars tugagach fayl bir necha daqiqada tayyor bo'ladi"
                    } else {
                        "Dars boshlanishi bilan yozuv avtomatik boshlanadi"
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
            if (isActive) {
                OutlinedButton(onClick = onStop, enabled = !busy) {
                    if (busy) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Icon(Icons.Default.Stop, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(6.dp))
                        Text("To'xtatish")
                    }
                }
            } else {
                Button(onClick = onStart, enabled = !busy) {
                    if (busy) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Boshlash")
                    }
                }
            }
        }
    }
}

@Composable
private fun RecordingCard(
    recording: Recording,
    preparing: Boolean,
    onDownload: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(14.dp)) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                StatusBadge(recording.status)
                LessonFormat.scheduleLabel(recording.startedAt ?: recording.createdAt)?.let {
                    Text(
                        it,
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            val facts = listOfNotNull(
                RecordingFormat.durationLabel(recording.durationSec),
                RecordingFormat.sizeLabel(recording.sizeBytes),
            )
            if (facts.isNotEmpty()) {
                Spacer(Modifier.height(6.dp))
                Text(
                    facts.joinToString(" · "),
                    style = MaterialTheme.typography.bodyMedium,
                    color = DarslyTheme.colors.textMuted,
                )
            }

            RecordingFormat.statusHint(recording.status)?.let {
                Spacer(Modifier.height(4.dp))
                Text(
                    it,
                    style = MaterialTheme.typography.bodySmall,
                    color = if (recording.status == RecordingFormat.STATUS_FAILED) {
                        MaterialTheme.colorScheme.error
                    } else {
                        MaterialTheme.colorScheme.onSurfaceVariant
                    },
                )
            }

            // Tugma FAQAT mumkin bo'lgan amal uchun chiqadi: o'chiq turgan
            // "Yuklab olish" tugmasi ustozni "nega ishlamayapti?" deb o'ylatardi.
            if (RecordingFormat.canDownload(recording)) {
                Spacer(Modifier.height(10.dp))
                Button(
                    onClick = onDownload,
                    enabled = !preparing,
                    modifier = Modifier.fillMaxWidth(),
                ) {
                    if (preparing) {
                        CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
                    } else {
                        Icon(Icons.Default.Download, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(6.dp))
                        Text("Yuklab olish")
                    }
                }
            }
        }
    }
}

@Composable
private fun StatusBadge(status: String) {
    val c = DarslyTheme.colors
    val (bg, fg) = when (status) {
        RecordingFormat.STATUS_RECORDING -> c.liveSoft to MaterialTheme.colorScheme.error
        RecordingFormat.STATUS_READY -> c.scheduledSoft to c.success
        RecordingFormat.STATUS_FAILED -> c.liveSoft to MaterialTheme.colorScheme.error
        else -> c.cancelledSoft to c.warning
    }
    Surface(shape = RoundedCornerShape(percent = 50), color = bg) {
        Text(
            RecordingFormat.statusLabel(status),
            style = MaterialTheme.typography.labelMedium,
            color = fg,
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun OfflineBar() {
    Surface(color = MaterialTheme.colorScheme.errorContainer, modifier = Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(horizontal = 16.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Icon(
                Icons.Default.CloudOff,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onErrorContainer,
                modifier = Modifier.size(18.dp),
            )
            Text(
                "Internet yo'q — ro'yxat yangilanmadi",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onErrorContainer,
            )
        }
    }
}

@Composable
private fun ErrorState(message: String, onRetry: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(message, color = MaterialTheme.colorScheme.error)
                Spacer(Modifier.height(12.dp))
                Button(onClick = onRetry) { Text("Qayta urinish") }
            }
        }
    }
}

@Composable
private fun EmptyState() {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text("Yozuv yo'q", style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(4.dp))
                Text(
                    "Dars jonli bo'lganda yozib olishni boshlang — yozuv shu yerda paydo bo'ladi",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
