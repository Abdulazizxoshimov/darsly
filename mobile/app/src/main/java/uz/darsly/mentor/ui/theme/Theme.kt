package uz.darsly.mentor.ui.theme

import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Shapes
import androidx.compose.material3.Typography
import androidx.compose.material3.darkColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

/**
 * Darsly brend mavzusi — **web dizayn tizimidan** ko'chirilgan.
 *
 * ## Nega bu fayl qayta yozildi
 * Avval `darkColorScheme()` ga faqat **4 ta** rol berilgan edi (`primary`, `secondary`,
 * `background`, `surface`). Material qolgan ~26 rolni o'zining **standart binafsha**
 * palitrasidan to'ldiradi — natijada bitta ekranda uchta turli accent ko'rinardi:
 * bizning ko'k tugmalar, Google'ning binafsha FAB'i va web'ning boshqa binafshasi.
 * Endi barcha rollar aniq belgilangan.
 *
 * ## Manba (taxmin emas)
 * `frontend/src/styles.css` → `:root` tokenlari. Har bir qiymat shu fayldan olingan,
 * shuning uchun ustoz web'da va telefonda **bir xil mahsulotni** ko'radi.
 *
 * ## Nega faqat qorong'i mavzu
 * Web dizayni qorong'i (`--bg: #0a0e14`) va yorug' varianti yo'q. Mobilda tizim
 * mavzusiga qarab yorug'ga o'tish web bilan mutlaqo boshqa ko'rinish berardi.
 * Zoom ham dars xonasini har doim qorong'i ko'rsatadi — video kontrast uchun.
 */

// ─── Web tokenlari (styles.css :root) ────────────────────────────────────────
private val Bg = Color(0xFF0A0E14)
private val Card = Color(0xFF12171F)
private val Elevated = Color(0xFF161C26)
private val Hover = Color(0xFF1B222C)

private val Border = Color(0xFF232B37)
private val BorderStrong = Color(0xFF262F3B)

private val TextPrimary = Color(0xFFEEF1F6)
private val Text2 = Color(0xFF97A3B3)
private val Text3 = Color(0xFF6B7688)

private val Accent = Color(0xFF6C5CE7)
private val AccentHover = Color(0xFF7D6FF2)
private val AccentLight = Color(0xFF8B7CF6)

private val Danger = Color(0xFFEF4444)
private val Success = Color(0xFF34D399)
private val Warning = Color(0xFFF59E0B)
private val Info = Color(0xFF3B82F6)

private val DarslyScheme = darkColorScheme(
    primary = Accent,
    onPrimary = Color.White,
    primaryContainer = AccentHover,
    onPrimaryContainer = Color.White,
    inversePrimary = AccentLight,

    secondary = AccentLight,
    onSecondary = Color.White,
    secondaryContainer = Elevated,
    onSecondaryContainer = TextPrimary,

    tertiary = Info,
    onTertiary = Color.White,
    tertiaryContainer = Elevated,
    onTertiaryContainer = TextPrimary,

    background = Bg,
    onBackground = TextPrimary,
    surface = Card,
    onSurface = TextPrimary,
    surfaceVariant = Elevated,
    onSurfaceVariant = Text2,
    surfaceTint = Accent,
    inverseSurface = TextPrimary,
    inverseOnSurface = Bg,

    // Material 3 "surface container" oilasi — web'ning qatlamlari bilan mos.
    surfaceContainerLowest = Bg,
    surfaceContainerLow = Card,
    surfaceContainer = Elevated,
    surfaceContainerHigh = Hover,
    surfaceContainerHighest = Hover,
    surfaceBright = Hover,
    surfaceDim = Bg,

    error = Danger,
    onError = Color.White,
    errorContainer = Color(0xFF2A1416),
    onErrorContainer = Color(0xFFFFB4AB),

    outline = BorderStrong,
    outlineVariant = Border,
    scrim = Color(0xE6000000),
)

/**
 * Web radiuslari: `--r-sm: 8` · `--r-md: 10` · `--r-lg: 14` · `--r-xl: 16` · `--r-2xl: 20`.
 * Material'ning `extraLarge` (28dp) juda dumaloq — web bilan mos emas.
 */
private val DarslyShapes = Shapes(
    extraSmall = RoundedCornerShape(8.dp),
    small = RoundedCornerShape(8.dp),
    medium = RoundedCornerShape(10.dp),
    large = RoundedCornerShape(14.dp),
    extraLarge = RoundedCornerShape(20.dp),
)

/**
 * Tipografiya: web'da sarlavhalar **800**, tugmalar **700** (`styles.css` `.h1`, `.btn`).
 * Material default'i ancha yengil — shuning uchun og'irliklar moslandi.
 */
private val DarslyTypography = Typography().run {
    copy(
        headlineMedium = headlineMedium.copy(fontWeight = FontWeight.ExtraBold, letterSpacing = (-0.5).sp),
        headlineSmall = headlineSmall.copy(fontWeight = FontWeight.ExtraBold, letterSpacing = (-0.3).sp),
        titleLarge = titleLarge.copy(fontWeight = FontWeight.Bold),
        titleMedium = titleMedium.copy(fontWeight = FontWeight.Bold),
        titleSmall = titleSmall.copy(fontWeight = FontWeight.Bold),
        labelLarge = labelLarge.copy(fontWeight = FontWeight.Bold),
        labelMedium = labelMedium.copy(fontWeight = FontWeight.SemiBold),
    )
}

/**
 * Material 3 da bo'lmagan, lekin web'da bor semantik ranglar.
 *
 * `success`/`warning`/`info` — M3 sxemasida yo'q (faqat `error` bor). Ularni shu yerdan
 * olamiz, aks holda ekranlarda qattiq yozilgan `Color(0xFF...)` paydo bo'lardi va
 * web bilan ajralib ketardi.
 */
data class DarslyColors(
    val success: Color = Success,
    val warning: Color = Warning,
    val info: Color = Info,
    /** Holat nishonlari uchun yumshoq fon — web'da 0.15 alfa (`.badge--live` va h.k.). */
    val liveSoft: Color = Danger.copy(alpha = 0.15f),
    val scheduledSoft: Color = Info.copy(alpha = 0.15f),
    val endedSoft: Color = Text3.copy(alpha = 0.15f),
    val cancelledSoft: Color = Warning.copy(alpha = 0.15f),
    val textMuted: Color = Text3,
)

val LocalDarslyColors = staticCompositionLocalOf { DarslyColors() }

/** Qisqartma: `DarslyTheme.colors.success`. */
object DarslyTheme {
    val colors: DarslyColors
        @Composable get() = LocalDarslyColors.current
}

@Composable
fun DarslyTheme(content: @Composable () -> Unit) {
    CompositionLocalProvider(LocalDarslyColors provides DarslyColors()) {
        MaterialTheme(
            colorScheme = DarslyScheme,
            shapes = DarslyShapes,
            typography = DarslyTypography,
            content = content,
        )
    }
}
