package uz.darsly.mentor.ui.room

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.ExperimentalFoundationApi
import androidx.compose.foundation.combinedClickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AttachFile
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.PanTool
import androidx.compose.material.icons.filled.PersonRemove
import androidx.compose.material.icons.filled.Search
import androidx.compose.material.icons.filled.Send
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Checkbox
import androidx.compose.foundation.layout.Spacer
import androidx.compose.material3.LinearProgressIndicator
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
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import uz.darsly.mentor.util.ChatUpload
import uz.darsly.mentor.util.Share

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
    onMuteAll: (Boolean?) -> Unit,
    onMute: (String) -> Unit,
    onRemove: (String, Boolean) -> Unit,
    onAllowSpeak: (String) -> Unit,
    onLowerHand: (String) -> Unit,
    onLowerAllHands: () -> Unit,
    onRefresh: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    // Zoom andozasi: «Hammani o'chirish» tasdiq dialogi (checkbox bilan) va
    // chiqarishda qamrov tanlovi (shu dars / doimiy).
    var muteAllOpen by remember { mutableStateOf(false) }
    var removeTarget by remember { mutableStateOf<Pair<String, String>?>(null) } // identity to name

    // Qidiruv — mahalliy holat: u serverga ham, xona sessiyasiga ham tegishli
    // emas, panel yopilishi bilan yo'qolishi TO'G'RI xulq.
    var query by remember { mutableStateOf("") }
    val searching = query.isNotBlank()
    val visible = remember(state.roster, query) { ParticipantFilter.roster(state.roster, query) }

    // Panel ochilganda ro'yxat serverdan yangilanadi: LiveKit hodisalari orasida
    // o'tib ketgan o'zgarishlar (mute holati) shu yerda tekislanadi.
    LaunchedEffect(Unit) { onRefresh() }

    if (muteAllOpen) {
        var lockUnmute by remember { mutableStateOf(false) }
        AlertDialog(
            onDismissRequest = { muteAllOpen = false },
            title = { Text("Hammani o'chirish") },
            text = {
                Column {
                    Text("Barcha o'quvchilarning mikrofoni o'chiriladi.")
                    Spacer(Modifier.height(8.dp))
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Checkbox(checked = lockUnmute, onCheckedChange = { lockUnmute = it })
                        Text(
                            "O'quvchilar o'zi qayta ocholmasin",
                            style = MaterialTheme.typography.bodyMedium,
                        )
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    muteAllOpen = false
                    // checked -> allow_self_unmute=false; aks holda bayroqqa tegilmaydi
                    onMuteAll(if (lockUnmute) false else null)
                }) { Text("O'chirish") }
            },
            dismissButton = {
                TextButton(onClick = { muteAllOpen = false }) { Text("Bekor qilish") }
            },
        )
    }

    removeTarget?.let { (identity, name) ->
        AlertDialog(
            onDismissRequest = { removeTarget = null },
            title = { Text("$name chiqarilsinmi?") },
            text = {
                Text(
                    "«Doimiy» — bu o'quvchi (shu ism bilan) sizning BARCHA darslaringizga " +
                        "qaytib kira olmaydi. Keyin Kabinet → Qora ro'yxat bo'limidan qaytarib olsangiz bo'ladi.",
                )
            },
            confirmButton = {
                Row {
                    TextButton(onClick = {
                        removeTarget = null
                        onRemove(identity, false)
                    }) { Text("Shu darsdan") }
                    TextButton(onClick = {
                        removeTarget = null
                        onRemove(identity, true)
                    }) { Text("Doimiy", color = MaterialTheme.colorScheme.error) }
                }
            },
            dismissButton = {
                TextButton(onClick = { removeTarget = null }) { Text("Bekor") }
            },
        )
    }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    if (searching) {
                        "Ishtirokchilar (${visible.size}/${state.roster.size})"
                    } else {
                        "Ishtirokchilar (${state.roster.size})"
                    },
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                TextButton(onClick = { muteAllOpen = true }) { Text("Hammani mute") }
            }

            // ── Qidiruv ───────────────────────────────────────────────────────
            // Katta darsda (200–300 kishi) aniq bir o'quvchini ko'z bilan topib
            // mute qilish imkonsiz. Maydon FAQAT ro'yxat kattalashganda chiqadi
            // (`ParticipantFilter.SEARCH_MIN_COUNT`) — kichik guruhda u shunchaki
            // joy egallab, klaviatura bilan ro'yxatni yopib qo'yardi.
            //
            // `|| searching`: ro'yxat qidiruv paytida qisqarib ketsa (kimdir
            // chiqib ketdi) maydon ichida yozuv bilan g'oyib bo'lmasin.
            if (ParticipantFilter.shouldShowSearch(state.roster.size) || searching) {
                OutlinedTextField(
                    value = query,
                    onValueChange = { query = it },
                    placeholder = { Text("Ism bo'yicha qidirish") },
                    leadingIcon = { Icon(Icons.Default.Search, contentDescription = null) },
                    trailingIcon = {
                        if (searching) {
                            IconButton(onClick = { query = "" }) {
                                Icon(Icons.Default.Close, contentDescription = "Tozalash")
                            }
                        }
                    },
                    singleLine = true,
                    modifier = Modifier.fillMaxWidth().padding(vertical = 8.dp),
                )
            }

            // Qidiruv paytida qo'l navbati YASHIRILADI: ustoz aniq bir odamni
            // qidirayotganda navbat ekranning yarmini egallab, natijani pastga
            // surib yuborardi. Qo'l ko'targanlar baribir ro'yxatda belgisi
            // bilan ko'rinadi.
            if (state.hands.isNotEmpty() && !searching) {
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

            if (visible.isEmpty()) {
                Text(
                    if (searching) {
                        "«${query.trim()}» bo'yicha hech kim topilmadi"
                    } else {
                        "Xonada hali hech kim yo'q"
                    },
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(vertical = 16.dp),
                )
            } else {
                LazyColumn(Modifier.heightIn(max = 360.dp)) {
                    items(visible, key = { it.identity }) { p ->
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
                            IconButton(onClick = { removeTarget = p.identity to p.name }) {
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
 *
 * Ikki moderatsiya/qulaylik amali shu yerda:
 *  · **uzoq bosish** → xabarni o'chirish (tasdiq bilan) — №6;
 *  · **📎** → fayl ulashish (rasm/PDF/hujjat, 20 MB gacha) — №15.
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ChatSheet(
    state: RoomUiState,
    onDismiss: () -> Unit,
    onSend: (String) -> Unit,
    onSendFile: (Uri) -> Unit,
    onDelete: (String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val ctx = LocalContext.current
    var draft by remember { mutableStateOf("") }
    val listState = rememberLazyListState()
    // O'chirish tasdig'i: (id, ko'rsatiladigan qisqa matn).
    var deleteTarget by remember { mutableStateOf<Pair<String, String>?>(null) }

    // `GetContent` — tizim fayl tanlagichi. Turi `*/*`: server allowlist'i
    // kengaytma bo'yicha ishlaydi va MIME filtri ba'zi provayderlarda hujjat
    // fayllarini butunlay yashirib qo'yadi (ustoz "faylim yo'q" deb o'ylardi).
    val picker = rememberLauncherForActivityResult(ActivityResultContracts.GetContent()) { uri ->
        if (uri != null) onSendFile(uri)
    }

    // Yangi xabar kelganda pastga suramiz — ustoz oxirgi savolni ko'rsin.
    LaunchedEffect(state.chat.size) {
        if (state.chat.isNotEmpty()) listState.animateScrollToItem(state.chat.lastIndex)
    }

    deleteTarget?.let { (id, preview) ->
        AlertDialog(
            onDismissRequest = { deleteTarget = null },
            title = { Text("Xabar o'chirilsinmi?") },
            text = {
                Column {
                    Text(preview, style = MaterialTheme.typography.bodyMedium, maxLines = 3, overflow = TextOverflow.Ellipsis)
                    Spacer(Modifier.height(8.dp))
                    Text(
                        "Xabar hammadan olib tashlanadi va o'rnida hech qanday iz " +
                            "qolmaydi. Bu amalni qaytarib bo'lmaydi.",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = {
                    deleteTarget = null
                    onDelete(id)
                }) { Text("O'chirish", color = MaterialTheme.colorScheme.error) }
            },
            dismissButton = {
                TextButton(onClick = { deleteTarget = null }) { Text("Bekor qilish") }
            },
        )
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
                    items(state.chat, key = { it.id }) { m ->
                        ChatBubble(
                            m = m,
                            onOpenFile = { url -> Share.openUrl(ctx, url) },
                            onLongPress = { deleteTarget = m.id to (m.body.ifBlank { m.file?.name.orEmpty() }) },
                        )
                    }
                }
            }

            if (state.chatUploading) {
                LinearProgressIndicator(Modifier.fillMaxWidth().padding(vertical = 4.dp))
            }

            Row(verticalAlignment = Alignment.CenterVertically) {
                IconButton(
                    onClick = { picker.launch(ChatUpload.PICKER_MIME) },
                    enabled = !state.chatUploading,
                ) {
                    Icon(Icons.Default.AttachFile, contentDescription = "Fayl biriktirish")
                }
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

            Text(
                "Uzoq bosib turib xabarni o'chirish mumkin",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 4.dp),
            )
        }
    }
}

@OptIn(ExperimentalFoundationApi::class)
@Composable
private fun ChatBubble(
    m: ChatMessageUi,
    onOpenFile: (String) -> Unit,
    onLongPress: () -> Unit,
) {
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
                // Uzoq bosish — moderatsiya menyusi o'rniga. Zoom'da ham
                // o'chirish xabarning O'ZIDAN chaqiriladi; alohida "boshqarish"
                // rejimi 40 kishilik xonada ortiqcha qadam bo'lardi.
                modifier = Modifier.combinedClickable(
                    onClick = { m.file?.url?.takeIf { it.isNotBlank() }?.let(onOpenFile) },
                    onLongClick = onLongPress,
                ),
            ) {
                Column(Modifier.padding(horizontal = 10.dp, vertical = 6.dp)) {
                    if (m.body.isNotBlank()) {
                        Text(
                            m.body,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (m.self) {
                                MaterialTheme.colorScheme.onPrimary
                            } else {
                                MaterialTheme.colorScheme.onSurface
                            },
                        )
                    }
                    m.file?.let { file ->
                        if (m.body.isNotBlank()) Spacer(Modifier.height(4.dp))
                        ChatFileRow(file = file, onSelf = m.self)
                    }
                }
            }
        }
    }
}

/**
 * Fayl kartochkasi. Bosilganda tashqi ilovada ochiladi (`ACTION_VIEW`).
 *
 * NEGA ILOVA ICHIDA EMAS: fayl PDF, Office hujjati yoki rasm bo'lishi mumkin
 * va ularning har biri uchun ko'ruvchi yozish — tizimda allaqachon bor
 * narsaning yomonroq nusxasi. Yozuvlar ekrani ham xuddi shu qarorni oladi.
 */
@Composable
private fun ChatFileRow(file: ChatFileUi, onSelf: Boolean) {
    val fg = if (onSelf) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurface
    Row(verticalAlignment = Alignment.CenterVertically) {
        Text(ChatUpload.icon(file.name), style = MaterialTheme.typography.bodyLarge)
        Spacer(Modifier.width(6.dp))
        Column(Modifier.widthIn(max = 200.dp)) {
            Text(
                file.name,
                style = MaterialTheme.typography.bodyMedium,
                color = fg,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            Text(
                if (file.size > 0) "${ChatUpload.sizeLabel(file.size)} · ochish uchun bosing" else "Ochish uchun bosing",
                style = MaterialTheme.typography.labelSmall,
                color = fg.copy(alpha = 0.75f),
            )
        }
    }
}
