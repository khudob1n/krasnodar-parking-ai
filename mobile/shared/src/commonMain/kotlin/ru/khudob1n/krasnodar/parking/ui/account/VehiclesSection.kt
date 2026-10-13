package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
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
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.normalizePlate
import ru.khudob1n.krasnodar.parking.ui.components.InputField
import ru.khudob1n.krasnodar.parking.ui.components.PrimaryButton
import ru.khudob1n.krasnodar.parking.ui.components.TextButton
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** «Машины»: госномера табличками, удаление и форма добавления, которая открывается по кнопке. */
@Composable
fun VehiclesSection(vehicles: VehicleRepository) {
    val c = AppTheme.colors
    val scope = rememberCoroutineScope()
    val list by vehicles.vehicles.collectAsState()
    var adding by remember { mutableStateOf(false) }
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
        } catch (e: Exception) {
            error = humanError(e) { status ->
                when (status) {
                    409 -> "Эта машина уже добавлена"
                    400 -> "Номер должен быть вида А123ВС93"
                    else -> null
                }
            }
        } finally {
            busy = false
        }
    }

    LaunchedEffect(Unit) { guarded { vehicles.refresh() } }

    ProfileSection("Машины") {
        list.forEachIndexed { i, v ->
            if (i > 0) RowDivider()
            ProfileRow(trailing = { RowAction("Удалить", enabled = !busy) { scope.launch { guarded { vehicles.delete(v.id) } } } }) {
                PlateBadge(v.plate)
                if (v.title.isNotBlank()) Text(v.title, style = AppTheme.type.small, color = c.textSecondary)
            }
        }
        if (list.isNotEmpty()) RowDivider()
        if (!adding) {
            AddRow(if (list.isEmpty()) "Добавить машину" else "Добавить ещё машину") { adding = true }
        } else {
            val plate = normalizePlate(plateInput)
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                InputField(
                    value = plateInput,
                    onValueChange = { plateInput = it.take(12) },
                    placeholder = "Госномер, например А123ВС93",
                    keyboardType = KeyboardType.Ascii,
                    imeAction = ImeAction.Next,
                )
                InputField(value = title, onValueChange = { title = it.take(40) }, placeholder = "Название, если нужно: «Рабочая»")
                PrimaryButton("Добавить", enabled = plate != null && !busy) {
                    if (plate != null) scope.launch {
                        guarded {
                            vehicles.add(plate, title.trim())
                            plateInput = ""
                            title = ""
                            adding = false
                        }
                    }
                }
                TextButton("Отмена") { adding = false; error = null }
            }
        }
        error?.let { Text(it, style = AppTheme.type.small, color = c.warningText, modifier = Modifier.padding(start = 16.dp, end = 16.dp, bottom = 12.dp)) }
    }
}
