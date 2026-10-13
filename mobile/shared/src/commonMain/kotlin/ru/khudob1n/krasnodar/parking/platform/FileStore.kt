package ru.khudob1n.krasnodar.parking.platform

/**
 * Файлы приложения для больших данных (кэш зон ~1 МБ), которым не место в KeyValueStore.
 * Android - context.filesDir, iOS - Library/Caches.
 */
expect object FileStore {
    fun read(name: String): String?

    fun write(name: String, text: String)
}
