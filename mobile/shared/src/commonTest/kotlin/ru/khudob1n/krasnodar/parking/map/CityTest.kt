package ru.khudob1n.krasnodar.parking.map

import kotlin.math.abs
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class CityTest {
    @Test
    fun radiusShrinksWithZoom() {
        val r13 = radiusForZoom(13.0, 45.0)
        val r14 = radiusForZoom(14.0, 45.0)
        assertTrue(r13 in 2900..3200, "z13: $r13")
        assertTrue(abs(r13 - 2 * r14) <= 1, "каждый шаг масштаба - вдвое меньше: $r13, $r14")
    }

    @Test
    fun radiusIsCapped() {
        assertEquals(5000, radiusForZoom(10.0, 45.0))
        assertEquals(100, radiusForZoom(20.0, 45.0))
    }
}

class GeoTest {
    @Test
    fun haversine() {
        // Один градус широты - около 111 км.
        val d = haversineMeters(ru.khudob1n.krasnodar.parking.data.LatLng(45.0, 39.0), ru.khudob1n.krasnodar.parking.data.LatLng(46.0, 39.0))
        assertTrue(d in 111_000..111_300, "1°: $d")
        assertEquals(0, haversineMeters(City.center, City.center))
    }
}
