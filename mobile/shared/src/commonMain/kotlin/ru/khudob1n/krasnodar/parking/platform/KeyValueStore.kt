package ru.khudob1n.krasnodar.parking.platform

/**
 * Маленькое хранилище строк на устройстве: SharedPreferences на Android, NSUserDefaults на iOS.
 * Сейчас в нём только токены входа.
 */
expect object KeyValueStore {
    fun get(key: String): String?

    /** null - удалить. */
    fun set(key: String, value: String?)
}
