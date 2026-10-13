package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import ru.khudob1n.krasnodar.parking.data.ParkingSession
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.clockOf
import ru.khudob1n.krasnodar.parking.data.dayOf
import ru.khudob1n.krasnodar.parking.data.formatPlate
import ru.khudob1n.krasnodar.parking.data.formatRub
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** «История»: последние парковки - где, какая машина, когда и сколько стоило. */
@Composable
fun HistorySection(sessions: SessionRepository) {
    val c = AppTheme.colors
    val list by sessions.sessions.collectAsState()
    LaunchedEffect(Unit) {
        try {
            sessions.refresh()
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            // Нет связи - покажем то, что уже есть.
        }
    }

    ProfileSection("История парковок") {
        if (list.isEmpty()) {
            Text("Парковок пока не было", style = AppTheme.type.body, color = c.textSecondary, modifier = Modifier.padding(16.dp))
        }
        list.forEachIndexed { i, s ->
            if (i > 0) RowDivider()
            ProfileRow(trailing = {
                Text(
                    when {
                        s.isActive -> "идёт"
                        s.debt > 0 -> "долг ${formatRub(s.debt)}"
                        else -> formatRub(s.cost)
                    },
                    style = AppTheme.type.button,
                    color = when {
                        s.isActive -> c.parkingPaid
                        s.debt > 0 -> c.warningText
                        else -> c.textPrimary
                    },
                )
            }) {
                Text("Зона ${s.zone} · ${formatPlate(s.plate)}", style = AppTheme.type.body, color = c.textPrimary)
                Text(s.periodLabel(), style = AppTheme.type.small, color = c.textSecondary)
            }
        }
    }
}

/** «13 окт, 18:00–19:30»; у идущей - «13 окт, с 18:00». */
internal fun ParkingSession.periodLabel(): String {
    val day = dayOf(startedAt)
    val end = endedAt
    return if (end == null) "$day, с ${clockOf(startedAt)}" else "$day, ${clockOf(startedAt)}–${clockOf(end)}"
}
