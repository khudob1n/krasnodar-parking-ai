package ru.khudob1n.krasnodar.parking.platform

import java.io.File

actual object FileStore {
    private fun file(name: String) = File(Platform.appContext.filesDir, name)

    actual fun read(name: String): String? = file(name).takeIf { it.exists() }?.readText()

    actual fun write(name: String, text: String) {
        // Сначала во временный файл: если приложение убьют посреди записи, старый кэш останется целым.
        val tmp = file("$name.tmp")
        tmp.writeText(text)
        tmp.renameTo(file(name))
    }
}
