package ru.khudob1n.krasnodar.parking.map

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.rememberScrollState
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import ru.khudob1n.krasnodar.parking.data.ApiClient
import ru.khudob1n.krasnodar.parking.ui.parking.ParkingActions
import ru.khudob1n.krasnodar.parking.ui.parking.StartParking
import ru.khudob1n.krasnodar.parking.data.Quote
import ru.khudob1n.krasnodar.parking.data.Zone
import ru.khudob1n.krasnodar.parking.data.ZoneKind
import ru.khudob1n.krasnodar.parking.data.clockOf
import ru.khudob1n.krasnodar.parking.data.formatMinutes
import ru.khudob1n.krasnodar.parking.data.formatRub
import ru.khudob1n.krasnodar.parking.data.paidHoursLabel
import ru.khudob1n.krasnodar.parking.data.priceLabel
import ru.khudob1n.krasnodar.parking.data.PaymentMode
import ru.khudob1n.krasnodar.parking.ui.parking.ChoiceRow
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Быстрый выбор длительности - самые частые варианты. */
private val PRESETS = listOf(15, 30, 60, 120, 180)
private const val STEP = 15
private const val MAX_MINUTES = 24 * 60

/** Платная ли зона: у бесплатных и у зон без цены считать нечего. */
val Zone.isPaid: Boolean get() = kind != ZoneKind.Free && pricePerHour > 0

/**
 * Карточка зоны: номер с таблички, адрес, тариф и места, а у платной - выбор времени и цена с
 * сервера. Цена поминутная: в ней только минуты, попавшие в платные часы.
 */
@Composable
fun ZoneSheet(zone: Zone, api: ApiClient, parking: ParkingActions, onClose: () -> Unit) {
    val c = AppTheme.colors
    Column(
        Modifier.fillMaxWidth().widthIn(max = 560.dp).clip(AppTheme.shapes.card).background(c.backgroundPrimary).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
            Box(Modifier.size(12.dp).background(if (zone.isPaid) c.parkingPaid else c.parkingFree, CircleShape))
            Text("Зона ${zone.number}", style = AppTheme.type.cardTitle, color = c.textPrimary, modifier = Modifier.weight(1f))
            Box(
                Modifier.size(36.dp).clip(CircleShape).background(c.backgroundSecondary).clickable(role = Role.Button, onClickLabel = "Закрыть", onClick = onClose),
                contentAlignment = Alignment.Center,
            ) { Text("✕", style = AppTheme.type.button, color = c.textSecondary) }
        }
        Text(zone.address, style = AppTheme.type.body, color = c.textSecondary)
        if (zone.description.isNotBlank() && zone.description != zone.address) {
            Text(zone.description, style = AppTheme.type.small, color = c.functional)
        }
        val hours = zone.paidHoursLabel()
        Text(if (hours.isEmpty()) zone.priceLabel() else "${zone.priceLabel()} · $hours", style = AppTheme.type.body, color = c.textPrimary)
        val places = buildList {
            if (zone.totalPlaces > 0) add("мест ${zone.totalPlaces}")
            if (zone.disabledPlaces > 0) add("для инвалидов ${zone.disabledPlaces}")
            if (zone.inBuilding) add("в здании")
        }
        if (places.isNotEmpty()) Text(places.joinToString(", ").replaceFirstChar { it.uppercase() }, style = AppTheme.type.small, color = c.functional)

        if (zone.isPaid) {
            var mode by remember { mutableStateOf(PaymentMode.Postpaid) }
            var minutes by remember(zone.number) { mutableStateOf(60) }
            ChoiceRow(
                listOf(PaymentMode.Postpaid, PaymentMode.Prepaid), mode,
                { if (it == PaymentMode.Postpaid) "По факту" else "Заранее" },
                Modifier.padding(top = 4.dp),
            ) { mode = it }
            if (mode == PaymentMode.Postpaid) {
                Text(
                    "Ставите и уезжаете, когда нужно. Спишем за фактические минуты, когда остановите парковку. Не дольше суток.",
                    style = AppTheme.type.small, color = c.functional,
                )
                StartParking(zone, mode, minutes, null, parking)
            } else {
                val quote = PriceCalculator(zone, api, minutes) { minutes = it }
                StartParking(zone, mode, minutes, quote, parking)
            }
        }
    }
}

/** Выбор времени и цена с сервера; возвращает последний расчёт для этой зоны и времени. */
@Composable
private fun PriceCalculator(zone: Zone, api: ApiClient, minutes: Int, onMinutes: (Int) -> Unit): Quote? {
    val c = AppTheme.colors
    var quote by remember(zone.number) { mutableStateOf<Quote?>(null) }
    var quoteMinutes by remember(zone.number) { mutableStateOf(0) }
    var failed by remember(zone.number) { mutableStateOf(false) }

    // Цену считает сервер (тот же расчёт, что при оплате); пока человек листает варианты - ждём.
    LaunchedEffect(zone.number, minutes) {
        delay(250)
        try {
            quote = api.price(zone.number, minutes)
            quoteMinutes = minutes
            failed = false
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            failed = true
        }
    }

    Row(
        Modifier.fillMaxWidth().horizontalScroll(rememberScrollState()).padding(top = 4.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Chip("−", selected = false, enabled = minutes > STEP) { onMinutes((minutes - STEP).coerceAtLeast(STEP)) }
        PRESETS.forEach { preset -> Chip(formatMinutes(preset), selected = minutes == preset) { onMinutes(preset) } }
        Chip("+", selected = false, enabled = minutes < MAX_MINUTES) { onMinutes((minutes + STEP).coerceAtMost(MAX_MINUTES)) }
    }

    // Расчёт для прошлого выбора не показываем: цена должна совпадать с выбранным временем.
    val q = quote?.takeIf { it.zone == zone.number && quoteMinutes == minutes }
    Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
        Text(
            if (q != null) formatRub(q.total) else "…",
            style = AppTheme.type.h3, color = c.textPrimary,
        )
        Text(
            if (q != null) "за ${formatMinutes(minutes)}, до ${clockOf(q.end)}" else "за ${formatMinutes(minutes)}",
            style = AppTheme.type.body, color = c.textSecondary, modifier = Modifier.padding(bottom = 4.dp),
        )
    }
    when {
        failed -> Text("Не удалось посчитать цену: нет связи с сервером", style = AppTheme.type.small, color = c.warningText)
        q != null && q.paidMinutes < minutes -> Text(
            if (q.paidMinutes == 0) "Всё это время парковка бесплатная" else "Платных из них ${formatMinutes(q.paidMinutes)}, остальное бесплатно",
            style = AppTheme.type.small, color = c.functional,
        )
        q != null -> Text("Оплата поминутная: остановите парковку раньше - заплатите только за фактические минуты", style = AppTheme.type.small, color = c.functional)
    }
    return q
}

@Composable
private fun Chip(label: String, selected: Boolean, enabled: Boolean = true, onClick: () -> Unit) {
    val c = AppTheme.colors
    Box(
        Modifier.clip(AppTheme.shapes.pill)
            .background(if (selected) c.textPrimary else c.backgroundSecondary)
            .clickable(enabled = enabled, role = Role.Button, onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 9.dp),
    ) {
        Text(
            label, style = AppTheme.type.button,
            color = when {
                selected -> c.backgroundPrimary
                enabled -> c.textPrimary
                else -> c.functionalTram
            },
        )
    }
}
