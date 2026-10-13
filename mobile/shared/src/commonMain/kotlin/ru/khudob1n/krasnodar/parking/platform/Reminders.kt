package ru.khudob1n.krasnodar.parking.platform

import androidx.compose.runtime.Composable

/** Одно уведомление: [id] - стабильный ключ (по нему заменяем и отменяем), [atEpochMillis] - когда показать. */
data class Reminder(val id: String, val atEpochMillis: Long, val title: String, val text: String)

/**
 * Локальные уведомления на устройстве: приходят и без интернета, и когда приложение закрыто.
 * Android - AlarmManager, iOS - UNUserNotificationCenter.
 */
expect object Reminders {
    /** Запланировать или перепланировать (тот же id - замена). Время в прошлом - ничего. */
    fun schedule(reminder: Reminder)

    fun cancel(id: String)
}

/** Запрос разрешения на уведомления (Android 13+, iOS). Возвращает функцию «спросить». */
@Composable
expect fun rememberNotificationPermissionRequest(): () -> Unit
