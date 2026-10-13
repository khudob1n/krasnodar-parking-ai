package ru.khudob1n.krasnodar.parking.ui.account

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme

/** Кнопка-крестик в углу карточки. */
@Composable
fun CloseButton(onClose: () -> Unit) {
    val c = AppTheme.colors
    Box(
        Modifier.size(36.dp).clip(CircleShape).background(c.backgroundSecondary).clickable(role = Role.Button, onClickLabel = "Закрыть", onClick = onClose),
        contentAlignment = Alignment.Center,
    ) { Text("✕", style = AppTheme.type.button, color = c.textSecondary) }
}

/**
 * Раздел профиля: подпись сверху и белая плашка со строками, разделёнными тонкой линией,
 * - как в настройках телефона.
 */
@Composable
fun ProfileSection(title: String, content: @Composable ColumnScope.() -> Unit) {
    val c = AppTheme.colors
    Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
        Text(title.uppercase(), style = AppTheme.type.small, color = c.functional, modifier = Modifier.padding(start = 16.dp))
        Column(Modifier.fillMaxWidth().clip(AppTheme.shapes.card).background(c.backgroundPrimary), content = content)
    }
}

/** Тонкая линия между строками раздела, с отступом слева, как у строк. */
@Composable
fun RowDivider() {
    Box(Modifier.padding(start = 16.dp).fillMaxWidth().height(1.dp).background(AppTheme.colors.backgroundSecondary))
}

/** Строка раздела: слева содержимое, справа необязательное действие. */
@Composable
fun ProfileRow(onClick: (() -> Unit)? = null, trailing: @Composable RowScope.() -> Unit = {}, content: @Composable ColumnScope.() -> Unit) {
    Row(
        Modifier.fillMaxWidth().then(if (onClick != null) Modifier.clickable(role = Role.Button, onClick = onClick) else Modifier)
            .padding(horizontal = 16.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(2.dp), content = content)
        trailing()
    }
}

/** Строка-кнопка «+ Добавить …» синим. */
@Composable
fun AddRow(label: String, onClick: () -> Unit) {
    ProfileRow(onClick = onClick) {
        Text("+  $label", style = AppTheme.type.button, color = AppTheme.colors.parkingPaid)
    }
}

/** Мелкое действие справа в строке: «Удалить». */
@Composable
fun RowAction(label: String, color: Color = AppTheme.colors.functional, enabled: Boolean = true, onClick: () -> Unit) {
    Text(
        label, style = AppTheme.type.small, color = color,
        modifier = Modifier.clip(AppTheme.shapes.pill).clickable(enabled = enabled, role = Role.Button, onClick = onClick).padding(8.dp),
    )
}

/** Госномер, похожий на настоящий: белая табличка с чёрной рамкой и регионом справа. */
@Composable
fun PlateBadge(plate: String) {
    val shape = RoundedCornerShape(6.dp)
    val main = if (plate.length >= 6) "${plate[0]} ${plate.substring(1, 4)} ${plate.substring(4, 6)}" else plate
    val region = if (plate.length > 6) plate.substring(6) else ""
    Row(
        Modifier.clip(shape).background(Color.White).border(1.5.dp, Color.Black, shape),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(main, style = AppTheme.type.mono, color = Color.Black, modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp))
        if (region.isNotEmpty()) {
            Box(Modifier.size(width = 1.5.dp, height = 24.dp).background(Color.Black))
            Text(region, style = AppTheme.type.mono, color = Color.Black, modifier = Modifier.padding(horizontal = 6.dp, vertical = 4.dp))
        }
    }
}

/** Ошибка запроса текстом для человека; [byStatus] - свои тексты для кодов ответа сервера. */
fun humanError(e: Exception, byStatus: (Int) -> String? = { null }): String = when (e) {
    is ru.khudob1n.krasnodar.parking.data.ApiException -> byStatus(e.status) ?: e.message ?: "Ошибка сервера"
    else -> "Нет связи с сервером"
}
