package ru.khudob1n.krasnodar.parking.ui.theme

import androidx.compose.runtime.Immutable
import androidx.compose.runtime.staticCompositionLocalOf
import androidx.compose.ui.graphics.Color

/**
 * Цвета сайта один в один - passenger/client/styles/colors.css. Имена - как CSS-переменные
 * (--background-primary -> backgroundPrimary), чтобы сверяться с сайтом без таблицы соответствий.
 */
@Immutable
data class AppColors(
    val backgroundPrimary: Color,
    val textPrimary: Color,
    val backgroundSecondary: Color,
    val textSecondary: Color,
    val logoCircle: Color,
    val logoArrow: Color,
    val functional: Color,
    val functionalTram: Color,
    val walk: Color,
    val walkIcon: Color,
    val mapBg: Color,
    /** Текст предупреждений: жёлтый значков на белом не читается (2,4:1), в светлой теме темнее. */
    val warningText: Color,
    val isDark: Boolean,
) {
    // Цвета видов транспорта и объектов - одни на обе темы (как на сайте).
    val bus = Color(0xFF00B400)
    val busTranslucent = Color(0xFF6DC76D)
    val tram = Color(0xFFFF640F)
    val tramTranslucent = Color(0xFFED9F74)
    val troll = Color(0xFF00B4FF)
    val trollTranslucent = Color(0xFF80D9FF)
    val train = Color(0xFF9747FF)
    val busTerminal = Color(0xFFE5294A)
    val airport = Color(0xFF2673C9)
    val favorite = Color(0xFFFF640F)
    val warning = Color(0xFFE09B00)

    // Парковки: синий знака «Парковка» - платные, зелёный - бесплатные.
    val parkingPaid = Color(0xFF2673C9)
    val parkingFree = Color(0xFF00B400)
}

val LightColors = AppColors(
    backgroundPrimary = Color.White,
    textPrimary = Color.Black,
    backgroundSecondary = Color(0xFFF2F2F2),
    textSecondary = Color(0xFF1E2841),
    logoCircle = Color(0xFFB10000),
    logoArrow = Color.White,
    functional = Color(0xFF55647D),
    functionalTram = Color(0xFF9BAAC3),
    walk = Color(0xFF3A4150),
    walkIcon = Color(0xFF3A4150),
    mapBg = Color(0xFFF7F7F5),
    warningText = Color(0xFF8F5F00),
    isDark = false,
)

val DarkColors = AppColors(
    backgroundPrimary = Color(0xFF16181C),
    textPrimary = Color(0xFFF0F1F4),
    backgroundSecondary = Color(0xFF1F2228),
    textSecondary = Color(0xFFC7CDDC),
    logoCircle = Color.White,
    logoArrow = Color(0xFFB10000),
    functional = Color(0xFF97A3BA),
    functionalTram = Color(0xFF6C7690),
    walk = Color(0xFF4A5263),
    walkIcon = Color(0xFFB4BBC8),
    mapBg = Color(0xFF010508),
    warningText = Color(0xFFE09B00),
    isDark = true,
)

val LocalAppColors = staticCompositionLocalOf { LightColors }
