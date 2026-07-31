package uz.darsly.mentor.ui.theme

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.graphics.StrokeCap
import androidx.compose.ui.graphics.StrokeJoin
import androidx.compose.ui.graphics.drawscope.Stroke
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Jonly puls belgisi — brend logotipining chiziq varianti.
 *
 * SVG manba: `~/jonly-logolar/jonly-B-ikonka.svg` (M16 62 H36 L49 32 L66 88
 * L77 62 H104, viewBox 120). Rastr resurs emas, Canvas: istalgan o'lchamda
 * tiniq va rangi temadan keladi (masalan, sarlavhada mint, boshqa joyda oq).
 */
@Composable
fun PulseMark(
    size: Dp = 22.dp,
    color: Color = MaterialTheme.colorScheme.primary,
    modifier: Modifier = Modifier,
) {
    Canvas(modifier.size(size)) {
        val s = this.size.minDimension
        fun x(v: Float) = v / 120f * s
        val path = Path().apply {
            moveTo(x(14f), x(62f))
            lineTo(x(34f), x(62f))
            lineTo(x(48f), x(28f))
            lineTo(x(68f), x(92f))
            lineTo(x(80f), x(62f))
            lineTo(x(106f), x(62f))
        }
        drawPath(
            path = path,
            color = color,
            style = Stroke(
                width = x(13f),
                cap = StrokeCap.Round,
                join = StrokeJoin.Round,
            ),
        )
    }
}
