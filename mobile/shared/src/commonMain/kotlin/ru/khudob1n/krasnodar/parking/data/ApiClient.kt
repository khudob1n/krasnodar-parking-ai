package ru.khudob1n.krasnodar.parking.data

import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.client.request.parameter
import io.ktor.client.statement.bodyAsText
import io.ktor.http.isSuccess
import kotlinx.coroutines.CancellationException
import kotlinx.serialization.json.Json

/** Ответ сервера не 2xx: status - код, message - поле error из тела, если есть. */
class ApiException(val status: Int, message: String) : Exception(message)

/** Клиент API бэкенда (backend/README.md). */
class ApiClient(private val http: HttpClient, private val baseUrl: String) {
    /** GET /healthz: true, если сервер ответил 2xx. Сетевые ошибки - тоже «недоступен». */
    suspend fun isAlive(): Boolean = try {
        http.get(joinUrl(baseUrl, "/healthz")).status.isSuccess()
    } catch (e: CancellationException) {
        throw e
    } catch (_: Exception) {
        false
    }

    /** Зоны в радиусе от точки, ближайшие первыми (GET /api/v1/zones/nearby). */
    suspend fun nearby(point: LatLng, radiusMeters: Int, limit: Int = 100): List<Zone> {
        val response = http.get(joinUrl(baseUrl, "/api/v1/zones/nearby")) {
            parameter("lat", point.lat)
            parameter("lon", point.lon)
            parameter("radius", radiusMeters)
            parameter("limit", limit)
        }
        val body = response.bodyAsText()
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(body))
        return json.decodeFromString<NearbyResponse>(body).zones
    }
}

// coerceInputValues: null от Go (пустой срез) превращается в значение по умолчанию.
internal val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

internal fun joinUrl(baseUrl: String, path: String) = baseUrl.trimEnd('/') + path

/** Текст из {"error": "..."} бэкенда, иначе - тело как есть. */
internal fun errorMessage(body: String): String = runCatching {
    json.decodeFromString<Map<String, String>>(body)["error"]
}.getOrNull() ?: body
