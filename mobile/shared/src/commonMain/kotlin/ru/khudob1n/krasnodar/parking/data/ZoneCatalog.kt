package ru.khudob1n.krasnodar.parking.data

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.withContext
import ru.khudob1n.krasnodar.parking.map.haversineMeters
import kotlin.time.Duration.Companion.seconds

/**
 * Что сейчас с зонами: [zones] - что рисуем, [loading] - идёт запрос к серверу,
 * [error] - последний запрос не удался (зоны из кэша при этом остаются на карте).
 */
data class ZoneCatalogState(
    val zones: List<Zone> = emptyList(),
    val loading: Boolean = false,
    val error: String? = null,
)

/**
 * Все зоны города на телефоне. При запуске сразу показывает зоны из кэша, потом спрашивает
 * сервер, не поменялись ли они (ETag), и если да - обновляет карту и кэш.
 * [readCache]/[writeCache] - файл кэша; в тестах подменяются.
 */
class ZoneCatalog(
    private val api: ApiClient,
    private val readCache: () -> String?,
    private val writeCache: (String) -> Unit,
) {
    private val _state = MutableStateFlow(ZoneCatalogState())
    val state: StateFlow<ZoneCatalogState> = _state.asStateFlow()

    private var etag: String? = null

    /** Зона по номеру из уже загруженных. */
    fun find(number: Int): Zone? = _state.value.zones.firstOrNull { it.number == number }

    /**
     * Загрузить зоны: кэш, затем сервер. Бесплатный сервер после простоя просыпается до минуты,
     * поэтому неудачный запрос повторяем [attempts] раз с паузой.
     */
    suspend fun load(attempts: Int = 3) {
        if (_state.value.zones.isEmpty()) {
            val cached = withContext(Dispatchers.Default) {
                runCatching { readCache()?.let { json.decodeFromString<ZoneCatalogData>(it) } }.getOrNull()
            }
            if (cached != null) {
                etag = cached.etag
                _state.update { it.copy(zones = cached.zones) }
            }
        }
        _state.update { it.copy(loading = true, error = null) }
        repeat(attempts) { attempt ->
            try {
                val fresh = api.allZones(etag)
                if (fresh != null) {
                    etag = fresh.etag
                    _state.update { it.copy(zones = fresh.zones) }
                    withContext(Dispatchers.Default) {
                        runCatching { writeCache(json.encodeToString(ZoneCatalogData.serializer(), fresh)) }
                    }
                }
                _state.update { it.copy(loading = false, error = null) }
                return
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                if (attempt == attempts - 1) {
                    _state.update { it.copy(loading = false, error = "Не удалось загрузить зоны: ${e.message ?: "нет связи с сервером"}") }
                } else {
                    delay((5 * (attempt + 1)).seconds)
                }
            }
        }
    }
}

/** [count] зон, ближайших к точке, с расстоянием в метрах до ближайшего участка зоны. */
fun nearestZones(zones: List<Zone>, from: LatLng, count: Int): List<Pair<Zone, Int>> =
    zones.map { it to distanceToZone(from, it) }.sortedBy { it.second }.take(count)

/** До ближайшей точки зоны: концы участков улиц или точка площадки. */
fun distanceToZone(from: LatLng, zone: Zone): Int {
    val points = zone.segments.flatMap { listOf(it.start, it.stop) }.ifEmpty { listOf(zone.location) }
    return points.minOf { haversineMeters(from, it) }
}
