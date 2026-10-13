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

    /**
     * Все зоны города (GET /api/v1/zones). [etag] - метка уже сохранённых зон: если они не
     * изменились, сервер отвечает 304 и возвращается null.
     */
    suspend fun allZones(etag: String?): ZoneCatalogData? {
        val response = http.get(joinUrl(baseUrl, "/api/v1/zones")) {
            etag?.let { header(HttpHeaders.IfNoneMatch, it) }
        }
        if (response.status.value == 304) return null
        val body = ok(response)
        val zones = json.decodeFromString<NearbyResponse>(body).zones
        return ZoneCatalogData(etag = response.headers[HttpHeaders.ETag], zones = zones)
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

    /** Запросить код: сервер позвонит, код - последние 4 цифры номера (POST /api/v1/auth/code). */
    suspend fun requestCode(phone: String): CodeResult {
        val response = post("/api/v1/auth/code", buildJsonObject { put("phone", phone) }.toString())
        return when {
            response.status.isSuccess() -> CodeResult.Sent
            response.status.value == 429 ->
                CodeResult.TooSoon(response.headers[HttpHeaders.RetryAfter]?.toIntOrNull() ?: 60)
            else -> CodeResult.Failed(errorMessage(response.bodyAsText()))
        }
    }

    /** Войти по коду из звонка (POST /api/v1/auth/verify). Неверный код - ApiException 401. */
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

    /**
     * Данные карты - платёжному провайдеру (тестовый fakepay живёт на том же сервере, у настоящего
     * был бы свой адрес и SDK). Возвращает токен карты. Плохой номер или срок - ApiException 400.
     */
    suspend fun tokenizeCard(number: String, expMonth: Int, expYear: Int, cvc: String): String =
        json.decodeFromString<CardToken>(ok(post("/fakepay/v1/tokens", buildJsonObject {
            put("number", number)
            put("expMonth", expMonth)
            put("expYear", expYear)
            put("cvc", cvc)
        }.toString()))).token

    /** Карты пользователя (GET /api/v1/cards). */
    suspend fun cards(token: String): List<PaymentCard> =
        json.decodeFromString<CardsResponse>(ok(getAuthorized("/api/v1/cards", token))).cards

    /** Привязать карту по токену провайдера (POST /api/v1/cards). Банк отклонил - ApiException 402. */
    suspend fun addCard(token: String, cardToken: String): PaymentCard =
        json.decodeFromString<PaymentCard>(ok(post("/api/v1/cards", buildJsonObject { put("token", cardToken) }.toString(), token)))

    /** Отвязать карту (DELETE /api/v1/cards/{id}). */
    suspend fun deleteCard(token: String, id: Long) {
        ok(http.delete(joinUrl(baseUrl, "/api/v1/cards/$id")) { header(HttpHeaders.Authorization, "Bearer $token") })
    }

    /**
     * Начать парковку (POST /api/v1/sessions). С предоплатой сразу списывается цена [minutes] минут,
     * с постоплатой ([minutes] не важен) - ничего, пока не остановят. Машина уже стоит - ApiException
     * 409, карта не прошла или висит долг - 402.
     */
    suspend fun startSession(token: String, zone: Int, vehicleId: Long, minutes: Int, cardId: Long, mode: PaymentMode): ParkingSession =
        json.decodeFromString<ParkingSession>(ok(post("/api/v1/sessions", buildJsonObject {
            put("zone", zone)
            put("vehicleId", vehicleId)
            put("minutes", minutes)
            put("cardId", cardId)
            put("mode", if (mode == PaymentMode.Postpaid) "postpaid" else "prepaid")
        }.toString(), token)))

    /** Оплатить долг завершённой парковки картой [cardId] (POST /api/v1/sessions/{id}/pay). */
    suspend fun paySession(token: String, id: Long, cardId: Long): ParkingSession =
        json.decodeFromString<ParkingSession>(ok(post("/api/v1/sessions/$id/pay", buildJsonObject { put("cardId", cardId) }.toString(), token)))

    /** Продлить на [minutes] минут (POST /api/v1/sessions/{id}/extend). */
    suspend fun extendSession(token: String, id: Long, minutes: Int): ParkingSession =
        json.decodeFromString<ParkingSession>(ok(post("/api/v1/sessions/$id/extend", buildJsonObject { put("minutes", minutes) }.toString(), token)))

    /** Остановить сейчас: в итоге только фактические минуты (POST /api/v1/sessions/{id}/stop). */
    suspend fun stopSession(token: String, id: Long): ParkingSession =
        json.decodeFromString<ParkingSession>(ok(post("/api/v1/sessions/$id/stop", "{}", token)))

    /** Последние парковки, новые первыми (GET /api/v1/sessions). */
    suspend fun sessions(token: String, limit: Int = 20): List<ParkingSession> =
        json.decodeFromString<SessionsResponse>(ok(getAuthorized("/api/v1/sessions?limit=$limit", token))).sessions

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
