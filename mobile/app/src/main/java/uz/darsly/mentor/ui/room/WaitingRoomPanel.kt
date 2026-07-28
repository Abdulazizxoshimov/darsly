package uz.darsly.mentor.ui.room

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import uz.darsly.mentor.data.api.WaitingRoomRequest
import uz.darsly.mentor.ui.profile.ProfileForm
import uz.darsly.mentor.ui.theme.DarslyTheme

/**
 * Kutish xonasi paneli — xona ekranining yuqorisida (M27).
 *
 * ## Xulq qoidalari (Zoom mezoni)
 * · Kutayotgan **yo'q bo'lsa panel umuman ko'rinmaydi** — bo'sh "0 kishi
 *   kutmoqda" qatori sahnadan joy o'g'irlagan bo'lardi.
 * · Kimdir kelsa panel **o'zi ochiladi**: ustoz ekran ulashib turganda uni
 *   qidirib topishga vaqti yo'q.
 * · Ustoz panelni yiqsa, **yopiq qoladi** — lekin sarlavhadagi son ko'rinib
 *   turadi, ya'ni ma'lumot yo'qolmaydi.
 * · Ro'yxat balandligi cheklangan: 30 kishi kirmoqchi bo'lsa panel butun
 *   ekranni egallamasligi kerak, dars davom etyapti.
 */
@Composable
fun WaitingRoomPanel(
    state: WaitingRoomUiState,
    onAdmit: (WaitingRoomRequest) -> Unit,
    onReject: (WaitingRoomRequest) -> Unit,
    onAdmitAll: () -> Unit,
    modifier: Modifier = Modifier,
) {
    if (state.isEmpty) return

    // Yig'ish tanlovi ustoz qo'lida, lekin YANGI so'rov kelganda panel o'zi
    // ochiladi.
    //
    // Bu `LaunchedEffect` ichida bo'lishi SHART: kompozitsiya vaqtida holat
    // yozish (`if (count > last) collapsed = false`) Compose'da cheksiz qayta
    // kompozitsiyaga olib keladigan klassik xato. Yon ta'sir — effekt ishi.
    var collapsed by rememberSaveable { mutableStateOf(false) }
    var lastCount by rememberSaveable { mutableIntStateOf(0) }
    LaunchedEffect(state.count) {
        if (state.count > lastCount) collapsed = false
        lastCount = state.count
    }

    Card(
        modifier.fillMaxWidth().padding(horizontal = 12.dp, vertical = 6.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceContainerHigh,
        ),
    ) {
        Column(Modifier.padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                CountBadge(state.count)
                Spacer(Modifier.width(10.dp))
                Text(
                    WaitingRoomState.title(state.count),
                    style = MaterialTheme.typography.titleSmall,
                    modifier = Modifier.weight(1f),
                )
                if (state.count > 1 && !collapsed) {
                    TextButton(onClick = onAdmitAll) { Text("Hammasini kiritish") }
                }
                IconButton(onClick = { collapsed = !collapsed }) {
                    Icon(
                        if (collapsed) Icons.Default.ExpandMore else Icons.Default.ExpandLess,
                        contentDescription = if (collapsed) "Ochish" else "Yig'ish",
                    )
                }
            }

            AnimatedVisibility(visible = !collapsed) {
                LazyColumn(
                    // Panel sahnani bosib ketmasin — dars davom etyapti.
                    Modifier.heightIn(max = 220.dp),
                    verticalArrangement = Arrangement.spacedBy(6.dp),
                ) {
                    items(state.requests, key = { it.id }) { request ->
                        RequestRow(
                            request = request,
                            deciding = state.isDeciding(request.id),
                            onAdmit = { onAdmit(request) },
                            onReject = { onReject(request) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun CountBadge(count: Int) {
    Surface(
        shape = CircleShape,
        color = MaterialTheme.colorScheme.primary,
        modifier = Modifier.size(28.dp),
    ) {
        Row(
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                count.toString(),
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.onPrimary,
            )
        }
    }
}

@Composable
private fun RequestRow(
    request: WaitingRoomRequest,
    deciding: Boolean,
    onAdmit: () -> Unit,
    onReject: () -> Unit,
) {
    Surface(
        shape = MaterialTheme.shapes.medium,
        color = MaterialTheme.colorScheme.surface,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Row(
            Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(8.dp),
        ) {
            Surface(
                shape = CircleShape,
                color = MaterialTheme.colorScheme.surfaceVariant,
                modifier = Modifier.size(32.dp),
            ) {
                Row(
                    horizontalArrangement = Arrangement.Center,
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        ProfileForm.initials(request.requesterName),
                        style = MaterialTheme.typography.labelMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }

            Column(Modifier.weight(1f)) {
                Text(
                    request.requesterName.ifBlank { "O'quvchi" },
                    style = MaterialTheme.typography.bodyMedium,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
                WaitingRoomState.waitingLabel(request)?.let {
                    Text(
                        it,
                        style = MaterialTheme.typography.labelSmall,
                        color = DarslyTheme.colors.textMuted,
                    )
                }
            }

            if (deciding) {
                CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp)
            } else {
                // "Kiritish" — asosiy amal, shuning uchun to'ldirilgan tugma;
                // "Rad etish" — ikkilamchi va faqat ikonka, tasodifan bosilishi
                // kamroq bo'lsin (bosilsa o'quvchi darsdan tashqarida qoladi).
                OutlinedButton(
                    onClick = onReject,
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(
                        horizontal = 10.dp,
                        vertical = 4.dp,
                    ),
                ) {
                    Icon(
                        Icons.Default.Close,
                        contentDescription = "Rad etish",
                        modifier = Modifier.size(16.dp),
                        tint = MaterialTheme.colorScheme.error,
                    )
                }
                Button(
                    onClick = onAdmit,
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(
                        horizontal = 12.dp,
                        vertical = 4.dp,
                    ),
                ) {
                    Icon(Icons.Default.Check, contentDescription = null, modifier = Modifier.size(16.dp))
                    Spacer(Modifier.width(4.dp))
                    Text("Kiritish", style = MaterialTheme.typography.labelMedium)
                }
            }
        }
    }
}
