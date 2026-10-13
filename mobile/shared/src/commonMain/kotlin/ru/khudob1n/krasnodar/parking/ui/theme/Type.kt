package ru.khudob1n.krasnodar.parking.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.text.TextStyle
import org.jetbrains.compose.resources.Font
import org.jetbrains.compose.resources.FontResource
import androidx.compose.runtime.Composable
import ru.khudob1n.krasnodar.parking.resources.Res
import ru.khudob1n.krasnodar.parking.resources.jetbrains_mono
import ru.khudob1n.krasnodar.parking.resources.onest
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontVariation
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp

// Onest - шрифт интерфейса сайта, JetBrains Mono - подписи на карте и номера маршрутов.
// Оба вариативные (ось wght), поэтому один файл на все начертания.
@Composable
private fun variable(res: FontResource, vararg weights: FontWeight) = FontFamily(
    weights.map { Font(res, it, variationSettings = FontVariation.Settings(FontVariation.weight(it.weight))) },
)

@Composable
fun onestFamily() = variable(Res.font.onest, FontWeight.Normal, FontWeight.Medium, FontWeight.SemiBold, FontWeight.Bold)

@Composable
fun jetBrainsMonoFamily() = variable(Res.font.jetbrains_mono, FontWeight.Normal, FontWeight.Medium, FontWeight.Bold)

/** Размеры - мобильная вёрстка сайта (components/UI/Typography, карточки карты). */
@Immutable
data class AppTypography(val onest: FontFamily = FontFamily.Default, val jetBrainsMono: FontFamily = FontFamily.Monospace) {
    val h2: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.Medium, fontSize = 39.sp, lineHeight = 1.08.em)
    val h3: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.Medium, fontSize = 32.sp, lineHeight = 1.11.em)
    val h4: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.Medium, fontSize = 26.sp, lineHeight = 1.12.em)
    val cardTitle: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.SemiBold, fontSize = 24.sp, lineHeight = 1.15.em)
    val caption: TextStyle = TextStyle(fontFamily = onest, fontSize = 18.sp, lineHeight = 1.33.em)
    val body: TextStyle = TextStyle(fontFamily = onest, fontSize = 16.sp, lineHeight = 1.35.em)
    val button: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.Medium, fontSize = 15.sp, lineHeight = 20.sp)
    val small: TextStyle = TextStyle(fontFamily = onest, fontSize = 13.sp, lineHeight = 1.3.em)
    val routeNumber: TextStyle = TextStyle(fontFamily = onest, fontWeight = FontWeight.SemiBold, fontSize = 18.sp, lineHeight = 22.sp)
    val mono: TextStyle = TextStyle(fontFamily = jetBrainsMono, fontSize = 14.sp)
}

val LocalAppTypography = staticCompositionLocalOf { AppTypography() }

/** Размер текста, не зависящий от настройки шрифта: для букв и цифр внутри кружков фиксированного размера. */
@androidx.compose.runtime.Composable
fun fixedSp(dp: Float) = with(androidx.compose.ui.platform.LocalDensity.current) { androidx.compose.ui.unit.Dp(dp).toSp() }
