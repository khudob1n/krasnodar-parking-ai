package ru.khudob1n.krasnodar.parking.platform

import platform.Foundation.NSUserDefaults

actual object KeyValueStore {
    private val defaults get() = NSUserDefaults.standardUserDefaults

    actual fun get(key: String): String? = defaults.stringForKey(key)

    actual fun set(key: String, value: String?) {
        if (value == null) defaults.removeObjectForKey(key) else defaults.setObject(value, forKey = key)
    }
}
