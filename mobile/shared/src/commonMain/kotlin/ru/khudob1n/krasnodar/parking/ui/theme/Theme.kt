package ru.khudob1n.krasnodar.parking.ui.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.ReadOnlyComposable
import androidx.compose.ui.unit.dp

/** Тема приложения: light / dark / как в системе - как переключатель темы на сайте. */
enum class ThemePreference { Light, Dark, System }

@Composable
fun ParkingTheme(preference: ThemePreference = ThemePreference.System, content: @Composable () -> Unit) {
    val dark = when (preference) {
        ThemePreference.Light -> false
        ThemePreference.Dark -> true
        ThemePreference.System -> isSystemInDarkTheme()
    }
    val colors = if (dark) DarkColors else LightColors
    // Material-компоненты (поля ввода, диалоги) - в тех же цветах, что и свои компоненты.
    val scheme = (if (dark) darkColorScheme() else lightColorScheme()).copy(
        primary = colors.textPrimary,
        onPrimary = colors.backgroundPrimary,
        background = colors.backgroundPrimary,
        onBackground = colors.textPrimary,
        surface = colors.backgroundPrimary,
        onSurface = colors.textPrimary,
        surfaceVariant = colors.backgroundSecondary,
        onSurfaceVariant = colors.textSecondary,
    )
    CompositionLocalProvider(LocalAppColors provides colors, LocalAppTypography provides AppTypography(onestFamily(), jetBrainsMonoFamily())) {
        MaterialTheme(colorScheme = scheme, content = content)
    }
}

/** Доступ к токенам: AppTheme.colors.bus, AppTheme.type.cardTitle, AppTheme.shapes.mapButton. */
object AppTheme {
    val colors: AppColors @Composable @ReadOnlyComposable get() = LocalAppColors.current
    val type: AppTypography @Composable @ReadOnlyComposable get() = LocalAppTypography.current
    val shapes = AppShapes
}

/**
 * Скругления сайта. На сайте углы - «сквирлы» (--smooth-corners: 0.8); обычное скругление того же
 * радиуса визуально почти не отличается на таких размерах.
 */
object AppShapes {
    val mapButton = SquircleShape(16.dp)
    val sidepage = RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)
    val card = RoundedCornerShape(24.dp)
    val pill = RoundedCornerShape(11.dp)
    val routeBadge = RoundedCornerShape(10.dp)
    val search = SquircleShape(16.dp)
}
