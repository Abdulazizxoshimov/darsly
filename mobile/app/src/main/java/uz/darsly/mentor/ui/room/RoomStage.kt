package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.ScreenShare
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import io.livekit.android.room.Room
import uz.darsly.mentor.data.livekit.RaisedHand
import uz.darsly.mentor.data.livekit.RoomReaction
import uz.darsly.mentor.ui.theme.DarslyTheme

/**
 * Xonaning asosiy sahnasi (M20 · R-1).
 *
 * ## Uchta holat
 *  1. **Ekran ulashilmoqda** — eng katta joyni "efirdasiz" kartasi egallaydi.
 *     Ustozning o'z ekranini qayta ko'rsatish **ataylab qilinmaydi**: bu cheksiz
 *     ko'zgu effekti beradi (ekranda ekran ichida ekran) va GPU'ni bekorga yeydi.
 *     Zoom ham host'ga "You are sharing your screen" deb yozadi, kadrni qaytarmaydi.
 *  2. **O'quvchilar bor** — video plitkalari to'ri (1 ta bo'lsa katta, ko'p bo'lsa 2 ustun).
 *  3. **Hech kim yo'q** — nima qilish kerakligini aytadigan bo'sh holat
 *     ("havolani ulashing"), chunki ustoz birinchi marta kirganda ekran bo'm-bo'sh
 *     qolsa "ishlamayapti" deb o'ylaydi.
 */
@Composable
fun RoomStage(
    room: Room?,
    state: RoomUiState,
    modifier: Modifier = Modifier,
) {
    Box(modifier.fillMaxSize()) {
        when {
            state.screenOn -> ScreenSharingBanner(state)
            state.participants.isEmpty() -> EmptyStage()
            else -> ParticipantGrid(room, state)
        }

        // ⭐ Ekran ulashish RAMKASI — Zoom'dagi kabi chiziqli chegara.
        //
        // Ilova ichida chiziladi (butun OS ekrani atrofida emas): tizim darajasidagi
        // ramka `SYSTEM_ALERT_WINDOW` talab qiladi VA `MediaProjection` uni ham
        // yozib olardi — ya'ni o'quvchilar ustozning ramkasini ko'rib turardi.
        // Ilova ichidagi ramka esa ustoz ilovaga qaytganda "ulashish hali ham
        // ketyapti" degan signalni beradi; fonda esa buni bildirishnoma bajaradi.
        if (state.screenOn) {
            Box(
                Modifier
                    .fillMaxSize()
                    .border(3.dp, MaterialTheme.colorScheme.error, RoundedCornerShape(6.dp)),
            )
        }

        // Dars signallari — sahna USTIDA, chap tepada. Ekran ulashilayotgan
        // paytda ham ko'rinadi (o'sha holatda sahna faqat banner). Ilova fonda
        // bo'lsa bu ro'yxat ko'rinmaydi — o'shanda bildirishnoma ishlaydi
        // (`LessonNotifications.updateSignals`).
        SignalsOverlay(
            hands = state.hands,
            reactions = state.reactions,
            modifier = Modifier.align(Alignment.TopStart).padding(12.dp),
        )

        // O'z kameramiz — kichik suzuvchi oyna (Zoom'dagi kabi o'ng pastda).
        if (room != null && state.camOn && state.localVideo != null) {
            VideoTile(
                room = room,
                track = state.localVideo,
                label = "Siz",
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(12.dp)
                    .width(96.dp)
                    .height(140.dp)
                    .border(1.dp, MaterialTheme.colorScheme.outline, RoundedCornerShape(14.dp)),
            )
        }
    }
}

@Composable
private fun ScreenSharingBanner(state: RoomUiState) {
    Column(
        Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Box(
            Modifier
                .size(72.dp)
                .clip(CircleShape)
                .background(MaterialTheme.colorScheme.primary),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Default.ScreenShare,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.onPrimary,
                modifier = Modifier.size(36.dp),
            )
        }
        Spacer(Modifier.height(16.dp))
        Text("Ekraningiz ulashilmoqda", style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(6.dp))
        Text(
            "O'quvchilar telefoningiz ekranini ko'rib turibdi. " +
                "Boshqa ilovaga o'tsangiz ham ulashish davom etadi.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
        if (state.screenAudioOn) {
            Spacer(Modifier.height(12.dp))
            Text(
                "🔊 Ekran ovozi ham uzatilmoqda",
                style = MaterialTheme.typography.labelMedium,
                color = DarslyTheme.colors.success,
            )
        }
        Spacer(Modifier.height(20.dp))
        Text(
            "${state.participantCount - 1} o'quvchi",
            style = MaterialTheme.typography.labelLarge,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun EmptyStage() {
    Column(
        Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Hali hech kim qo'shilmadi", style = MaterialTheme.typography.titleMedium)
        Spacer(Modifier.height(6.dp))
        Text(
            "Dars havolasini o'quvchilarga yuboring — ular qo'shilganda shu yerda ko'rinadi.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
    }
}

@Composable
private fun ParticipantGrid(room: Room?, state: RoomUiState) {
    // Bitta o'quvchi bo'lsa — bitta katta plitka; ko'p bo'lsa 2 ustun.
    val columns = if (state.participants.size == 1) 1 else 2
    LazyVerticalGrid(
        columns = GridCells.Fixed(columns),
        modifier = Modifier.fillMaxSize().padding(8.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(state.participants, key = { it.identity }) { p ->
            Box(
                Modifier
                    .fillMaxWidth()
                    .aspectRatio(if (columns == 1) 0.75f else 0.8f),
            ) {
                if (room != null) {
                    VideoTile(
                        room = room,
                        track = p.videoTrack,
                        label = p.name,
                        placeholder = p.name,
                        modifier = Modifier
                            .fillMaxSize()
                            .then(
                                // Gapirayotgan o'quvchi brend rangi bilan ajratiladi —
                                // ustoz kim gapirayotganini bir qarashda ko'radi.
                                if (p.speaking) {
                                    Modifier.border(
                                        2.dp,
                                        MaterialTheme.colorScheme.primary,
                                        RoundedCornerShape(14.dp),
                                    )
                                } else {
                                    Modifier
                                }
                            ),
                    )
                }
                if (p.micMuted) {
                    Row(
                        Modifier
                            .align(Alignment.TopEnd)
                            .padding(8.dp)
                            .clip(CircleShape)
                            .background(MaterialTheme.colorScheme.error)
                            .padding(4.dp),
                    ) {
                        Icon(
                            Icons.Default.MicOff,
                            contentDescription = "Mikrofoni o'chiq",
                            tint = MaterialTheme.colorScheme.onError,
                            modifier = Modifier.size(14.dp),
                        )
                    }
                }
            }
        }
    }
}

/**
 * Qo'l navbati va oxirgi reaksiyalar — sahna ustidagi yengil qatlam.
 *
 * Hech narsa bo'lmasa UMUMAN chizilmaydi: bo'sh karta sahnadan joy yeb, ustozning
 * diqqatini bekorga tortadi.
 */
@Composable
private fun SignalsOverlay(
    hands: List<RaisedHand>,
    reactions: List<RoomReaction>,
    modifier: Modifier = Modifier,
) {
    if (hands.isEmpty() && reactions.isEmpty()) return
    Column(
        modifier = modifier.widthIn(max = 200.dp),
        verticalArrangement = Arrangement.spacedBy(6.dp),
    ) {
        if (hands.isNotEmpty()) {
            SignalCard {
                Text(
                    "✋ Qo'l ko'targanlar (${hands.size})",
                    style = MaterialTheme.typography.labelSmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                // Faqat birinchi uchtasi: navbat uzun bo'lsa ham ekranni to'ldirmasin.
                hands.take(3).forEachIndexed { i, h ->
                    Text(
                        "${i + 1}. ${h.name}",
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
                if (hands.size > 3) {
                    Text(
                        "va yana ${hands.size - 3} ta",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }
        if (reactions.isNotEmpty()) {
            SignalCard {
                reactions.take(3).forEach { r ->
                    Text(
                        "${r.emoji} ${r.name}",
                        style = MaterialTheme.typography.bodySmall,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }
        }
    }
}

@Composable
private fun SignalCard(content: @Composable ColumnScope.() -> Unit) {
    Surface(
        color = MaterialTheme.colorScheme.surface.copy(alpha = 0.92f),
        shape = RoundedCornerShape(12.dp),
        tonalElevation = 3.dp,
    ) {
        Column(
            modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp),
            verticalArrangement = Arrangement.spacedBy(2.dp),
            content = content,
        )
    }
}
