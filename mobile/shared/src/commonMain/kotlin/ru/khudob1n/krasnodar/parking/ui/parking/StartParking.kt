package ru.khudob1n.krasnodar.parking.ui.parking

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.ApiException
import ru.khudob1n.krasnodar.parking.data.AuthRepository
import ru.khudob1n.krasnodar.parking.data.CardRepository
import ru.khudob1n.krasnodar.parking.data.PaymentMode
import ru.khudob1n.krasnodar.parking.data.label
import ru.khudob1n.krasnodar.parking.ui.account.translateCardError
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
    val cards: CardRepository,
    val sessions: SessionRepository,
    /** Открыть профиль: войти, добавить машину или карту. */
    val openAccount: () -> Unit,
    val onStarted: (ParkingSession) -> Unit,
)

/**
 * Кнопка «Поставить» под ценой. Не вошёл - ведёт ко входу, нет машины или карты - в профиль;
 * машин или карт несколько - сначала выбрать. С предоплатой сразу списывается [quote], с
 * постоплатой - ничего до остановки.
 */
@Composable
fun StartParking(zone: Zone, mode: PaymentMode, minutes: Int, quote: Quote?, actions: ParkingActions) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    val session by actions.auth.session.collectAsState()
    val vehicles by actions.vehicles.vehicles.collectAsState()
    val cards by actions.cards.cards.collectAsState()
    var chosenVehicle by remember { mutableStateOf<Long?>(null) }
    var chosenCard by remember { mutableStateOf<Long?>(null) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    when {
        session == null -> PrimaryButton("Войти, чтобы оплатить", onClick = actions.openAccount)
        vehicles.isEmpty() -> {
            Text("Добавьте машину: парковку оплачивают за конкретный госномер.", style = AppTheme.type.small, color = c.textSecondary)
            PrimaryButton("Добавить машину", onClick = actions.openAccount)
        }
        cards.isEmpty() -> {
            Text("Привяжите карту: с неё спишется оплата.", style = AppTheme.type.small, color = c.textSecondary)
            PrimaryButton("Привязать карту", onClick = actions.openAccount)
        }
        else -> Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            val vehicle = vehicles.firstOrNull { it.id == chosenVehicle } ?: vehicles.first()
            // По умолчанию - последняя привязанная карта, как и на сервере.
            val card = cards.firstOrNull { it.id == chosenCard } ?: cards.last()
            if (vehicles.size > 1) ChoiceRow(vehicles, vehicle, { formatPlate(it.plate) }) { chosenVehicle = it.id }
            if (cards.size > 1) ChoiceRow(cards, card, { it.label() }) { chosenCard = it.id }
            else Text("Оплата: ${card.label()}", style = AppTheme.type.small, color = c.textSecondary)
            val price = if (mode == PaymentMode.Prepaid) quote?.let { " · ${formatRub(it.total)}" }.orEmpty() else ""
            PrimaryButton("Поставить ${formatPlate(vehicle.plate)}$price", enabled = !busy) {
                scope.launch {
                    busy = true
                    error = null
                    try {
                        actions.onStarted(actions.sessions.start(zone.number, vehicle.id, minutes, card.id, mode))
                    } catch (e: CancellationException) {
                        throw e
                    } catch (e: ApiException) {
                        error = when {
                            e.status == 409 -> "Эта машина уже на парковке. Продлите ту парковку или остановите её."
                            e.message == "pay for the previous parking first" -> "Сначала оплатите прошлую парковку: она в профиле, в истории."
                            e.status == 402 -> translateCardError(e.message ?: "")
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
