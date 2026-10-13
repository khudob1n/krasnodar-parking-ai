package ru.khudob1n.krasnodar.parking.map

import ru.khudob1n.krasnodar.parking.data.LatLng
import kotlin.math.PI
import kotlin.math.asin
import kotlin.math.cos
import kotlin.math.roundToInt
import kotlin.math.sin
import kotlin.math.sqrt

/** Расстояние по поверхности Земли в метрах (формула гаверсинусов). */
fun haversineMeters(a: LatLng, b: LatLng): Int {
    val r = 6_371_000.0
    val dLat = (b.lat - a.lat) * PI / 180
    val dLon = (b.lon - a.lon) * PI / 180
    val h = sin(dLat / 2).let { it * it } + cos(a.lat * PI / 180) * cos(b.lat * PI / 180) * sin(dLon / 2).let { it * it }
    return (2 * r * asin(sqrt(h))).roundToInt()
}
