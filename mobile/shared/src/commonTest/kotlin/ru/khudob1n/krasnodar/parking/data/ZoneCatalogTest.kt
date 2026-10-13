package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals

class ZoneCatalogTest {
    private fun zone(number: Int, lat: Double, lon: Double) =
        Zone(number = number, kind = ZoneKind.Street, address = "", location = LatLng(lat, lon))

    @Test
    fun nearestFirst() {
        val zones = listOf(zone(1, 45.0, 39.0), zone(2, 45.01, 39.0), zone(3, 45.001, 39.0))
        val nearest = nearestZones(zones, LatLng(45.0, 39.0), 2)
        assertEquals(listOf(1, 3), nearest.map { it.first.number })
        assertEquals(0, nearest.first().second)
    }

    @Test
    fun distanceUsesClosestSegmentEnd() {
        val z = zone(1, 45.01, 39.0).copy(segments = listOf(Segment(LatLng(45.0, 39.0), LatLng(45.02, 39.0))))
        assertEquals(0, distanceToZone(LatLng(45.0, 39.0), z))
    }

    @Test
    fun cacheRoundTrip() {
        val data = ZoneCatalogData(etag = "\"abc\"", zones = listOf(zone(1103, 45.0, 39.0)))
        val text = json.encodeToString(ZoneCatalogData.serializer(), data)
        assertEquals(data, json.decodeFromString(ZoneCatalogData.serializer(), text))
    }
}
