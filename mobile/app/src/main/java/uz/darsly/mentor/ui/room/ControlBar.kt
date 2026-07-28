package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Chat
import androidx.compose.material.icons.filled.People
import androidx.compose.material.icons.filled.CallEnd
import androidx.compose.material.icons.filled.Cameraswitch
import androidx.compose.material.icons.filled.Mic
import androidx.compose.material.icons.filled.MicOff
import androidx.compose.material.icons.filled.ScreenShare
import androidx.compose.material.icons.filled.StopScreenShare
import androidx.compose.material.icons.filled.Videocam
import androidx.compose.material.icons.filled.VideocamOff
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Xona boshqaruv paneli (R-2).
 *
 * ## Nega vertikal tugmalar ro'yxati almashtirildi
 * Avval ekranda uzun matnli tugmalar ustma-ust turardi ("Mikrofonni o'chirish",
 * "Kamerani o'chirish", "Orqa kameraga") — bu sozlamalar ro'yxatiga o'xshardi va
 * ekranning yarmini egallardi. Zoom, Meet va Teams — hammasi **pastdagi gorizontal
 * ikonka paneli**ni ishlatadi: bitta qarashda holat ko'rinadi, barmoq yetadigan joyda.
 *
 * ## Rang mantiqi (web dizayn tizimi bilan bir xil)
 *  · Faol/normal — `surfaceVariant` doira
 *  · **O'chirilgan** mikrofon/kamera — `error` (qizil), chunki bu **ogohlantiruvchi holat**
 *  · Ekran ulashish yoniq — `primary` (brend binafshasi), ya'ni "hozir efirda"
 *  · Chiqish — doim qizil
 */
private const val ITEM_COUNT = 7

@Composable
fun ControlBar(
    micOn: Boolean,
    camOn: Boolean,
    screenOn: Boolean,
    enabled: Boolean,
    onToggleMic: () -> Unit,
    onToggleCam: () -> Unit,
    onFlipCamera: () -> Unit,
    onToggleShare: () -> Unit,
    onLeave: () -> Unit,
    onOpenParticipants: () -> Unit,
    onOpenChat: () -> Unit,
    participantCount: Int = 0,
    handsCount: Int = 0,
    unreadChat: Int = 0,
    modifier: Modifier = Modifier,
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        modifier = modifier.fillMaxWidth(),
    ) {
        // ⚠️ O'LCHAM EKRANGA MOSLASHADI.
        //
        // Qurilma sinovida (2026-07-28) topildi: panelda 7 ta tugma bor va ular
        // qat'iy 52dp bo'lganda TELEFONDA (portret) sig'masdi — oxirgi tugmaning
        // yozuvi vertikal cho'zilib, o'zi ekran chetidan chiqib ketardi.
        // Planshetda (landshaft) muammo ko'rinmasdi, ya'ni faqat kichik ekranda
        // chiqadigan regressiya edi.
        //
        // Endi tugma o'lchami mavjud endan hisoblanadi: hamma narsa har qanday
        // ekranda sig'adi, keng ekranda esa avvalgidek qulay o'lchamda qoladi.
        BoxWithConstraints(Modifier.fillMaxWidth()) {
            val slot = (maxWidth - 16.dp) / ITEM_COUNT
            val button = (slot - 6.dp).coerceIn(38.dp, 52.dp)
            val compact = slot < 56.dp

            Row(
                Modifier.fillMaxWidth().padding(vertical = 12.dp, horizontal = 8.dp),
                horizontalArrangement = Arrangement.SpaceEvenly,
                verticalAlignment = Alignment.CenterVertically,
            ) {
            ControlButton(
                icon = if (micOn) Icons.Default.Mic else Icons.Default.MicOff,
                label = if (micOn) "Mikrofon" else "Ovozsiz",
                active = !micOn,
                activeColor = MaterialTheme.colorScheme.error,
                enabled = enabled,
                onClick = onToggleMic,
                size = button,
                compact = compact,
            )
            ControlButton(
                icon = if (camOn) Icons.Default.Videocam else Icons.Default.VideocamOff,
                label = if (camOn) "Kamera" else "Kamerasiz",
                active = !camOn,
                activeColor = MaterialTheme.colorScheme.error,
                enabled = enabled,
                onClick = onToggleCam,
                size = button,
                compact = compact,
            )
            ControlButton(
                icon = Icons.Default.Cameraswitch,
                label = "Almashtirish",
                active = false,
                // Kamera o'chiq bo'lsa almashtirishning ma'nosi yo'q.
                enabled = enabled && camOn,
                onClick = onFlipCamera,
                size = button,
                compact = compact,
            )
            ControlButton(
                icon = if (screenOn) Icons.Default.StopScreenShare else Icons.Default.ScreenShare,
                label = if (screenOn) "To'xtatish" else "Ekran",
                active = screenOn,
                activeColor = MaterialTheme.colorScheme.primary,
                enabled = enabled,
                onClick = onToggleShare,
                size = button,
                compact = compact,
            )
            // Ishtirokchilar — nishonda qo'l ko'targanlar soni ko'rinadi:
            // ustoz panelni ochmasdan ham "kimdir so'rayapti" ni biladi.
            ControlButton(
                icon = Icons.Default.People,
                label = "Ishtirokchi",
                active = handsCount > 0,
                activeColor = MaterialTheme.colorScheme.primary,
                enabled = enabled,
                badge = if (handsCount > 0) handsCount else participantCount.takeIf { it > 0 },
                onClick = onOpenParticipants,
                size = button,
                compact = compact,
            )
            ControlButton(
                icon = Icons.AutoMirrored.Filled.Chat,
                label = "Chat",
                active = unreadChat > 0,
                activeColor = MaterialTheme.colorScheme.primary,
                enabled = enabled,
                badge = unreadChat.takeIf { it > 0 },
                onClick = onOpenChat,
                size = button,
                compact = compact,
            )
            ControlButton(
                icon = Icons.Default.CallEnd,
                label = "Chiqish",
                active = true,
                activeColor = MaterialTheme.colorScheme.error,
                enabled = true,
                onClick = onLeave,
                size = button,
                compact = compact,
            )
            }
        }
    }
}

@Composable
private fun ControlButton(
    icon: ImageVector,
    label: String,
    active: Boolean,
    enabled: Boolean,
    onClick: () -> Unit,
    activeColor: Color = MaterialTheme.colorScheme.primary,
    /** Tugma ustidagi kichik son (o'qilmagan xabar / qo'l ko'targanlar). */
    badge: Int? = null,
    /** Ekranga moslashgan tugma diametri. */
    size: Dp = 52.dp,
    /** Tor ekran: yozuv kichikroq va bir qatorda qoladi. */
    compact: Boolean = false,
) {
    val bg = when {
        !enabled -> MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)
        active -> activeColor
        else -> MaterialTheme.colorScheme.surfaceVariant
    }
    val fg = when {
        !enabled -> MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
        active -> Color.White
        else -> MaterialTheme.colorScheme.onSurface
    }

    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Box {
            IconButton(
                onClick = onClick,
                enabled = enabled,
                modifier = Modifier.size(size).clip(CircleShape).background(bg),
            ) {
                Icon(icon, contentDescription = label, tint = fg)
            }
            if (badge != null && badge > 0) {
                Box(
                    Modifier
                        .align(Alignment.TopEnd)
                        .size(20.dp)
                        .clip(CircleShape)
                        .background(MaterialTheme.colorScheme.error),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        if (badge > 9) "9+" else badge.toString(),
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.onError,
                    )
                }
            }
        }
        Text(
            label,
            style = if (compact) {
                MaterialTheme.typography.labelSmall.copy(fontSize = 9.sp, lineHeight = 11.sp)
            } else {
                MaterialTheme.typography.labelSmall
            },
            color = if (enabled) {
                MaterialTheme.colorScheme.onSurfaceVariant
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
            },
            maxLines = 1,
            overflow = TextOverflow.Ellipsis,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(top = 4.dp).width(size + 6.dp),
        )
    }
}
