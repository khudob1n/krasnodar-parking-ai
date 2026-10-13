package ru.khudob1n.krasnodar.parking

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import ru.khudob1n.krasnodar.parking.data.ApiClient
import ru.khudob1n.krasnodar.parking.platform.AppConfig
import ru.khudob1n.krasnodar.parking.platform.createHttpClient
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme
import ru.khudob1n.krasnodar.parking.ui.theme.ParkingTheme

/** Корень интерфейса, общий для Android и iOS. Пока - экран-заглушка с проверкой сервера. */
@Composable
fun App(onDarkChange: (Boolean) -> Unit = {}) {
    ParkingTheme {
        val colors = AppTheme.colors
        LaunchedEffect(colors.isDark) { onDarkChange(colors.isDark) }
        val api = remember { ApiClient(createHttpClient(), AppConfig.apiBaseUrl) }
        // null - ещё проверяем, true/false - ответ сервера.
        var alive by remember { mutableStateOf<Boolean?>(null) }
        LaunchedEffect(Unit) { alive = api.isAlive() }

        Box(Modifier.fillMaxSize().background(colors.backgroundPrimary).safeDrawingPadding().padding(16.dp)) {
            Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                Text("Парковки Краснодара", style = AppTheme.type.h3, color = colors.textPrimary)
                Text(
                    "Оплата по минутам, напоминание до конца парковки и все зоны города на карте.",
                    style = AppTheme.type.body,
                    color = colors.textSecondary,
                )
            }
            Row(
                Modifier.align(Alignment.BottomStart).fillMaxWidth()
                    .background(colors.backgroundSecondary, AppTheme.shapes.card).padding(16.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                val (dot, label) = when (alive) {
                    null -> colors.functional to "Проверяю сервер…"
                    true -> Color(0xFF00B400) to "Сервер на связи"
                    false -> colors.logoCircle to "Сервер недоступен: ${AppConfig.apiBaseUrl}"
                }
                Box(Modifier.size(10.dp).background(dot, CircleShape))
                Spacer(Modifier.size(10.dp))
                Text(label, style = AppTheme.type.small, color = colors.textSecondary)
            }
        }
    }
}
