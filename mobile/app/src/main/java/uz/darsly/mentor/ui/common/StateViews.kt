package uz.darsly.mentor.ui.common

import androidx.annotation.DrawableRes
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.core.MutableTransitionState
import androidx.compose.animation.core.tween
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.painterResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import uz.darsly.mentor.R

/**
 * Bo'sh/xato holatlari — YAGONA manba.
 *
 * Auditda: har bo'sh holat sof matn + standart tugma edi ("arzon" ko'rinardi).
 * Endi har biri: line-art illyustratsiya + sarlavha + izoh + (ixtiyoriy) amal
 * tugmasi. Kirishda yumshoq fade+slide (keskin "sakrash" yo'q). Amal tugmasida
 * haptik. `res/drawable/il_*` illyustratsiyalari bilan.
 *
 * Ikkalasi ham LazyColumn ichida — pull-to-refresh bo'sh/xato ekranida ham ishlaydi.
 */
@Composable
fun EmptyState(
    @DrawableRes illustration: Int,
    title: String,
    message: String,
    modifier: Modifier = Modifier,
    actionLabel: String? = null,
    onAction: (() -> Unit)? = null,
) {
    StateScaffold(modifier) {
        Image(
            painter = painterResource(illustration),
            contentDescription = null,
            modifier = Modifier.size(120.dp),
        )
        Spacer(Modifier.height(20.dp))
        Text(title, style = MaterialTheme.typography.titleMedium, textAlign = TextAlign.Center)
        Spacer(Modifier.height(6.dp))
        Text(
            message,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
        if (actionLabel != null && onAction != null) {
            Spacer(Modifier.height(20.dp))
            val click = rememberHapticClick(onClick = onAction)
            Button(onClick = click) { Text(actionLabel) }
        }
    }
}

@Composable
fun ErrorState(
    message: String,
    onRetry: () -> Unit,
    modifier: Modifier = Modifier,
    @DrawableRes illustration: Int = R.drawable.il_error_generic,
) {
    StateScaffold(modifier) {
        Image(
            painter = painterResource(illustration),
            contentDescription = null,
            modifier = Modifier.size(120.dp),
        )
        Spacer(Modifier.height(20.dp))
        Text(
            "Nimadir noto'g'ri ketdi",
            style = MaterialTheme.typography.titleMedium,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(6.dp))
        Text(
            message,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
        Spacer(Modifier.height(20.dp))
        val click = rememberHapticClick(onClick = onRetry)
        OutlinedButton(onClick = click) { Text("Qayta urinish") }
    }
}

/** Markazlashtirilgan, scroll qilinadigan konteyner + kirish animatsiyasi. */
@Composable
private fun StateScaffold(
    modifier: Modifier,
    content: @Composable ColumnScope.() -> Unit,
) {
    // initialState=false → targetState=true bilan birinchi kompozitsiyada animatsiya bo'ladi.
    val visible = remember { MutableTransitionState(false).apply { targetState = true } }
    LazyColumn(modifier.fillMaxSize()) {
        item {
            AnimatedVisibility(
                visibleState = visible,
                enter = fadeIn(tween(400)) + slideInVertically(tween(400)) { it / 6 },
            ) {
                Column(
                    Modifier
                        .fillParentMaxSize()
                        .padding(32.dp),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                    content = content,
                )
            }
        }
    }
}
