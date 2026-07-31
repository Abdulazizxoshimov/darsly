@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.blocklist

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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.PersonOff
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import uz.darsly.mentor.data.api.BlocklistEntry
import uz.darsly.mentor.util.LessonFormat

/**
 * Qora ro'yxat — doimiy ban qo'yilgan ishtirokchilar (№4).
 *
 * ## Nega bu ekran mavjud
 * Ban QO'YISH telefondan mumkin edi (xona panelidagi «Doimiy»), OLIB TASHLASH
 * esa faqat web'da. Ustoz asosan telefondan ishlaydi (PRODUCT.md:31) — ya'ni
 * dars o'rtasida shoshib bosilgan tugma amalda qaytarib bo'lmas holat edi.
 *
 * ## Nega tasdiq dialogi bor
 * Olib tashlash — bir bosishda bajariladigan, lekin ko'rinmas oqibatli amal:
 * o'quvchi shu ondan boshlab BARCHA darslarga qaytib kira oladi. Tasdiq
 * "kimni" degan savolga ism bilan javob beradi.
 */
@Composable
fun BlocklistScreen(
    onBack: () -> Unit,
    vm: BlocklistViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }
    var confirmTarget by remember { mutableStateOf<BlocklistEntry?>(null) }

    LaunchedEffect(Unit) { vm.start() }
    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbar.showSnackbar(it)
            vm.noticeShown()
        }
    }

    confirmTarget?.let { entry ->
        val name = BlocklistFormat.nameOf(entry)
        AlertDialog(
            onDismissRequest = { confirmTarget = null },
            title = { Text("$name qaytarilsinmi?") },
            text = {
                Text(
                    "Ban olib tashlanadi va bu ishtirokchi shu ondan boshlab " +
                        "barcha darslaringizga qaytib kira oladi.",
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    confirmTarget = null
                    vm.unblock(entry)
                }) { Text("Qaytarish") }
            },
            dismissButton = {
                TextButton(onClick = { confirmTarget = null }) { Text("Bekor qilish") }
            },
        )
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
                title = { Text("Qora ro'yxat") },
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (state.offline) OfflineBar()

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
                        item {
                            Text(
                                "Bu ishtirokchilar (ism bo'yicha) hech qaysi darsingizga " +
                                    "kira olmaydi.",
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(bottom = 4.dp),
                            )
                        }
                        items(state.items, key = { it.id }) { entry ->
                            BlockedRow(
                                entry = entry,
                                removing = state.removingId == entry.id,
                                // Boshqa qator o'chirilayotganda ikkinchi so'rov
                                // yuborilmasin (VM ham rad etardi, lekin bosiladigan
                                // tugma "ishlamadi"dek ko'rinardi).
                                enabled = state.removingId == null,
                                onUnblock = { confirmTarget = entry },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun BlockedRow(
    entry: BlocklistEntry,
    removing: Boolean,
    enabled: Boolean,
    onUnblock: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Row(
            Modifier.padding(14.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            Icon(
                Icons.Default.PersonOff,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.error,
                modifier = Modifier.size(20.dp),
            )
            Column(Modifier.weight(1f)) {
                Text(
                    BlocklistFormat.nameOf(entry),
                    style = MaterialTheme.typography.bodyLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                LessonFormat.scheduleLabel(entry.createdAt)?.let {
                    Text(
                        "Bloklangan: $it",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
            if (removing) {
                CircularProgressIndicator(Modifier.size(18.dp), strokeWidth = 2.dp)
            } else {
                TextButton(onClick = onUnblock, enabled = enabled) { Text("Qaytarish") }
            }
        }
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

/**
 * Bo'sh holat — bu ekranda ENG KO'P uchraydigan holat, shuning uchun u
 * "xato"dek emas, tinch ma'lumotdek ko'rinadi.
 */
@Composable
private fun EmptyState() {
    // LazyColumn — pull-to-refresh ishlashi uchun (oddiy Column'ni tortib bo'lmaydi).
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillMaxWidth().padding(top = 96.dp, start = 32.dp, end = 32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text("Qora ro'yxat bo'sh", style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(8.dp))
                Text(
                    "Dars paytida ishtirokchini chiqarganda «Doimiy» ni tanlasangiz, " +
                        "u shu yerda paydo bo'ladi va istagan paytda qaytarib olasiz.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                )
            }
        }
    }
}

@Composable
private fun ErrorState(message: String, onRetry: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillMaxWidth().padding(top = 96.dp, start = 32.dp, end = 32.dp),
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(message, color = MaterialTheme.colorScheme.error, textAlign = TextAlign.Center)
                Spacer(Modifier.height(12.dp))
                Button(onClick = onRetry) { Text("Qayta urinish") }
            }
        }
    }
}
