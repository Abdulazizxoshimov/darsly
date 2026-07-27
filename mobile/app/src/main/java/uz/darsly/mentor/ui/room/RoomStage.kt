package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
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
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import io.livekit.android.room.Room
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
