@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.schedule

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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.CloudOff
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
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import uz.darsly.mentor.ui.common.NotificationBell
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
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.data.api.Lesson
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.util.LessonFormat
import uz.darsly.mentor.util.ScheduleFormat
import java.time.LocalDate
import uz.darsly.mentor.ui.theme.neonGlow

/**
 * Jadval — darslar kunlar bo'yicha.
 *
 * Har kun sarlavha ostida guruhlanadi ("Bugun, 27-iyul · 3 dars · 4 soat"), dars
 * qatorining chap tomonida esa soat turadi. Vaqt chapda bo'lishi ataylab: ustoz
 * ro'yxatni **vaqt ustuni** bo'ylab tez o'qiy oladi, sarlavhalarni o'qimasdan.
 */
@Composable
fun ScheduleScreen(
    onOpenLesson: (Lesson) -> Unit,
    unread: Int,
    onOpenNotifications: () -> Unit,
    vm: ScheduleViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }
    val today = remember { LocalDate.now() }
    var scheduleOpen by rememberSaveable { mutableStateOf(false) }

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
                title = { Text("Jadval") },
                actions = { NotificationBell(unread = unread, onClick = onOpenNotifications) },
            )
        },
        floatingActionButton = {
            // Ekranning asosiy harakati — Darslar FAB'i bilan bir xil B uslubi.
            ExtendedFloatingActionButton(
                onClick = { scheduleOpen = true },
                icon = { Icon(Icons.Default.Add, contentDescription = null) },
                text = { Text("Dars rejalashtirish") },
                containerColor = MaterialTheme.colorScheme.primary,
                contentColor = MaterialTheme.colorScheme.onPrimary,
                modifier = Modifier.neonGlow(
                    color = DarslyTheme.colors.neon,
                    shape = MaterialTheme.shapes.large,
                ),
            )
        },
    ) { padding ->
        Column(Modifier.fillMaxSize().padding(padding)) {
            if (state.offline) OfflineBar()

            PastToggle(
                checked = state.includePast,
                onChange = vm::setIncludePast,
            )

            PullToRefreshBox(
                isRefreshing = state.refreshing,
                onRefresh = { vm.refresh() },
                modifier = Modifier.fillMaxSize(),
            ) {
                when {
                    state.loading && state.isEmpty -> Box(
                        Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center,
                    ) { CircularProgressIndicator() }

                    state.error != null && state.isEmpty ->
                        ErrorState(state.error!!) { vm.refresh() }

                    state.isEmpty -> EmptyState(state.includePast)

                    else -> LazyColumn(
                        Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        state.days.forEach { day ->
                            item(key = "day-${day.date}") {
                                DayHeader(day, today)
                            }
                            items(day.lessons, key = { it.id }) { lesson ->
                                ScheduleRow(lesson, onClick = { onOpenLesson(lesson) })
                            }
                        }

                        if (state.undated.isNotEmpty()) {
                            item(key = "undated-header") {
                                Text(
                                    "Vaqti belgilanmagan",
                                    style = MaterialTheme.typography.titleSmall,
                                    color = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.padding(top = 12.dp),
                                )
                            }
                            items(state.undated, key = { it.id }) { lesson ->
                                ScheduleRow(lesson, onClick = { onOpenLesson(lesson) })
                            }
                        }
                    }
                }
            }
        }
    }

    if (scheduleOpen) {
        ScheduleLessonDialog(
            onDismiss = { scheduleOpen = false },
            onScheduled = {
                scheduleOpen = false
                // Yangi dars ro'yxatda darhol ko'rinsin — serverdan qayta o'qiymiz
                // (jadval kun bo'yicha guruhlangan, lokal qo'shishdan ko'ra ishonchli).
                vm.refresh(userInitiated = false)
                vm.showNotice("Dars rejalashtirildi")
            },
        )
    }
}

@Composable
private fun PastToggle(checked: Boolean, onChange: (Boolean) -> Unit) {
    Row(
        Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            "O'tgan darslarni ham ko'rsatish",
            style = MaterialTheme.typography.bodyMedium,
            modifier = Modifier.weight(1f),
        )
        Switch(checked = checked, onCheckedChange = onChange)
    }
}

@Composable
private fun DayHeader(day: ScheduleFormat.Day, today: LocalDate) {
    val isToday = day.date == today
    Column(Modifier.padding(top = 12.dp, bottom = 2.dp)) {
        Text(
            ScheduleFormat.dayTitle(day.date, today),
            style = MaterialTheme.typography.titleSmall,
            // Bugungi kun ajralib turadi — ustoz ro'yxatda o'zini tez topadi.
            color = if (isToday) {
                MaterialTheme.colorScheme.primary
            } else {
                MaterialTheme.colorScheme.onSurface
            },
        )
        Text(
            ScheduleFormat.daySummary(day),
            style = MaterialTheme.typography.labelSmall,
            color = DarslyTheme.colors.textMuted,
        )
    }
}

@Composable
private fun ScheduleRow(lesson: Lesson, onClick: () -> Unit) {
    Card(Modifier.fillMaxWidth().clickable(onClick = onClick)) {
        Row(
            Modifier.padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Vaqt ustuni — qat'iy kenglikda, shuning uchun qatorlar tekis turadi.
            Box(Modifier.width(52.dp)) {
                Text(
                    ScheduleFormat.timeLabel(lesson) ?: "—",
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.onSurface,
                )
            }
            Spacer(Modifier.width(8.dp))
            Column(Modifier.weight(1f)) {
                Text(
                    lesson.title,
                    style = MaterialTheme.typography.bodyLarge,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                val facts = listOfNotNull(
                    LessonFormat.durationLabel(lesson.durationMin),
                    if (lesson.hasPasscode) "Parol" else null,
                    if (lesson.isWaitingRoomEnabled) "Kutish xonasi" else null,
                )
                if (facts.isNotEmpty()) {
                    Text(
                        facts.joinToString(" · "),
                        style = MaterialTheme.typography.labelSmall,
                        color = DarslyTheme.colors.textMuted,
                    )
                }
            }
            StatusDot(lesson.status)
        }
    }
}

/** Holat nuqtasi — jadvalda to'liq nishon uchun joy yo'q, rang yetarli. */
@Composable
private fun StatusDot(status: String) {
    val c = DarslyTheme.colors
    val color = when (status) {
        "live" -> MaterialTheme.colorScheme.error
        "ended" -> c.textMuted
        else -> c.info
    }
    Surface(shape = RoundedCornerShape(percent = 50), color = color, modifier = Modifier.size(8.dp)) {}
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
                "Internet yo'q — saqlangan jadval",
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
private fun EmptyState(includePast: Boolean) {
    LazyColumn(Modifier.fillMaxSize()) {
        item {
            Column(
                Modifier.fillParentMaxSize().padding(24.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                Text(
                    if (includePast) "Jadval bo'sh" else "Rejalashtirilgan dars yo'q",
                    style = MaterialTheme.typography.titleMedium,
                )
                Spacer(Modifier.height(4.dp))
                Text(
                    if (includePast) {
                        "Vaqti belgilangan dars yaratsangiz shu yerda ko'rinadi"
                    } else {
                        "Kelgusi darslar shu yerda ko'rinadi. O'tganlarini " +
                            "yuqoridagi almashtirgich bilan ko'rishingiz mumkin"
                    },
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
            }
        }
    }
}
