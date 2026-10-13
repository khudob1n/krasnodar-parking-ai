package ru.khudob1n.krasnodar.parking.map

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import ru.khudob1n.krasnodar.parking.data.parseZoneNumber
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Поле «Номер зоны»: номер с таблички у парковки - самый быстрый путь к оплате. */
@Composable
fun ZoneSearch(modifier: Modifier = Modifier, onSearch: (Int) -> Unit) {
    val c = AppTheme.colors
    val focus = LocalFocusManager.current
    var text by remember { mutableStateOf("") }
    Box(modifier.clip(AppTheme.shapes.search).background(c.backgroundPrimary).padding(horizontal = 16.dp, vertical = 12.dp)) {
        if (text.isEmpty()) Text("Номер зоны с таблички", style = AppTheme.type.body, color = c.functional)
        BasicTextField(
            value = text,
            onValueChange = { value -> text = value.filter { it.isDigit() }.take(6) },
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            textStyle = AppTheme.type.body.copy(color = c.textPrimary),
            cursorBrush = SolidColor(c.textPrimary),
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number, imeAction = ImeAction.Search),
            keyboardActions = KeyboardActions(onSearch = {
                parseZoneNumber(text)?.let {
                    focus.clearFocus()
                    onSearch(it)
                }
            }),
        )
    }
}
