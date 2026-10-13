package ru.khudob1n.krasnodar.parking.data

import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.http.isSuccess
import kotlinx.coroutines.CancellationException

/** Клиент API бэкенда (backend/README.md). Пока умеет только проверить, что сервер жив. */
class ApiClient(private val http: HttpClient, private val baseUrl: String) {
    /** GET /healthz: true, если сервер ответил 2xx. Сетевые ошибки - тоже «недоступен». */
    suspend fun isAlive(): Boolean = try {
        http.get(joinUrl(baseUrl, "/healthz")).status.isSuccess()
    } catch (e: CancellationException) {
        throw e
    } catch (_: Exception) {
        false
    }
}

internal fun joinUrl(baseUrl: String, path: String) = baseUrl.trimEnd('/') + path
