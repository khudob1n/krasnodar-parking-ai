package ru.khudob1n.krasnodar.parking.platform

import io.ktor.client.HttpClient
import io.ktor.client.engine.okhttp.OkHttp
import java.util.concurrent.TimeUnit

// Бесплатный сервер после простоя просыпается до минуты: стандартных 10 секунд OkHttp мало.
actual fun createHttpClient(): HttpClient = HttpClient(OkHttp) {
    engine {
        config {
            connectTimeout(30, TimeUnit.SECONDS)
            readTimeout(90, TimeUnit.SECONDS)
        }
    }
}
