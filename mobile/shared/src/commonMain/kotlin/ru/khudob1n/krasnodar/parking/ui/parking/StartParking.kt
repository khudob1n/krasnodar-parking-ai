package ru.khudob1n.krasnodar.parking.ui.parking

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.ApiException
import ru.khudob1n.krasnodar.parking.data.AuthRepository
import ru.khudob1n.krasnodar.parking.data.ParkingSession
import ru.khudob1n.krasnodar.parking.data.Quote
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.Zone
import ru.khudob1n.krasnodar.parking.data.formatPlate
import ru.khudob1n.krasnodar.parking.data.formatRub
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Всё, что нужно карточке зоны, чтобы начать парковку, и куда вести, если чего-то не хватает. */
class ParkingActions(
    val auth: AuthRepository,
    val vehicles: VehicleRepository,
    val sessions: SessionRepository,
    /** Открыть профиль: войти или добавить машину. */
    val openAccount: () -> Unit,
    val onStarted: (ParkingSession) -> Unit,
)

/**
 * Кнопка «Начать парковку» под ценой. Не вошёл - ведёт ко входу, нет машин - к добавлению машины;
 * машин несколько - сначала выбрать, какую ставим.
 */
@Composable
fun StartParking(zone: Zone, minutes: Int, quote: Quote?, actions: ParkingActions) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    val session by actions.auth.session.collectAsState()
    val vehicles by actions.vehicles.vehicles.collectAsState()
    var chosen by remember { mutableStateOf<Long?>(null) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    when {
        session == null -> PrimaryButton("Войти, чтобы оплатить", onClick = actions.openAccount)
        vehicles.isEmpty() -> {
            Text("Добавьте машину: парковку оплачивают за конкретный госномер.", style = AppTheme.type.small, color = c.textSecondary)
            PrimaryButton("Добавить машину", onClick = actions.openAccount)
        }
        else -> Column {
            val vehicle = vehicles.firstOrNull { it.id == chosen } ?: vehicles.first()
            if (vehicles.size > 1) {
                Row(Modifier.horizontalScroll(rememberScrollState()).padding(bottom = 8.dp), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    vehicles.forEach { v ->
                        val selected = v.id == vehicle.id
                        Box(
                            Modifier.clip(AppTheme.shapes.pill).background(if (selected) c.textPrimary else c.backgroundSecondary)
                                .clickable(role = Role.Button) { chosen = v.id }.padding(horizontal = 12.dp, vertical = 8.dp),
                        ) {
                            Text(formatPlate(v.plate), style = AppTheme.type.button, color = if (selected) c.backgroundPrimary else c.textPrimary)
                        }
                    }
                }
            }
            val price = quote?.let { " · ${formatRub(it.total)}" }.orEmpty()
            PrimaryButton("Поставить ${formatPlate(vehicle.plate)}$price", enabled = !busy) {
                scope.launch {
                    busy = true
                    error = null
                    try {
                        actions.onStarted(actions.sessions.start(zone.number, vehicle.id, minutes))
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: ApiException) {
                        error = when (e.status) {
                            409 -> "Эта машина уже на парковке. Продлите ту парковку или остановите её."
                            else -> e.message
                        }
                    } catch (e: Exception) {
                        error = e.message ?: "Нет связи с сервером"
                    } finally {
                        busy = false
                    }
                }
            }
        }
    }
    error?.let { Text(it, style = AppTheme.type.small, color = c.warningText) }
}
