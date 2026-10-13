package ru.khudob1n.krasnodar.parking.ui.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Главная кнопка действия: на всю ширину, тёмная на светлой теме. */
@Composable
fun PrimaryButton(label: String, enabled: Boolean = true, modifier: Modifier = Modifier, onClick: () -> Unit) {
    val c = AppTheme.colors
    Box(
        modifier.fillMaxWidth().clip(AppTheme.shapes.mapButton)
            .background(if (enabled) c.textPrimary else c.backgroundSecondary)
            .clickable(enabled = enabled, role = Role.Button, onClick = onClick)
            .padding(vertical = 14.dp),
        contentAlignment = Alignment.Center,
    ) {
        Text(label, style = AppTheme.type.button, color = if (enabled) c.backgroundPrimary else c.functional)
    }
}

/** Второстепенная кнопка-ссылка. */
@Composable
fun TextButton(label: String, enabled: Boolean = true, onClick: () -> Unit) {
    val c = AppTheme.colors
    Box(Modifier.clip(AppTheme.shapes.pill).clickable(enabled = enabled, role = Role.Button, onClick = onClick).padding(horizontal = 8.dp, vertical = 8.dp)) {
        Text(label, style = AppTheme.type.button, color = if (enabled) c.parkingPaid else c.functional)
    }
}

/** Поле ввода в стиле приложения: серая плашка, подсказка внутри. */
@Composable
fun InputField(
    value: String,
    onValueChange: (String) -> Unit,
    placeholder: String,
    keyboardType: KeyboardType = KeyboardType.Text,
    imeAction: ImeAction = ImeAction.Done,
    onAction: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    val c = AppTheme.colors
    Box(modifier.fillMaxWidth().clip(AppTheme.shapes.search).background(c.backgroundSecondary).padding(horizontal = 16.dp, vertical = 14.dp)) {
        if (value.isEmpty()) Text(placeholder, style = AppTheme.type.body, color = c.functional)
        BasicTextField(
            value = value,
            onValueChange = onValueChange,
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            textStyle = AppTheme.type.body.copy(color = c.textPrimary),
            cursorBrush = SolidColor(c.textPrimary),
            keyboardOptions = KeyboardOptions(keyboardType = keyboardType, imeAction = imeAction),
            keyboardActions = KeyboardActions(onDone = { onAction() }, onGo = { onAction() }, onSend = { onAction() }),
        )
    }
}
