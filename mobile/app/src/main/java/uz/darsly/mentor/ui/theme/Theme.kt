package uz.darsly.mentor.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

private val Brand = Color(0xFF1D4ED8)

private val DarkScheme = darkColorScheme(
    primary = Color(0xFF7FA6FF),
    secondary = Color(0xFF9CC0FF),
    background = Color(0xFF0B0F1A),
    surface = Color(0xFF141A28),
)

private val LightScheme = lightColorScheme(
    primary = Brand,
    secondary = Color(0xFF2563EB),
)

@Composable
fun DarslyTheme(content: @Composable () -> Unit) {
    MaterialTheme(
        colorScheme = if (isSystemInDarkTheme()) DarkScheme else LightScheme,
        content = content,
    )
}
