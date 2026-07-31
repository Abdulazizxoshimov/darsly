@file:OptIn(ExperimentalMaterial3Api::class)

package uz.darsly.mentor.ui.room

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Surface
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import kotlinx.coroutines.delay
import uz.darsly.mentor.data.api.Poll

/** Izoh ekranda qancha turadi (snackbar'ning odatiy `Short` muddati). */
private const val NOTICE_MS = 3_000L

/**
 * So'rovnoma paneli (№7) — mentor uchun.
 *
 * Ikki rejim: ro'yxat (mavjud so'rovnomalar + natija) va yaratish formasi.
 * Ular bitta panelda, chunki ustozning bu yerdagi ishi bir zanjir: "yangi
 * savol bergim keldi → yubordim → natijasini ko'raman → e'lon qilaman".
 *
 * @param roomToken jonli natijani o'qish uchun host room-tokeni. `null` bo'lsa
 *   (masalan hali ulanmagan) natija ko'rsatilmaydi, lekin yaratish ishlaydi.
 * @param revision `poll_published` hodisasi hisoblagichi — boshqa qurilmadan
 *   e'lon qilinsa panel eskirib qolmasin.
 */
@Composable
fun PollSheet(
    lessonId: String,
    roomToken: String?,
    revision: Int,
    onDismiss: () -> Unit,
    vm: PollViewModel = hiltViewModel(),
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val state by vm.state.collectAsStateWithLifecycle()
    var composing by remember { mutableStateOf(false) }

    LaunchedEffect(lessonId, roomToken) { vm.start(lessonId, roomToken) }
    // Xonadagi e'lon (masalan web'dan) — ro'yxatni qayta so'raymiz.
    LaunchedEffect(revision) { if (revision > 0) vm.refresh() }

    ModalBottomSheet(onDismissRequest = onDismiss, sheetState = sheetState) {
        Column(Modifier.padding(horizontal = 16.dp).padding(bottom = 24.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    "So'rovnoma",
                    style = MaterialTheme.typography.titleMedium,
                    modifier = Modifier.weight(1f),
                )
                if (!composing) {
                    TextButton(onClick = { composing = true }) {
                        Icon(Icons.Default.Add, contentDescription = null, modifier = Modifier.size(18.dp))
                        Spacer(Modifier.width(4.dp))
                        Text("Yangi")
                    }
                }
            }

            // Amal natijasi. Panel ichida — snackbar `ModalBottomSheet` ortida
            // qolib ketardi (u Scaffold'dan yuqorida chiziladi). O'zi
            // yo'qoladi: qo'lda yopish uchun tugma qo'yish bir qatorlik izoh
            // uchun ortiqcha shovqin bo'lardi.
            state.notice?.let { text ->
                Text(
                    text,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.primary,
                    modifier = Modifier.padding(vertical = 4.dp),
                )
                LaunchedEffect(text) {
                    delay(NOTICE_MS)
                    vm.noticeShown()
                }
            }

            if (composing) {
                PollComposer(
                    creating = state.creating,
                    onCancel = { composing = false },
                    onSubmit = { q, opts, isPublic ->
                        vm.create(q, opts, isPublic)
                        composing = false
                    },
                )
                Spacer(Modifier.height(12.dp))
            }

            when {
                state.loading && state.polls.isEmpty() ->
                    CircularProgressIndicator(Modifier.padding(vertical = 24.dp))

                state.error != null && state.polls.isEmpty() -> {
                    Text(
                        state.error!!,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.error,
                        modifier = Modifier.padding(vertical = 12.dp),
                    )
                    OutlinedButton(onClick = { vm.refresh() }) { Text("Qayta urinish") }
                }

                state.polls.isEmpty() && !composing -> Text(
                    "Hali so'rovnoma yo'q. «Yangi» tugmasi bilan savol bering — " +
                        "o'quvchilar uni xonada ko'radi.",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(vertical = 16.dp),
                )

                else -> LazyColumn(
                    Modifier.heightIn(max = 420.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(state.polls, key = { it.id }) { poll ->
                        PollCard(
                            poll = poll,
                            results = state.results[poll.id],
                            busy = state.busyId == poll.id,
                            onPublish = { vm.publish(poll) },
                            onClose = { vm.close(poll) },
                        )
                    }
                }
            }
        }
    }
}

/**
 * Yaratish formasi.
 *
 * «Natija kimga ko'rinadi» — ATAYLAB shu yerda va ATAYLAB default'i yopiq
 * tomonda: rejim yaratilgandan keyin O'ZGARMAYDI (server qoidasi), ya'ni
 * tasodifan "hammaga" qilib yuborilgan so'rovnomani orqaga qaytarib
 * bo'lmaydi. Nazorat savoli — reyting emas.
 */
@Composable
private fun PollComposer(
    creating: Boolean,
    onCancel: () -> Unit,
    onSubmit: (String, List<String>, Boolean) -> Unit,
) {
    var question by remember { mutableStateOf("") }
    val options = remember { mutableStateListOf("", "") }
    var publicResults by remember { mutableStateOf(false) }
    var showErrors by remember { mutableStateOf(false) }

    val questionError = if (showErrors) PollForm.questionError(question) else null
    val optionsError = if (showErrors) PollForm.optionsError(options) else null

    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(14.dp)) {
            OutlinedTextField(
                value = question,
                onValueChange = { question = it },
                label = { Text("Savol") },
                isError = questionError != null,
                supportingText = questionError?.let { { Text(it) } },
                modifier = Modifier.fillMaxWidth(),
            )
            Spacer(Modifier.height(8.dp))

            options.forEachIndexed { i, value ->
                Row(verticalAlignment = Alignment.CenterVertically) {
                    OutlinedTextField(
                        value = value,
                        onValueChange = { options[i] = it },
                        label = { Text("${i + 1}-variant") },
                        singleLine = true,
                        modifier = Modifier.weight(1f),
                    )
                    // Kamida ikkita variant bo'lishi shart — o'chirish tugmasi
                    // shundan ortiqlarida chiqadi (bo'lmasa "ikkitasini ham
                    // o'chirdim, endi nima" holati bo'lardi).
                    if (options.size > PollForm.MIN_OPTIONS) {
                        IconButton(onClick = { options.removeAt(i) }) {
                            Icon(Icons.Default.Close, contentDescription = "Variantni o'chirish")
                        }
                    }
                }
                Spacer(Modifier.height(4.dp))
            }
            optionsError?.let {
                Text(it, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.error)
            }

            if (options.size < PollForm.MAX_OPTIONS) {
                TextButton(onClick = { options.add("") }) {
                    Icon(Icons.Default.Add, contentDescription = null, modifier = Modifier.size(18.dp))
                    Spacer(Modifier.width(4.dp))
                    Text("Variant qo'shish")
                }
            }

            Spacer(Modifier.height(4.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Column(Modifier.weight(1f)) {
                    Text("Natija hammaga ko'rinsin", style = MaterialTheme.typography.bodyMedium)
                    Text(
                        if (publicResults) {
                            "Siz «E'lon qilish» bosganingizdan keyin o'quvchilar ham ko'radi"
                        } else {
                            "Natijani faqat siz ko'rasiz (keyin o'zgartirib bo'lmaydi)"
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                Switch(checked = publicResults, onCheckedChange = { publicResults = it })
            }

            Spacer(Modifier.height(10.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                Button(
                    onClick = {
                        showErrors = true
                        if (PollForm.canSubmit(question, options)) {
                            onSubmit(question, options.toList(), publicResults)
                        }
                    },
                    enabled = !creating,
                ) {
                    if (creating) {
                        CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                    } else {
                        Text("Yuborish")
                    }
                }
                OutlinedButton(onClick = onCancel, enabled = !creating) { Text("Bekor qilish") }
            }
        }
    }
}

@Composable
private fun PollCard(
    poll: Poll,
    results: uz.darsly.mentor.data.api.PollResults?,
    busy: Boolean,
    onPublish: () -> Unit,
    onClose: () -> Unit,
) {
    Card(Modifier.fillMaxWidth()) {
        Column(Modifier.padding(14.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text(
                    poll.question,
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.weight(1f),
                )
                Surface(
                    shape = RoundedCornerShape(percent = 50),
                    color = if (poll.isActive) {
                        MaterialTheme.colorScheme.primaryContainer
                    } else {
                        MaterialTheme.colorScheme.surfaceVariant
                    },
                ) {
                    Text(
                        PollForm.statusLabel(poll),
                        style = MaterialTheme.typography.labelSmall,
                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
                    )
                }
            }

            Text(
                PollForm.visibilityHint(poll),
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 2.dp),
            )

            Spacer(Modifier.height(8.dp))
            val counts = results?.counts.orEmpty()
            val total = results?.total ?: 0
            poll.options.forEachIndexed { i, option ->
                val count = counts.getOrNull(i) ?: 0
                val percent = PollForm.percent(count, total)
                Column(Modifier.padding(vertical = 3.dp)) {
                    Row {
                        Text(
                            option,
                            style = MaterialTheme.typography.bodyMedium,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.weight(1f),
                        )
                        Text(
                            "$count · $percent%",
                            style = MaterialTheme.typography.labelMedium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                    LinearProgressIndicator(
                        progress = { percent / 100f },
                        modifier = Modifier.fillMaxWidth().height(4.dp).padding(top = 2.dp),
                    )
                }
            }
            Text(
                "Jami $total ovoz",
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 4.dp),
            )

            if (PollForm.canPublish(poll) || PollForm.canClose(poll)) {
                Spacer(Modifier.height(10.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    // «E'lon qilish» — FAQAT `public` so'rovnomada va faqat bir
                    // marta. `mentor_only` da server 400 berardi, ya'ni tugmani
                    // ko'rsatish ustozni boshi berk ko'chaga olib borardi.
                    if (PollForm.canPublish(poll)) {
                        Button(onClick = onPublish, enabled = !busy) {
                            if (busy) {
                                CircularProgressIndicator(Modifier.size(16.dp), strokeWidth = 2.dp)
                            } else {
                                Text("E'lon qilish")
                            }
                        }
                    }
                    if (PollForm.canClose(poll)) {
                        OutlinedButton(onClick = onClose, enabled = !busy) { Text("Yopish") }
                    }
                }
            }
        }
    }
}
