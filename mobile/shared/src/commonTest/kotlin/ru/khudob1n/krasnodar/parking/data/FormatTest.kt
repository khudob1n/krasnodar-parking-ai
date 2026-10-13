package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals

class FormatTest {
    private fun zone(kind: ZoneKind = ZoneKind.Street, price: Long = 6000, from: String = "08:00", to: String = "20:00") =
        Zone(number = 1101, kind = kind, address = "ул. Красная", location = LatLng(45.0, 39.0), pricePerHour = price, paidFrom = from, paidTo = to)

    @Test
    fun rubles() {
        assertEquals("60 ₽", formatRub(6000))
        assertEquals("45,50 ₽", formatRub(4550))
        assertEquals("0,05 ₽", formatRub(5))
    }

    @Test
    fun price() {
        assertEquals("60 ₽/ч", zone().priceLabel())
        assertEquals("Бесплатно", zone(kind = ZoneKind.Free).priceLabel())
        assertEquals("Бесплатно", zone(price = 0).priceLabel())
    }

    @Test
    fun paidHours() {
        assertEquals("платно 08:00–20:00", zone().paidHoursLabel())
        assertEquals("круглосуточно", zone(from = "00:00", to = "00:00").paidHoursLabel())
        assertEquals("", zone(kind = ZoneKind.Free).paidHoursLabel())
    }

    @Test
    fun distance() {
        assertEquals("350 м", formatDistance(350))
        assertEquals("1 км", formatDistance(1000))
        assertEquals("1,2 км", formatDistance(1234))
        assertEquals("2 км", formatDistance(1960))
    }

    @Test
    fun parsesNearbyResponse() {
        val body = """{"zones":[{"number":1101,"kind":"street","address":"ул. Красная, 1","location":{"lat":45.03,"lon":38.97},
            "segments":[{"start":{"lat":45.03,"lon":38.97},"stop":{"lat":45.031,"lon":38.971}}],"pricePerHour":6000,
            "paidFrom":"08:00","paidTo":"20:00","totalPlaces":40,"disabledPlaces":4,"inBuilding":false,"tariffText":"60 руб/час",
            "distanceMeters":120,"somethingNew":true}]}"""
        val zones = json.decodeFromString<NearbyResponse>(body).zones
        assertEquals(1, zones.size)
        assertEquals(ZoneKind.Street, zones[0].kind)
        assertEquals(120, zones[0].distanceMeters)
        assertEquals(LatLng(45.031, 38.971), zones[0].segments[0].stop)
    }

    @Test
    fun errorBody() {
        assertEquals("lat is required", errorMessage("""{"error":"lat is required"}"""))
        assertEquals("Bad Gateway", errorMessage("Bad Gateway"))
    }

    @Test
    fun minutes() {
        assertEquals("15 мин", formatMinutes(15))
        assertEquals("1 ч", formatMinutes(60))
        assertEquals("2 ч 30 мин", formatMinutes(150))
    }

    @Test
    fun clock() {
        assertEquals("18:30", clockOf("2026-10-13T18:30:00+03:00"))
        // Сервер может прислать время в UTC - показываем краснодарское.
        assertEquals("18:30", clockOf("2026-10-13T15:30:00Z"))
        assertEquals("13 окт", dayOf("2026-10-13T15:30:00Z"))
        assertEquals("14 окт", dayOf("2026-10-13T21:30:00Z"))
    }

    @Test
    fun zoneNumber() {
        assertEquals(1101, parseZoneNumber(" 1101 "))
        assertEquals(1101, parseZoneNumber("№1101"))
        assertEquals(null, parseZoneNumber("Красная"))
        assertEquals(null, parseZoneNumber("0"))
    }

    @Test
    fun parsesQuote() {
        val q = json.decodeFromString<Quote>("""{"zone":1101,"start":"2026-10-13T19:30:00+03:00","end":"2026-10-13T20:30:00+03:00","total":3000,"paidMinutes":30}""")
        assertEquals(3000, q.total)
        assertEquals(30, q.paidMinutes)
    }
}
