package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import ru.khudob1n.krasnodar.parking.data.ApiException
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.formatPlate
import ru.khudob1n.krasnodar.parking.data.normalizePlate
import ru.khudob1n.krasnodar.parking.ui.components.InputField
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.components.TextButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** «Мои машины»: список госномеров, удаление и добавление нового. */
@Composable
fun VehiclesSection(vehicles: VehicleRepository) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    val list by vehicles.vehicles.collectAsState()
    var plateInput by remember { mutableStateOf("") }
    var title by remember { mutableStateOf("") }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }

    suspend fun guarded(action: suspend () -> Unit) {
        busy = true
        error = null
        try {
            action()
        } catch (e: CancellationException) {
            throw e
        } catch (e: ApiException) {
            error = when (e.status) {
                409 -> "Эта машина уже добавлена"
                400 -> "Номер должен быть вида А123ВС93"
                else -> e.message
            }
        } catch (e: Exception) {
            error = e.message ?: "Нет связи с сервером"
        } finally {
            busy = false
        }
    }

    LaunchedEffect(Unit) { guarded { vehicles.refresh() } }

    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        Text("Мои машины", style = AppTheme.type.caption, color = c.textPrimary)
        list.forEach { v ->
            Row(
                Modifier.fillMaxWidth().clip(AppTheme.shapes.pill).background(c.backgroundSecondary).padding(start = 14.dp, end = 4.dp, top = 4.dp, bottom = 4.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(Modifier.weight(1f)) {
                    Text(formatPlate(v.plate), style = AppTheme.type.routeNumber, color = c.textPrimary)
                    if (v.title.isNotBlank()) Text(v.title, style = AppTheme.type.small, color = c.textSecondary)
                }
                TextButton("Удалить", enabled = !busy) { scope.launch { guarded { vehicles.delete(v.id) } } }
            }
        }
        val plate = normalizePlate(plateInput)
        InputField(
            value = plateInput,
            onValueChange = { plateInput = it.take(12) },
            placeholder = "Госномер, например А123ВС93",
            keyboardType = KeyboardType.Ascii,
            imeAction = ImeAction.Next,
        )
        InputField(value = title, onValueChange = { title = it.take(40) }, placeholder = "Название, если нужно")
        PrimaryButton("Добавить машину", enabled = plate != null && !busy) {
            if (plate != null) scope.launch {
                guarded {
                    vehicles.add(plate, title.trim())
                    plateInput = ""
                    title = ""
                }
            }
        }
        error?.let { Text(it, style = AppTheme.type.small, color = c.warningText) }
    }
}
