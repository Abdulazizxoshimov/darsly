package uz.darsly.mentor.ui.common

import androidx.compose.animation.core.LinearEasing
import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Shape
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

/**
 * Skelet-yuklanish (shimmer) — spinner o'rniga.
 *
 * Auditda: hamma joyda `CircularProgressIndicator`. Sekin internetda (loyihaning
 * asosiy auditoriyasi) foydalanuvchi bo'sh ekran + aylanuvchi doira ko'rib
 * "muzlab qoldimi?" deb o'ylaydi. Skelet kontent SHAKLINI oldindan ko'rsatadi —
 * kutish sub'ektiv qisqaradi. Bu estetika emas, past-tarmoq uchun funksional.
 */

/** Bitta shimmerlanuvchi blok. Kenglik/balandlik chaqiruvchidan (Modifier bilan). */
@Composable
fun SkeletonBox(
    modifier: Modifier = Modifier,
    shape: Shape = RoundedCornerShape(8.dp),
) {
    val transition = rememberInfiniteTransition(label = "skeleton")
    val progress by transition.animateFloat(
        initialValue = 0f,
        targetValue = 1f,
        animationSpec = infiniteRepeatable(
            animation = tween(durationMillis = 1300, easing = LinearEasing),
            repeatMode = RepeatMode.Restart,
        ),
        label = "skeleton-sweep",
    )
    val base = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.55f)
    val highlight = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.95f)
    Spacer(
        modifier
            .clip(shape)
            .drawBehind {
                val w = size.width
                val sweep = w * 1.5f
                // Yorug' chiziq chapdan o'ngga suriladi.
                val startX = -sweep + progress * (w + sweep)
                drawRect(
                    Brush.linearGradient(
                        colors = listOf(base, highlight, base),
                        start = Offset(startX, 0f),
                        end = Offset(startX + sweep, 0f),
                    )
                )
            },
    )
}

/** Bitta matn qatori ko'rinishidagi skelet chizig'i. */
@Composable
fun SkeletonLine(width: Dp, height: Dp = 12.dp) {
    SkeletonBox(Modifier.width(width).height(height), RoundedCornerShape(6.dp))
}

/**
 * Darslar ro'yxati skeleti — `LessonCard` shaklini taqlid qiladi (sarlavha +
 * nishon qatori + tugma). LessonsScreen'dagi spinner o'rniga.
 */
@Composable
fun LessonListSkeleton() {
    LazyColumn(
        Modifier.fillMaxWidth(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        items(count = 5) { SkeletonCard() }
    }
}

/**
 * Oddiy ro'yxat skeleti — bildirishnoma/arxiv kabi (ikonka doirasi + ikki qator).
 */
@Composable
fun ListRowSkeleton(rows: Int = 6) {
    LazyColumn(
        Modifier.fillMaxWidth(),
        contentPadding = PaddingValues(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        items(count = rows) {
            Row(
                Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                SkeletonBox(Modifier.size(40.dp), RoundedCornerShape(percent = 50))
                Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    SkeletonLine(width = 220.dp, height = 14.dp)
                    SkeletonLine(width = 140.dp, height = 12.dp)
                }
            }
        }
    }
}

@Composable
private fun SkeletonCard() {
    Column(
        Modifier
            .fillMaxWidth()
            .clip(RoundedCornerShape(18.dp))
            .background(MaterialTheme.colorScheme.surface)
            .padding(16.dp),
    ) {
        SkeletonLine(width = 200.dp, height = 16.dp)
        Spacer(Modifier.height(12.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            SkeletonBox(Modifier.width(64.dp).height(22.dp), RoundedCornerShape(percent = 50))
            SkeletonLine(width = 90.dp, height = 14.dp)
        }
        Spacer(Modifier.height(14.dp))
        SkeletonBox(Modifier.fillMaxWidth().height(40.dp), RoundedCornerShape(10.dp))
    }
}
