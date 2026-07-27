package uz.darsly.mentor.ui.room

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
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
import androidx.compose.ui.unit.dp

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
    modifier: Modifier = Modifier,
) {
    Surface(
        color = MaterialTheme.colorScheme.surface,
        modifier = modifier.fillMaxWidth(),
    ) {
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
            )
            ControlButton(
                icon = if (camOn) Icons.Default.Videocam else Icons.Default.VideocamOff,
                label = if (camOn) "Kamera" else "Kamerasiz",
                active = !camOn,
                activeColor = MaterialTheme.colorScheme.error,
                enabled = enabled,
                onClick = onToggleCam,
            )
            ControlButton(
                icon = Icons.Default.Cameraswitch,
                label = "Almashtirish",
                active = false,
                // Kamera o'chiq bo'lsa almashtirishning ma'nosi yo'q.
                enabled = enabled && camOn,
                onClick = onFlipCamera,
            )
            ControlButton(
                icon = if (screenOn) Icons.Default.StopScreenShare else Icons.Default.ScreenShare,
                label = if (screenOn) "To'xtatish" else "Ekran",
                active = screenOn,
                activeColor = MaterialTheme.colorScheme.primary,
                enabled = enabled,
                onClick = onToggleShare,
            )
            ControlButton(
                icon = Icons.Default.CallEnd,
                label = "Chiqish",
                active = true,
                activeColor = MaterialTheme.colorScheme.error,
                enabled = true,
                onClick = onLeave,
            )
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
        IconButton(
            onClick = onClick,
            enabled = enabled,
            modifier = Modifier.size(52.dp).clip(CircleShape).background(bg),
        ) {
            Icon(icon, contentDescription = label, tint = fg)
        }
        Text(
            label,
            style = MaterialTheme.typography.labelSmall,
            color = if (enabled) {
                MaterialTheme.colorScheme.onSurfaceVariant
            } else {
                MaterialTheme.colorScheme.onSurfaceVariant.copy(alpha = 0.4f)
            },
            modifier = Modifier.padding(top = 4.dp),
        )
    }
}
