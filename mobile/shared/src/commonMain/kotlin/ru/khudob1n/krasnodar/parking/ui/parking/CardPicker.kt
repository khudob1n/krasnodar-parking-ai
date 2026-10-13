package ru.khudob1n.krasnodar.parking.ui.parking

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.horizontalScroll
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Ряд вариантов-«таблеток», выбранный - тёмный: машины, карты, способ оплаты. */
@Composable
fun <T> ChoiceRow(options: List<T>, selected: T, label: (T) -> String, modifier: Modifier = Modifier, onSelect: (T) -> Unit) {
    val c = AppTheme.colors
    Row(modifier.horizontalScroll(rememberScrollState()), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        options.forEach { option ->
            val isSelected = option == selected
            Box(
                Modifier.clip(AppTheme.shapes.pill).background(if (isSelected) c.textPrimary else c.backgroundSecondary)
                    .clickable(role = Role.Button) { onSelect(option) }.padding(horizontal = 12.dp, vertical = 8.dp),
            ) {
                Text(label(option), style = AppTheme.type.button, color = if (isSelected) c.backgroundPrimary else c.textPrimary)
            }
        }
    }
}
