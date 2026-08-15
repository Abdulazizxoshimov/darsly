@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.notifications

import androidx.compose.foundation.BorderStroke
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
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudOff
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.ExperimentalMaterial3Api
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
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.hilt.navigation.compose.hiltViewModel
import uz.darsly.mentor.R
import uz.darsly.mentor.data.api.Notification
import uz.darsly.mentor.ui.common.EmptyState
import uz.darsly.mentor.ui.common.ErrorState
import uz.darsly.mentor.ui.common.ListRowSkeleton
import uz.darsly.mentor.ui.theme.DarslyTheme
import uz.darsly.mentor.util.NotificationFormat

/**
 * Bildirishnomalar.
 *
 * O'qilmagan yozuv **ko'rinib turishi** kerak: chap tomonda nuqta, quyuqroq fon
 * va qalin sarlavha. Zoom ham shu naqshni ishlatadi — foydalanuvchi ro'yxatni
 * o'qimasdan, bir qarashda "yangi bormi?" degan savolga javob oladi.
 *
 * Yozuvni bosish uni o'qilgan deb belgilaydi. Darsga bog'liq bildirishnoma
 * bosilganda o'sha darsni ochish mumkin ([onOpenLesson]).
 */
@Composable
fun NotificationsScreen(
    onOpenLesson: (String) -> Unit,
    vm: NotificationsViewModel = hiltViewModel(),
) {
    val state by vm.state.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }

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
                title = { Text("Bildirishnomalar") },
                actions = {
                    if (state.unread > 0) {
                        TextButton(onClick = { vm.markAllRead() }, enabled = !state.markingAll) {
                            Text("Hammasini o'qildi")
                        }
                    }
                },
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
                    state.loading && state.items.isEmpty() -> ListRowSkeleton()

                    state.error != null && state.items.isEmpty() ->
                        ErrorState(message = state.error!!, onRetry = { vm.refresh() })

                    state.items.isEmpty() -> EmptyState(
                        illustration = R.drawable.il_empty_notifications,
                        title = "Bildirishnoma yo'q",
                        message = "Dars boshlanishidan oldin eslatma va kutish xonasidagi " +
                            "o'quvchilar haqidagi xabarlar shu yerda chiqadi",
                    )

                    else -> LazyColumn(
                        Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                    ) {
                        items(state.items, key = { it.id }) { item ->
                            NotificationCard(
                                item = item,
                                onClick = {
                                    vm.markRead(item)
                                    item.lessonId?.let(onOpenLesson)
                                },
                            )
                        }
                    }
                }
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


@Composable
private fun NotificationCard(item: Notification, onClick: () -> Unit) {
    val unread = item.isUnread
    Card(
        Modifier.fillMaxWidth().clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = if (unread) {
                MaterialTheme.colorScheme.surfaceContainerHigh
            } else {
                MaterialTheme.colorScheme.surface
            },
        ),
        border = if (unread) BorderStroke(1.dp, MaterialTheme.colorScheme.primary) else null,
    ) {
        Row(
            Modifier.padding(14.dp),
            horizontalArrangement = Arrangement.spacedBy(10.dp),
        ) {
            // O'qilmaganlik belgisi — matnga tayanmaydigan, bir qarashda ko'rinadigan.
            Box(Modifier.padding(top = 6.dp)) {
                Surface(
                    shape = CircleShape,
                    color = if (unread) {
                        MaterialTheme.colorScheme.primary
                    } else {
                        MaterialTheme.colorScheme.surfaceVariant
                    },
                    modifier = Modifier.size(8.dp),
                ) {}
            }

            Column(Modifier.weight(1f)) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    TypeBadge(item.type)
                    NotificationFormat.relativeTime(item.createdAt)?.let {
                        Text(
                            it,
                            style = MaterialTheme.typography.labelSmall,
                            color = DarslyTheme.colors.textMuted,
                        )
                    }
                }
                Spacer(Modifier.height(6.dp))
                Text(
                    item.title,
                    style = MaterialTheme.typography.titleSmall,
                    color = MaterialTheme.colorScheme.onSurface,
                )
                if (item.body.isNotBlank()) {
                    Spacer(Modifier.height(2.dp))
                    Text(
                        item.body,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
    }
}

@Composable
private fun TypeBadge(type: String) {
    val c = DarslyTheme.colors
    val (bg, fg) = when (type) {
        NotificationFormat.TYPE_LESSON_REMINDER -> c.scheduledSoft to c.info
        NotificationFormat.TYPE_WAITING_ROOM -> c.cancelledSoft to c.warning
        else -> c.endedSoft to c.textMuted
    }
    Surface(shape = RoundedCornerShape(percent = 50), color = bg) {
        Text(
            NotificationFormat.typeLabel(type),
            style = MaterialTheme.typography.labelSmall,
            color = fg,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
        )
    }
}
