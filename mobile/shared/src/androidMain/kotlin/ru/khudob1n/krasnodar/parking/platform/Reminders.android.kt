package ru.khudob1n.krasnodar.parking.platform

import android.Manifest
import android.app.AlarmManager
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat

private const val CHANNEL_ID = "parking"
private const val EXTRA_ID = "id"
private const val EXTRA_TITLE = "title"
private const val EXTRA_TEXT = "text"

actual object Reminders {
    private val context get() = Platform.appContext
    private val alarms get() = context.getSystemService(AlarmManager::class.java)

    actual fun schedule(reminder: Reminder) {
        if (reminder.atEpochMillis <= System.currentTimeMillis()) return
        val intent = pending(reminder.id, reminder.title, reminder.text)
        // Точное время - если пользователь разрешил «будильники»; иначе система может сдвинуть
        // уведомление на несколько минут, но в режиме сна оно всё равно придёт.
        if (Build.VERSION.SDK_INT < 31 || alarms.canScheduleExactAlarms()) {
            alarms.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, reminder.atEpochMillis, intent)
        } else {
            alarms.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, reminder.atEpochMillis, intent)
        }
    }

    actual fun cancel(id: String) {
        alarms.cancel(pending(id, "", ""))
    }

    /** Один PendingIntent на id: тот же id - тот же будильник (замена или отмена). */
    private fun pending(id: String, title: String, text: String): PendingIntent {
        val intent = Intent(context, ReminderReceiver::class.java)
            .setAction("ru.khudob1n.krasnodar.parking.REMINDER")
            .setData(android.net.Uri.parse("reminder://$id"))
            .putExtra(EXTRA_ID, id)
            .putExtra(EXTRA_TITLE, title)
            .putExtra(EXTRA_TEXT, text)
        return PendingIntent.getBroadcast(context, 0, intent, PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE)
    }

    internal fun show(context: Context, id: String, title: String, text: String) {
        val manager = context.getSystemService(NotificationManager::class.java)
        manager.createNotificationChannel(NotificationChannel(CHANNEL_ID, "Парковка", NotificationManager.IMPORTANCE_HIGH))
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(context, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) return
        val open = context.packageManager.getLaunchIntentForPackage(context.packageName)?.let {
            PendingIntent.getActivity(context, 0, it, PendingIntent.FLAG_IMMUTABLE)
        }
        // Значок «P» приложения; не нашёлся - значок приложения целиком.
        val icon = context.resources.getIdentifier("ic_launcher_foreground", "drawable", context.packageName)
            .takeIf { it != 0 } ?: context.applicationInfo.icon
        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(icon)
            .setContentTitle(title)
            .setContentText(text)
            .setStyle(NotificationCompat.BigTextStyle().bigText(text))
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .setCategory(NotificationCompat.CATEGORY_REMINDER)
            .setAutoCancel(true)
            .apply { open?.let { setContentIntent(it) } }
            .build()
        NotificationManagerCompat.from(context).notify(id.hashCode(), notification)
    }
}

/** Будильник сработал - показать уведомление. */
class ReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val id = intent.getStringExtra(EXTRA_ID) ?: return
        Reminders.show(context, id, intent.getStringExtra(EXTRA_TITLE).orEmpty(), intent.getStringExtra(EXTRA_TEXT).orEmpty())
    }
}

@Composable
actual fun rememberNotificationPermissionRequest(): () -> Unit {
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestPermission()) { }
    return {
        if (Build.VERSION.SDK_INT >= 33 &&
            ContextCompat.checkSelfPermission(Platform.appContext, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) launcher.launch(Manifest.permission.POST_NOTIFICATIONS)
    }
}
