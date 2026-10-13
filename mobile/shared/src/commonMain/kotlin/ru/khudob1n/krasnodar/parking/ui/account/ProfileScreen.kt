package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.ApiException
import ru.khudob1n.krasnodar.parking.data.AuthRepository
import ru.khudob1n.krasnodar.parking.data.CardRepository
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.CODE_DIGITS
import ru.khudob1n.krasnodar.parking.data.CodeResult
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.formatPhone
import ru.khudob1n.krasnodar.parking.data.normalizePhone
import ru.khudob1n.krasnodar.parking.ui.components.InputField
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.components.TextButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/**
 * Профиль на весь экран поверх карты. До входа - вход по коду, после - разделы как в настройках
 * телефона: номер, машины, карты оплаты, история парковок и выход.
 */
@Composable
fun ProfileScreen(
    auth: AuthRepository,
    vehicles: VehicleRepository,
    cards: CardRepository,
    sessions: SessionRepository,
    onClose: () -> Unit,
) {
    val c = AppTheme.colors
    val session by auth.session.collectAsState()
    val scope = rememberCoroutineScope()
    Column(Modifier.fillMaxSize().background(c.backgroundSecondary).safeDrawingPadding()) {
        Row(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 12.dp), verticalAlignment = Alignment.CenterVertically) {
            Text(if (session == null) "Вход" else "Профиль", style = AppTheme.type.cardTitle, color = c.textPrimary, modifier = Modifier.weight(1f))
            CloseButton(onClose)
        }
        Column(
            Modifier.fillMaxWidth().weight(1f).verticalScroll(rememberScrollState()).padding(horizontal = 16.dp).padding(bottom = 24.dp),
            verticalArrangement = Arrangement.spacedBy(20.dp),
        ) {
            val current = session
            if (current == null) {
                Column(
                    Modifier.fillMaxWidth().clip(AppTheme.shapes.card).background(c.backgroundPrimary).padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                ) { LoginFlow(auth) }
            } else {
                Column(
                    Modifier.fillMaxWidth().clip(AppTheme.shapes.card).background(c.backgroundPrimary).padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(2.dp),
                ) {
                    Text(formatPhone(current.user.phone), style = AppTheme.type.cardTitle, color = c.textPrimary)
                    Text("Номер для входа и напоминаний", style = AppTheme.type.small, color = c.textSecondary)
                }
                VehiclesSection(vehicles)
                CardsSection(cards)
                HistorySection(sessions)
                Column(Modifier.fillMaxWidth().clip(AppTheme.shapes.card).background(c.backgroundPrimary)) {
                    ProfileRow(onClick = { scope.launch { auth.logout(); vehicles.clear(); cards.clear(); sessions.clear() } }) {
                        Text("Выйти", style = AppTheme.type.button, color = c.busTerminal)
                    }
                }
            }
        }
    }
}

/**
 * Два шага: номер телефона, потом код - последние 4 цифры номера, с которого позвонили (звонок
 * сбрасывается сам, отвечать не нужно). Повторный звонок можно запросить через минуту - кнопка
 * сама показывает, сколько ждать.
 */
@Composable
private fun LoginFlow(auth: AuthRepository) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    var phoneInput by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf<String?>(null) } // номер, на который позвонили
    var code by remember { mutableStateOf("") }
    var wait by remember { mutableStateOf(0) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(wait > 0) {
        while (wait > 0) {
            delay(1000)
            wait--
        }
    }

    suspend fun send(target: String) {
        busy = true
        error = null
        try {
            when (val result = auth.requestCode(target)) {
                CodeResult.Sent -> { phone = target; wait = 60 }
                is CodeResult.TooSoon -> { phone = target; wait = result.seconds }
                is CodeResult.Failed -> error = result.message
            }
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            error = "Нет связи с сервером"
        } finally {
            busy = false
        }
    }

    suspend fun confirm(target: String, value: String) {
        busy = true
        error = null
        try {
            auth.verify(target, value)
        } catch (e: CancellationException) {
            throw e
        } catch (e: ApiException) {
            error = when (e.status) {
                401 -> "Неверный код"
                429 -> "Слишком много попыток. Запросите новый звонок"
                else -> e.message
            }
            code = ""
        } catch (_: Exception) {
            error = "Нет связи с сервером"
        } finally {
            busy = false
        }
    }

    val sentTo = phone
    if (sentTo == null) {
        val normalized = normalizePhone(phoneInput)
        Text("Номер нужен, чтобы оплачивать парковку и присылать напоминания.", style = AppTheme.type.small, color = c.textSecondary)
        InputField(
            value = phoneInput,
            onValueChange = { v -> phoneInput = v.filter { it.isDigit() || it in "+()- " }.take(18) },
            placeholder = "+7 900 000-00-00",
            keyboardType = KeyboardType.Phone,
            imeAction = ImeAction.Send,
            onAction = { if (normalized != null && !busy) scope.launch { send(normalized) } },
        )
        PrimaryButton("Позвонить мне", enabled = normalized != null && !busy) { normalized?.let { scope.launch { send(it) } } }
    } else {
        Text("Сейчас на ${formatPhone(sentTo)} позвонят. Отвечать не нужно: введите последние 4 цифры номера, который вам позвонил.", style = AppTheme.type.small, color = c.textSecondary)
        InputField(
            value = code,
            onValueChange = { v ->
                code = v.filter { it.isDigit() }.take(CODE_DIGITS)
                // Все цифры введены - входим сразу, без лишнего нажатия.
                if (code.length == CODE_DIGITS && !busy) scope.launch { confirm(sentTo, code) }
            },
            placeholder = "Последние 4 цифры номера",
            keyboardType = KeyboardType.NumberPassword,
        )
        Row(verticalAlignment = Alignment.CenterVertically) {
            TextButton(if (wait > 0) "Новый звонок через $wait с" else "Позвонить ещё раз", enabled = wait == 0 && !busy) {
                scope.launch { send(sentTo) }
            }
            Box(Modifier.weight(1f))
            TextButton("Другой номер", enabled = !busy) { phone = null; code = ""; error = null }
        }
    }
    error?.let { Text(it, style = AppTheme.type.small, color = c.warningText) }
}
