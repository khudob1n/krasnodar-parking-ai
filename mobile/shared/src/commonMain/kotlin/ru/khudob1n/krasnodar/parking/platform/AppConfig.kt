package ru.khudob1n.krasnodar.parking.platform

/** Настройки сборки: Android-приложение заполняет их из BuildConfig при запуске. */
object AppConfig {
    /** Адрес бэкенда без «/» на конце, например https://parking.example.ru. */
    var apiBaseUrl: String = "http://127.0.0.1:8080"
    var versionName: String = ""
    var isDebug: Boolean = false
}
