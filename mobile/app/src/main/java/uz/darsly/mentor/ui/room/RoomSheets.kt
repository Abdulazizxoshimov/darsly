package uz.darsly.mentor.ui.room

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.PanTool
import androidx.compose.material.icons.filled.PersonRemove
import androidx.compose.material.icons.filled.Send
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

/**
 * Ishtirokchilar paneli — mobil ustoz uchun moderatsiya.
 *
 * Shu paytgacha telefondan dars o'tayotgan ustoz shovqin qilayotgan o'quvchini
 * mute ham qila olmasdi: bu amallar faqat webda bor edi. Endi telefon ham
 * to'liq boshqaruv beradi.
 *
 * Qo'l ko'targanlar ENG TEPADA va navbat tartibida: ustoz 40 kishilik ro'yxatni
 * ko'zi bilan qidirmasligi kerak.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ParticipantsSheet(
    state: RoomUiState,
    onDismiss: () -> Unit,
    onMuteAll: () -> Unit,
    onMute: (String) -> Unit,
    onRemove: (String) -> Unit,
    onAllowSpeak: (String) -> Unit,
    onLowerHand: (String) -> Unit,
    onLowerAllHands: () -> Unit,
    onRefresh: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // Panel ochilganda ro'yxat serverdan yangilanadi: LiveKit hodisalari orasida
    // o'tib ketgan o'zgarishlar (mute holati) shu yerda tekislanadi.
    LaunchedEffect(Unit) { onRefresh() }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "Ishtirokchilar (${state.roster.size})",
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                TextButton(onClick = onMuteAll) { Text("Hammani mute") }
            }

            if (state.hands.isNotEmpty()) {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(
                        "✋ Qo'l ko'targanlar (${state.hands.size})",
                        style = MaterialTheme.typography.labelLarge,
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.weight(1f),
                    )
                    TextButton(onClick = onLowerAllHands) { Text("Tushirish") }
                }
                state.hands.forEachIndexed { i, h ->
                    Row(
                        Modifier.fillMaxWidth().padding(vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Text("${i + 1}.", style = MaterialTheme.typography.bodyMedium)
                        Text(
                            h.name,
                            style = MaterialTheme.typography.bodyMedium,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f).padding(start = 8.dp),
                        )
                        TextButton(onClick = { onAllowSpeak(h.identity) }) { Text("Ruxsat") }
                        IconButton(onClick = { onLowerHand(h.identity) }) {
                            Icon(Icons.Default.PanTool, contentDescription = "Qo'lni tushirish")
                        }
                    }
                }
            }

            if (state.roster.isEmpty()) {
                Text(
                    "Xonada hali hech kim yo'q",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(vertical = 16.dp),
                )
            } else {
                LazyColumn(Modifier.heightIn(max = 360.dp)) {
                    items(state.roster, key = { it.identity }) { p ->
                        Row(
                            Modifier.fillMaxWidth().padding(vertical = 6.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Text(
                                p.name,
                                style = MaterialTheme.typography.bodyMedium,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                                modifier = Modifier.weight(1f),
                            )
                            if (p.handRaised) {
                                Icon(
                                    Icons.Default.PanTool,
                                    contentDescription = "Qo'l ko'tarilgan",
                                    tint = MaterialTheme.colorScheme.primary,
                                )
                            }
                            IconButton(onClick = { onMute(p.identity) }) {
                                Icon(
                                    if (p.audioMuted) Icons.Default.MicOff else Icons.Default.Mic,
                                    contentDescription = "Mute",
                                )
                            }
                            IconButton(onClick = { onRemove(p.identity) }) {
                                Icon(
                                    Icons.Default.PersonRemove,
                                    contentDescription = "Chiqarib yuborish",
                                    tint = MaterialTheme.colorScheme.error,
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

/**
 * Chat paneli.
 *
 * Xabarlar SERVERDA saqlanadi; bu oyna faqat ko'rinish. Shaxsiy xabar alohida
 * belgilanadi — ustoz "buni hamma ko'rdimi?" degan savolga bir qarashda javob
 * topishi kerak.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatSheet(
    state: RoomUiState,
    onDismiss: () -> Unit,
    onSend: (String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    var draft by remember { mutableStateOf("") }
    val listState = rememberLazyListState()

    // Yangi xabar kelganda pastga suramiz — ustoz oxirgi savolni ko'rsin.
    LaunchedEffect(state.chat.size) {
        if (state.chat.isNotEmpty()) listState.animateScrollToItem(state.chat.lastIndex)
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp)) {
            Text("Chat", style = MaterialTheme.typography.titleMedium)

            if (state.chat.isEmpty()) {
                Text(
                    "Hali xabar yo'q",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(vertical = 16.dp),
                )
            } else {
                LazyColumn(
                    state = listState,
                    modifier = Modifier.heightIn(max = 380.dp).padding(vertical = 8.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    items(state.chat, key = { it.id }) { m -> ChatBubble(m) }
                }
            }

            Row(verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(
                    value = draft,
                    onValueChange = { draft = it },
                    placeholder = { Text("Xabar yozing…") },
                    singleLine = true,
                    modifier = Modifier.weight(1f),
                )
                IconButton(
                    onClick = {
                        val body = draft.trim()
                        if (body.isNotEmpty()) {
                            onSend(body)
                            draft = ""
                        }
                    },
                ) {
                    Icon(Icons.Default.Send, contentDescription = "Yuborish")
                }
            }
        }
    }
}

@Composable
private fun ChatBubble(m: ChatMessageUi) {
    val dm = m.toIdentity != null
    Box(
        Modifier.fillMaxWidth(),
        contentAlignment = if (m.self) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Column(horizontalAlignment = if (m.self) Alignment.End else Alignment.Start) {
            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(
                    if (m.self) "Siz" else m.name,
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                if (dm) {
                    Text(
                        "shaxsiy",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.tertiary,
                    )
                }
            }
            Surface(
                color = when {
                    m.self -> MaterialTheme.colorScheme.primary
                    dm -> MaterialTheme.colorScheme.tertiaryContainer
                    else -> MaterialTheme.colorScheme.surfaceVariant
                },
                shape = RoundedCornerShape(12.dp),
            ) {
                Text(
                    m.body,
                    style = MaterialTheme.typography.bodyMedium,
                    color = if (m.self) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface,
                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                )
            }
        }
    }
}
