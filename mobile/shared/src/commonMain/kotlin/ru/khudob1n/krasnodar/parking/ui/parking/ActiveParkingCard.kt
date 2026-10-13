package ru.khudob1n.krasnodar.parking.ui.parking

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.rememberUpdatedState
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.ApiException
import ru.khudob1n.krasnodar.parking.data.ParkingSession
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.clockOf
import ru.khudob1n.krasnodar.parking.data.formatCountdown
import ru.khudob1n.krasnodar.parking.data.formatPlate
import ru.khudob1n.krasnodar.parking.data.formatRub
import ru.khudob1n.krasnodar.parking.ui.account.CloseButton
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.components.TextButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme
import kotlin.time.Clock
import kotlin.time.Duration

/** Варианты продления одним нажатием. */
private val EXTEND_OPTIONS = listOf(15, 30, 60)

/**
 * Идущая парковка: обратный отсчёт до конца оплаченного времени, продление и остановка.
 * Остановили - карточка показывает итог (только фактические минуты), пока её не закроют.
 */
@Composable
fun ActiveParkingCard(session: ParkingSession, sessions: SessionRepository, onClose: () -> Unit) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    var now by remember { mutableStateOf(Clock.System.now()) }
    var busy by remember { mutableStateOf(false) }
    var confirmStop by remember(session.id) { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    // Продление меняет plannedEnd той же сессии - цикл читает свежую версию, а не ту, с которой начал.
    val current by rememberUpdatedState(session)
    LaunchedEffect(session.id) {
        while (current.isActive) {
            now = Clock.System.now()
            // Время вышло - сервер уже считает парковку завершённой, берём итог оттуда.
            if (current.remaining(now) == Duration.ZERO) {
                runCatching { sessions.refresh() }
                break
            }
            delay(1000)
        }
    }

    fun act(action: suspend () -> Unit) {
        scope.launch {
            busy = true
            error = null
            try {
                action()
            } catch (e: CancellationException) {
                throw e
            } catch (e: ApiException) {
                error = when (e.status) {
                    409 -> "Парковка уже завершена"
                    400 -> "Парковка не может длиться больше суток"
                    else -> e.message
                }
            } catch (e: Exception) {
                error = e.message ?: "Нет связи с сервером"
            } finally {
                busy = false
            }
        }
    }

    Column(
        Modifier.fillMaxWidth().widthIn(max = 560.dp).clip(AppTheme.shapes.card).background(c.backgroundPrimary).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp),
    ) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Text(
                if (session.isActive) "Парковка идёт" else "Парковка завершена",
                style = AppTheme.type.cardTitle, color = c.textPrimary, modifier = Modifier.weight(1f),
            )
            if (!session.isActive) CloseButton(onClose)
        }
        Text("Зона ${session.zone} · ${formatPlate(session.plate)}", style = AppTheme.type.body, color = c.textSecondary)
        if (session.isActive) {
            Row(verticalAlignment = Alignment.Bottom, horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(formatCountdown(session.remaining(now)), style = AppTheme.type.h3.copy(fontFamily = AppTheme.type.jetBrainsMono), color = c.textPrimary)
                Text("до ${clockOf(session.plannedEnd)}", style = AppTheme.type.body, color = c.textSecondary, modifier = Modifier.padding(bottom = 4.dp))
            }
            Text("Если стоять до конца - ${formatRub(session.cost)}", style = AppTheme.type.small, color = c.functional)
            Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                EXTEND_OPTIONS.forEach { m ->
                    TextButton(if (m < 60) "+$m мин" else "+${m / 60} ч", enabled = !busy) { act { sessions.extend(session.id, m) } }
                }
            }
            if (confirmStop) {
                Text("Остановить сейчас? Спишутся только фактические минуты.", style = AppTheme.type.small, color = c.textSecondary)
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalAlignment = Alignment.CenterVertically) {
                    PrimaryButton("Остановить", enabled = !busy, modifier = Modifier.weight(1f)) { act { sessions.stop(session.id) } }
                    TextButton("Отмена", enabled = !busy) { confirmStop = false }
                }
            } else {
                PrimaryButton("Остановить парковку", enabled = !busy) { confirmStop = true }
            }
        } else {
            Text(
                "С ${clockOf(session.startedAt)} до ${clockOf(session.endedAt ?: session.plannedEnd)} · ${formatRub(session.cost)}",
                style = AppTheme.type.body, color = c.textPrimary,
            )
        }
        error?.let { Text(it, style = AppTheme.type.small, color = c.warningText) }
    }
}
