package uz.darsly.mentor.ui.theme

import androidx.compose.foundation.border
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Neon nur (B «Jonli efir» dizayn tili).
 *
 * QOIDA: glow — kamyob imtiyoz. Faqat uch joyda ishlatiladi: jonli dars
 * kartasi, LIVE indikator va asosiy harakat tugmasi. Har joyga qo'yilsa
 * «arzon o'yin» ko'rinishiga aylanadi — shuning uchun umumiy modifier
 * bitta joyda turibdi va chaqiruvchilar sanaladigan bo'lib qoladi.
 *
 * Texnik eslatma: rangli soya (`ambientColor`/`spotColor`) Android 9 (API 28)
 * dan boshlab ishlaydi; minSdk 26–27 da soya rangsiz tushadi — chegara esa
 * baribir ko'rinadi, ya'ni ma'no yo'qolmaydi (graceful degradation).
 */
fun Modifier.neonGlow(
    color: Color,
    shape: Shape,
    elevation: Dp = 12.dp,
    borderColor: Color? = null,
): Modifier {
    val glow = this.shadow(
        elevation = elevation,
        shape = shape,
        clip = false,
        ambientColor = color,
        spotColor = color,
    )
    return if (borderColor != null) glow.border(1.dp, borderColor, shape) else glow
}

/** Jonli dars kartasi uchun tayyor kombinatsiya: mint nur + mint chegara. */
@Composable
fun Modifier.liveCardGlow(shape: Shape = MaterialTheme.shapes.large): Modifier =
    neonGlow(
        color = DarslyTheme.colors.neon,
        shape = shape,
        elevation = 14.dp,
        borderColor = DarslyTheme.colors.neonBorder,
    )
