package uz.darsly.mentor.ui.common

import androidx.compose.animation.core.animateFloatAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.LocalIndication
import androidx.compose.foundation.clickable
import androidx.compose.foundation.interaction.MutableInteractionSource
import androidx.compose.foundation.interaction.collectIsPressedAsState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.composed
import androidx.compose.ui.draw.scale
import androidx.compose.ui.hapticfeedback.HapticFeedback
import androidx.compose.ui.hapticfeedback.HapticFeedbackType
import androidx.compose.ui.platform.LocalHapticFeedback

/**
 * Mikro-interaksiya — butun ilova uchun YAGONA manba.
 *
 * Auditda topilgani: ilovada haptika 0 ta, bosilish animatsiyasi yo'q edi — bu
 * "arzon" his berardi. Zamonaviy Android (Material 3 Expressive) tugma bosilganda
 * yengil titrash + kichik "cho'kish" (scale 0.96) beradi. Bu shu yerda bir marta
 * yozilib, har ekranda qayta ishlatiladi (har joyda qo'lda emas).
 *
 * Diqqat: Compose 1.7 (BOM 2024.10) da `HapticFeedbackType` faqat `TextHandleMove`
 * (yengil tik) va `LongPress` (kuchli) beradi — shularni ishlatamiz.
 */

/** Bosilganda titrash + scale 0.96. Card, Row, custom bosiladigan elementlar uchun. */
fun Modifier.pressClickable(
    enabled: Boolean = true,
    haptic: HapticFeedbackType = HapticFeedbackType.TextHandleMove,
    pressedScale: Float = 0.96f,
    onClick: () -> Unit,
): Modifier = composed {
    val interaction = remember { MutableInteractionSource() }
    val pressed by interaction.collectIsPressedAsState()
    val scale by animateFloatAsState(
        targetValue = if (pressed) pressedScale else 1f,
        animationSpec = spring(),
        label = "press-scale",
    )
    val hf = LocalHapticFeedback.current
    this
        .scale(scale)
        .clickable(
            interactionSource = interaction,
            indication = LocalIndication.current,
            enabled = enabled,
        ) {
            hf.performHapticFeedback(haptic)
            onClick()
        }
}

/**
 * Material3 `Button`/`FAB` uchun: ular o'z bosilishini boshqaradi, faqat `onClick`
 * ga titrash qo'shamiz. `val click = rememberHapticClick { ... }` → `onClick = click`.
 */
@Composable
fun rememberHapticClick(
    haptic: HapticFeedbackType = HapticFeedbackType.TextHandleMove,
    onClick: () -> Unit,
): () -> Unit {
    val hf: HapticFeedback = LocalHapticFeedback.current
    return {
        hf.performHapticFeedback(haptic)
        onClick()
    }
}
