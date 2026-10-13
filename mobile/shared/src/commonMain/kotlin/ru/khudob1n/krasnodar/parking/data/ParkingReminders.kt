package ru.khudob1n.krasnodar.parking.data

import ru.khudob1n.krasnodar.parking.platform.Reminder
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Instant

/** За сколько до конца предупреждаем - то же, что сервер (backend/internal/reminders). */
private val WARN_BEFORE = 10.minutes

/**
 * Напоминания для парковки: за 10 минут до конца и в момент окончания. id включает номер
 * сессии, поэтому продление перепланирует те же уведомления, а остановка - отменяет.
 */
fun ParkingSession.reminders(): List<Reminder> {
    val end = Instant.parse(plannedEnd)
    val where = "Зона $zone · ${formatPlate(plate)}"
    return listOf(
        Reminder(warnId(id), (end - WARN_BEFORE).toEpochMilliseconds(), "Парковка закончится через 10 минут", "$where. Продлите, если ещё стоите."),
        Reminder(endId(id), end.toEpochMilliseconds(), "Оплаченное время вышло", "$where. Продлите парковку, чтобы не получить штраф."),
    )
}

fun reminderIds(sessionId: Long): List<String> = listOf(warnId(sessionId), endId(sessionId))

private fun warnId(id: Long) = "parking-$id-warn"

private fun endId(id: Long) = "parking-$id-end"
