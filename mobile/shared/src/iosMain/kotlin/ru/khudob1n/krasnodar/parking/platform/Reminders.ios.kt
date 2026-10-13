package ru.khudob1n.krasnodar.parking.platform

import androidx.compose.runtime.Composable
import platform.Foundation.NSDate
import platform.Foundation.timeIntervalSince1970
import platform.UserNotifications.UNAuthorizationOptionAlert
import platform.UserNotifications.UNAuthorizationOptionSound
import platform.UserNotifications.UNMutableNotificationContent
import platform.UserNotifications.UNNotificationRequest
import platform.UserNotifications.UNNotificationSound
import platform.UserNotifications.UNTimeIntervalNotificationTrigger
import platform.UserNotifications.UNUserNotificationCenter

actual object Reminders {
    private val center get() = UNUserNotificationCenter.currentNotificationCenter()

    actual fun schedule(reminder: Reminder) {
        val seconds = reminder.atEpochMillis / 1000.0 - NSDate().timeIntervalSince1970
        if (seconds <= 0) return
        val content = UNMutableNotificationContent()
        content.setTitle(reminder.title)
        content.setBody(reminder.text)
        content.setSound(UNNotificationSound.defaultSound)
        val trigger = UNTimeIntervalNotificationTrigger.triggerWithTimeInterval(seconds, repeats = false)
        // Тот же идентификатор заменяет запланированное уведомление.
        center.addNotificationRequest(UNNotificationRequest.requestWithIdentifier(reminder.id, content, trigger), withCompletionHandler = null)
    }

    actual fun cancel(id: String) {
        center.removePendingNotificationRequestsWithIdentifiers(listOf(id))
    }
}

@Composable
actual fun rememberNotificationPermissionRequest(): () -> Unit = {
    UNUserNotificationCenter.currentNotificationCenter()
        .requestAuthorizationWithOptions(UNAuthorizationOptionAlert or UNAuthorizationOptionSound) { _, _ -> }
}
