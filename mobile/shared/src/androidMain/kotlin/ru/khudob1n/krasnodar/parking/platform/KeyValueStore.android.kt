package ru.khudob1n.krasnodar.parking.platform

import android.content.Context

actual object KeyValueStore {
    private val prefs by lazy { Platform.appContext.getSharedPreferences("parking", Context.MODE_PRIVATE) }

    actual fun get(key: String): String? = prefs.getString(key, null)

    actual fun set(key: String, value: String?) {
        prefs.edit().apply { if (value == null) remove(key) else putString(key, value) }.apply()
    }
}
