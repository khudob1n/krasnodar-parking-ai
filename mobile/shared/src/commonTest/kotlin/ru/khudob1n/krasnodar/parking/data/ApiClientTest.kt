package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals

class ApiClientTest {
    @Test
    fun urlJoinsWithoutDoubleSlash() {
        assertEquals("https://example.ru/healthz", joinUrl("https://example.ru/", "/healthz"))
        assertEquals("https://example.ru/healthz", joinUrl("https://example.ru", "/healthz"))
    }
}
