package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class VehicleTest {
    @Test
    fun normalizesPlates() {
        assertEquals("А123ВС93", normalizePlate("а 123 bc 93"))
        assertEquals("М777ОР123", normalizePlate("M777OP-123"))
        assertNull(normalizePlate("Б123ВС93"), "буквы Б в номерах не бывает")
        assertNull(normalizePlate("А12ВС93"))
    }

    @Test
    fun formatsPlates() {
        assertEquals("А 123 ВС 93", formatPlate("А123ВС93"))
        assertEquals("М 777 ОР 123", formatPlate("М777ОР123"))
    }
}
