package ru.khudob1n.krasnodar.parking.data

import io.ktor.client.HttpClient
import io.ktor.client.request.delete
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.parameter
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.HttpResponse
import io.ktor.client.statement.bodyAsText
import io.ktor.http.content.TextContent
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.isSuccess
import kotlinx.coroutines.CancellationException
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.put

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

    /** Зона по номеру с таблички; null - такой зоны нет (GET /api/v1/zones/{number}). */
    suspend fun zone(number: Int): Zone? {
        val response = http.get(joinUrl(baseUrl, "/api/v1/zones/$number"))
        val body = response.bodyAsText()
        if (response.status.value == 404) return null
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(body))
        return json.decodeFromString<Zone>(body)
    }

    /** Сколько стоит стоять [minutes] минут с этого момента (GET /api/v1/zones/{number}/price). */
    suspend fun price(number: Int, minutes: Int): Quote {
        val response = http.get(joinUrl(baseUrl, "/api/v1/zones/$number/price")) {
            parameter("minutes", minutes)
        }
        val body = response.bodyAsText()
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(body))
        return json.decodeFromString<Quote>(body)
    }

    /** Отправить код в SMS (POST /api/v1/auth/code). */
    suspend fun requestCode(phone: String): CodeResult {
        val response = post("/api/v1/auth/code", buildJsonObject { put("phone", phone) }.toString())
        return when {
            response.status.isSuccess() -> CodeResult.Sent
            response.status.value == 429 ->
                CodeResult.TooSoon(response.headers[HttpHeaders.RetryAfter]?.toIntOrNull() ?: 60)
            else -> CodeResult.Failed(errorMessage(response.bodyAsText()))
        }
    }

    /** Войти по коду из SMS (POST /api/v1/auth/verify). Неверный код - ApiException 401. */
    suspend fun verify(phone: String, code: String): AuthSession =
        session(post("/api/v1/auth/verify", buildJsonObject { put("phone", phone); put("code", code) }.toString()))

    /** Новая пара токенов по refresh-токену (POST /api/v1/auth/refresh). */
    suspend fun refresh(refreshToken: String): AuthSession =
        session(post("/api/v1/auth/refresh", buildJsonObject { put("refreshToken", refreshToken) }.toString()))

    /** Забыть refresh-токен на сервере (POST /api/v1/auth/logout). */
    suspend fun logout(refreshToken: String) {
        val response = post("/api/v1/auth/logout", buildJsonObject { put("refreshToken", refreshToken) }.toString())
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(response.bodyAsText()))
    }

    private suspend fun session(response: HttpResponse): AuthSession {
        val body = response.bodyAsText()
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(body))
        return json.decodeFromString<AuthSession>(body)
    }

    /** POST с телом JSON; [token] - access-токен для запросов от имени пользователя. */
    internal suspend fun post(path: String, body: String, token: String? = null): HttpResponse =
        http.post(joinUrl(baseUrl, path)) {
            token?.let { header(HttpHeaders.Authorization, "Bearer $it") }
            setBody(TextContent(body, ContentType.Application.Json))
        }

    /** Машины пользователя (GET /api/v1/vehicles). */
    suspend fun vehicles(token: String): List<Vehicle> =
        json.decodeFromString<VehiclesResponse>(ok(getAuthorized("/api/v1/vehicles", token))).vehicles

    /** Добавить машину (POST /api/v1/vehicles). Уже есть - ApiException 409, плохой номер - 400. */
    suspend fun addVehicle(token: String, plate: String, title: String): Vehicle =
        json.decodeFromString<Vehicle>(ok(post("/api/v1/vehicles", buildJsonObject { put("plate", plate); put("title", title) }.toString(), token)))

    /** Удалить машину (DELETE /api/v1/vehicles/{id}). */
    suspend fun deleteVehicle(token: String, id: Long) {
        ok(http.delete(joinUrl(baseUrl, "/api/v1/vehicles/$id")) { header(HttpHeaders.Authorization, "Bearer $token") })
    }

    /** Тело ответа, если он 2xx; иначе ApiException с текстом ошибки сервера. */
    internal suspend fun ok(response: HttpResponse): String {
        val body = response.bodyAsText()
        if (!response.status.isSuccess()) throw ApiException(response.status.value, errorMessage(body))
        return body
    }

    /** GET от имени пользователя. */
    internal suspend fun getAuthorized(path: String, token: String): HttpResponse =
        http.get(joinUrl(baseUrl, path)) { header(HttpHeaders.Authorization, "Bearer $token") }
}

// coerceInputValues: null от Go (пустой срез) превращается в значение по умолчанию.
internal val json = Json { ignoreUnknownKeys = true; coerceInputValues = true }

internal fun joinUrl(baseUrl: String, path: String) = baseUrl.trimEnd('/') + path

/** Текст из {"error": "..."} бэкенда, иначе - тело как есть. */
internal fun errorMessage(body: String): String = runCatching {
    json.decodeFromString<Map<String, String>>(body)["error"]
}.getOrNull() ?: body
