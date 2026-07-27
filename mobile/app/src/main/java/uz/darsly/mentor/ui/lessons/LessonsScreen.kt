@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.lessons

import androidx.compose.foundation.clickable
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.ContentCopy
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.ExtendedFloatingActionButton
import androidx.compose.material3.Icon
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
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import uz.darsly.mentor.BuildConfig
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.util.LessonFormat
import uz.darsly.mentor.util.Share

/**
 * Darslar ro'yxati (M6 · M7 · M8).
 *
 * Holatlar (mezon B-3): yuklanish · xato · bo'sh · ro'yxat — to'rttasi ham alohida.
 * Ro'yxat keshdan kelgan bo'lsa va tarmoq yo'q bo'lsa yuqorida offline chizig'i turadi.
 */
@Composable
fun LessonsScreen(
    onOpenLesson: (Lesson) -> Unit,
    vm: LessonsViewModel = viewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val context = LocalContext.current
    val snackbar = remember { SnackbarHostState() }
    var createOpen by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(Unit) { vm.start() }

    LaunchedEffect(state.notice) {
        state.notice?.let {
            snackbar.showSnackbar(it)
            vm.noticeShown()
        }
    }

    Scaffold(
        snackbarHost = { SnackbarHost(snackbar) },
        topBar = {
            TopAppBar(
                title = { Text("Darslar") },
                actions = {
                    // M3 — chiqish. Sessiya tozalanadi va login ekraniga qaytiladi.
                    TextButton(onClick = { vm.logout() }, enabled = !state.loggingOut) {
                        Text("Chiqish")
                    }
                },
            )
        },
        floatingActionButton = {
            ExtendedFloatingActionButton(
                onClick = { createOpen = true },
                icon = { Icon(Icons.Default.Add, contentDescription = null) },
                text = { Text("Dars yaratish") },
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (state.offline) OfflineBar(state.cachedAtMillis)

            PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = { vm.refresh() },
                modifier = Modifier.fillMaxSize(),
            ) {
                when {
                    state.loading && state.lessons.isEmpty() -> LoadingState()
                    state.error != null && state.lessons.isEmpty() ->
                        ErrorState(state.error!!) { vm.refresh() }
                    state.lessons.isEmpty() -> EmptyState { createOpen = true }
                    else -> LessonList(
                        lessons = state.lessons,
                        onOpen = onOpenLesson,
                        onShare = { lesson ->
                            Share.sendText(
                                context,
                                LessonFormat.shareText(lesson, BuildConfig.WEB_BASE_URL),
                            )
                        },
                        onCopy = { lesson ->
                            val url = LessonFormat.joinUrl(BuildConfig.WEB_BASE_URL, lesson.joinSlug)
                            if (url == null) {
                                vm.showNotice("Bu darsning havolasi yo'q")
                            } else if (Share.copyToClipboard(context, url)) {
                                vm.showNotice("Havola nusxalandi")
                            }
                        },
                    )
                }
            }
        }
    }

    if (createOpen) {
        CreateLessonDialog(
            onDismiss = { createOpen = false },
            onCreated = { lesson ->
                createOpen = false
                vm.onLessonCreated(lesson)
            },
        )
    }
}

/** B-4: keshdagi ro'yxat ko'rsatilyapti — ustoz buni ko'rib turishi kerak. */
@Composable
private fun OfflineBar(cachedAtMillis: Long) {
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
            Column {
                Text(
                    "Internet yo'q — saqlangan ro'yxat",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onErrorContainer,
                )
                if (cachedAtMillis > 0) {
                    LessonFormat.scheduleLabel(LessonForm.rfc3339Utc(cachedAtMillis))?.let {
                        Text(
                            "Oxirgi yangilanish: $it",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onErrorContainer,
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun LoadingState() {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
        CircularProgressIndicator()
    }
}

@Composable
private fun ErrorState(message: String, onRetry: () -> Unit) {
    // Scroll qilinadigan konteyner: pull-to-refresh xato ekranida ham ishlasin.
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
private fun EmptyState(onCreate: () -> Unit) {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text("Hali dars yo'q", style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(4.dp))
                Text(
                    "Birinchi darsni shu yerda yarating va havolasini o'quvchilarga yuboring",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(16.dp))
                Button(onClick = onCreate) { Text("Dars yaratish") }
            }
        }
    }
}

@Composable
private fun LessonList(
    lessons: List<Lesson>,
    onOpen: (Lesson) -> Unit,
    onShare: (Lesson) -> Unit,
    onCopy: (Lesson) -> Unit,
) {
    LazyColumn(
        Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        // Bo'limlar tartibi `LessonFormat.sortForDisplay` bilan bir xil bo'lgani
        // uchun ro'yxatni qayta tartiblamaymiz — faqat sarlavha qo'yamiz.
        LessonFormat.Section.entries.forEach { section ->
            val group = lessons.filter { LessonFormat.sectionOf(it) == section }
            if (group.isEmpty()) return@forEach
            item(key = "section-$section") {
                Text(
                    LessonFormat.sectionTitle(section),
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(top = 4.dp),
                )
            }
            items(group, key = { it.id }) { lesson ->
                LessonCard(
                    lesson = lesson,
                    onClick = { onOpen(lesson) },
                    onShare = { onShare(lesson) },
                    onCopy = { onCopy(lesson) },
                )
            }
        }
    }
}

/**
 * Dars xususiyati yorlig'i — "Parol", "Kutish xonasi", "Yozib olish", "Qulflangan".
 *
 * 🟢K: avval bu yerda `AssistChip` ishlatilgan edi. `AssistChip` `onClick` TALAB
 * qiladi va bosilganda darsni ochib yuborardi — ya'ni yorliq filtr yoki tugmaga
 * o'xshab ko'rinardi. Bu shunchaki **ma'lumot**, shuning uchun bosilmaydigan
 * `Surface`: interfeys nimani bosish mumkinligi haqida yolg'on aytmasligi kerak.
 */
@Composable
private fun LessonBadge(label: String) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.surfaceVariant,
        contentColor = MaterialTheme.colorScheme.onSurfaceVariant,
    ) {
        Text(
            label,
            style = MaterialTheme.typography.labelSmall,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
        )
    }
}

@Composable
private fun LessonCard(
    lesson: Lesson,
    onClick: () -> Unit,
    onShare: () -> Unit,
    onCopy: () -> Unit,
) {
    Card(Modifier.fillMaxWidth().clickable(onClick = onClick)) {
        Column(Modifier.padding(16.dp)) {
            Text(lesson.title, style = MaterialTheme.typography.titleMedium)
            Spacer(Modifier.height(6.dp))

            Row(horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                Text(
                    LessonFormat.statusLabel(lesson.status),
                    style = MaterialTheme.typography.labelMedium,
                    color = if (lesson.status == "live") {
                        MaterialTheme.colorScheme.error
                    } else {
                        MaterialTheme.colorScheme.primary
                    },
                )
                Text(
                    LessonFormat.scheduleLabel(lesson.scheduledAt) ?: "Vaqti belgilanmagan",
                    style = MaterialTheme.typography.labelMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                LessonFormat.durationLabel(lesson.durationMin)?.let {
                    Text(
                        it,
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            val badges = buildList {
                if (lesson.hasPasscode) add("Parol")
                if (lesson.isWaitingRoomEnabled) add("Kutish xonasi")
                if (lesson.isRecordingEnabled) add("Yozib olish")
                if (lesson.isLocked) add("Qulflangan")
            }
            if (badges.isNotEmpty()) {
                Spacer(Modifier.height(8.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                    badges.forEach { label -> LessonBadge(label) }
                }
            }

            Spacer(Modifier.height(8.dp))
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Button(onClick = onClick, modifier = Modifier.weight(1f)) {
                    Text(if (lesson.status == "live") "Davom etish" else "Boshlash")
                }
                // M8 — join havolasi Telegramga tizim "Ulashish" oynasi orqali.
                TextButton(onClick = onShare, enabled = lesson.joinSlug != null) {
                    Icon(Icons.Default.Share, contentDescription = null, modifier = Modifier.size(18.dp))
                    // 🟢J: gorizontal qator — bo'shliq `width` bo'lishi kerak
                    // (`height(0.dp)` hech narsa qilmaydigan qator edi).
                    Spacer(Modifier.width(6.dp))
                    Text("Ulashish")
                }
                TextButton(onClick = onCopy, enabled = lesson.joinSlug != null) {
                    Icon(Icons.Default.ContentCopy, contentDescription = "Havolani nusxalash", modifier = Modifier.size(18.dp))
                }
            }
        }
    }
}
