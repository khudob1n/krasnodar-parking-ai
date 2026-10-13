package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.CardRepository
import ru.khudob1n.krasnodar.parking.data.TEST_CARDS
import ru.khudob1n.krasnodar.parking.data.expiry
import ru.khudob1n.krasnodar.parking.data.formatCardNumber
import ru.khudob1n.krasnodar.parking.data.label
import ru.khudob1n.krasnodar.parking.data.normalizeCardNumber
import ru.khudob1n.krasnodar.parking.data.parseExpiry
import ru.khudob1n.krasnodar.parking.ui.components.InputField
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.components.TextButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/**
 * «Карты оплаты»: привязанные карты и форма новой. Платёжный провайдер пока тестовый: настоящие
 * деньги не списываются, а подсказка под формой показывает номера, которые проверяют разные исходы.
 */
@Composable
fun CardsSection(cards: CardRepository) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    val list by cards.cards.collectAsState()
    var adding by remember { mutableStateOf(false) }
    var numberInput by remember { mutableStateOf("") }
    var expiryInput by remember { mutableStateOf("") }
    var cvc by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    suspend fun guarded(action: suspend () -> Unit) {
        busy = true
        error = null
        try {
            action()
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            error = humanError(e) { status ->
                when (status) {
                    402 -> "Банк отклонил карту. Попробуйте другую"
                    409 -> "Эта карта уже привязана"
                    else -> null
                }
            }.let(::translateCardError)
        } finally {
            busy = false
        }
    }

    LaunchedEffect(Unit) { guarded { cards.refresh() } }

    ProfileSection("Карты оплаты") {
        list.forEachIndexed { i, card ->
            if (i > 0) RowDivider()
            ProfileRow(trailing = { RowAction("Удалить", enabled = !busy) { scope.launch { guarded { cards.delete(card.id) } } } }) {
                Text(card.label(), style = AppTheme.type.body, color = c.textPrimary)
                Text("до ${card.expiry()}", style = AppTheme.type.small, color = c.textSecondary)
            }
        }
        if (list.isNotEmpty()) RowDivider()
        if (!adding) {
            AddRow(if (list.isEmpty()) "Привязать карту" else "Привязать ещё карту") { adding = true }
        } else {
            val number = normalizeCardNumber(numberInput)
            val expiry = parseExpiry(expiryInput)
            val ready = number != null && expiry != null && cvc.length == 3 && !busy
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                InputField(
                    value = numberInput,
                    onValueChange = { numberInput = formatCardNumber(it) },
                    placeholder = "Номер карты",
                    keyboardType = KeyboardType.Number,
                    imeAction = ImeAction.Next,
                )
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    InputField(
                        value = expiryInput,
                        onValueChange = { v -> expiryInput = v.filter { it.isDigit() }.take(4).let { if (it.length > 2) it.take(2) + "/" + it.drop(2) else it } },
                        placeholder = "ММ/ГГ",
                        keyboardType = KeyboardType.Number,
                        imeAction = ImeAction.Next,
                        modifier = Modifier.weight(1f),
                    )
                    InputField(
                        value = cvc,
                        onValueChange = { v -> cvc = v.filter { it.isDigit() }.take(3) },
                        placeholder = "CVC",
                        keyboardType = KeyboardType.NumberPassword,
                        modifier = Modifier.weight(1f),
                    )
                }
                PrimaryButton("Привязать", enabled = ready) {
                    if (number != null && expiry != null) scope.launch {
                        guarded {
                            cards.add(number, expiry.first, expiry.second, cvc)
                            numberInput = ""; expiryInput = ""; cvc = ""
                            adding = false
                        }
                    }
                }
                TextButton("Отмена") { adding = false; error = null }
                Text("Оплата тестовая, деньги не списываются. Нажмите на номер, чтобы подставить:", style = AppTheme.type.small, color = c.textSecondary)
                TEST_CARDS.forEach { (testNumber, outcome) ->
                    Text(
                        "$testNumber - $outcome", style = AppTheme.type.mono, color = c.textSecondary,
                        modifier = Modifier.clickable { numberInput = testNumber; if (expiryInput.isEmpty()) expiryInput = "12/30"; if (cvc.isEmpty()) cvc = "123" },
                    )
                }
            }
        }
        error?.let { Text(it, style = AppTheme.type.small, color = c.warningText, modifier = Modifier.padding(start = 16.dp, end = 16.dp, bottom = 12.dp)) }
    }
}

/** Ответы провайдера на английском - по-русски. */
internal fun translateCardError(message: String): String = when (message) {
    "card number is invalid" -> "Неверный номер карты"
    "card is expired" -> "Срок действия карты истёк"
    "cvc must be 3 digits" -> "CVC - три цифры с обратной стороны карты"
    "expiry date is invalid" -> "Срок действия - месяц и год, например 12/30"
    "card already added" -> "Эта карта уже привязана"
    "add a payment card first" -> "Карта отвязана. Привяжите карту в профиле"
    "unknown card token", "card declined" -> "Банк отклонил карту. Попробуйте другую"
    "insufficient funds" -> "На карте не хватает денег"
    else -> message
}
