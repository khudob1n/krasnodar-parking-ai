package ru.khudob1n.krasnodar.parking.platform

import io.ktor.client.HttpClient
import io.ktor.client.engine.darwin.Darwin

// Бесплатный сервер после простоя просыпается до минуты: ждём с запасом.
actual fun createHttpClient(): HttpClient = HttpClient(Darwin) {
    engine {
        configureSession { timeoutIntervalForRequest = 90.0 }
    }
}
