package ru.khudob1n.krasnodar.parking.map

import ru.khudob1n.krasnodar.parking.data.LatLng
import kotlin.math.PI
import kotlin.math.cos
import kotlin.math.min
import kotlin.math.pow
import kotlin.math.roundToInt

/** Центр и рамка карты - как у «Транспорта Краснодара». */
object City {
    val center = LatLng(45.0355, 38.9753)
    const val DEFAULT_ZOOM = 13.0

    const val NORTH = 45.35
    const val EAST = 39.45
    const val SOUTH = 44.85
    const val WEST = 38.6
}

/** Мельче этого масштаба зоны не грузим: в кадре полгорода, линии сливаются. */
const val ZONES_MIN_ZOOM = 12.0

/** Больше бэкенд не отдаёт (maxRadius в backend/internal/httpapi/nearby.go). */
private const val MAX_RADIUS = 5000

/**
 * Радиус, который накрывает экран телефона от центра до угла (~450 dp) на этом масштабе.
 * Метров в dp у MapLibre (тайлы 512): 78271.5 · cos(широта) / 2^zoom.
 */
fun radiusForZoom(zoom: Double, lat: Double): Int {
    val metersPerDp = 78271.517 * cos(lat * PI / 180) / 2.0.pow(zoom)
    return min(MAX_RADIUS, (metersPerDp * 450).roundToInt()).coerceAtLeast(100)
}
