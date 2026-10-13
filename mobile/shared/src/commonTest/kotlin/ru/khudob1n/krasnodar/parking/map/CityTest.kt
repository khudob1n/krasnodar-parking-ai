package ru.khudob1n.krasnodar.parking.map

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class GeoTest {
    @Test
    fun haversine() {
        // Один градус широты - около 111 км.
        val d = haversineMeters(ru.khudob1n.krasnodar.parking.data.LatLng(45.0, 39.0), ru.khudob1n.krasnodar.parking.data.LatLng(46.0, 39.0))
        assertTrue(d in 111_000..111_300, "1°: $d")
        assertEquals(0, haversineMeters(City.center, City.center))
    }
}
