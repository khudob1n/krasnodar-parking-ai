package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class AuthTest {
    @Test
    fun normalizesPhones() {
        assertEquals("+79181234567", normalizePhone("8 918 123-45-67"))
        assertEquals("+79181234567", normalizePhone("+7 (918) 1234567"))
        assertEquals("+79181234567", normalizePhone("9181234567"))
        assertNull(normalizePhone("8 861 123-45-67"), "городской номер SMS не получит")
        assertNull(normalizePhone("12345"))
    }

    @Test
    fun formatsPhone() {
        assertEquals("+7 918 123-45-67", formatPhone("+79181234567"))
    }

    @Test
    fun parsesSession() {
        val s = json.decodeFromString<AuthSession>(
            """{"accessToken":"a","refreshToken":"r","expiresIn":900,"user":{"id":7,"phone":"+79181234567","createdAt":"2026-10-13T10:00:00Z"}}""",
        )
        assertEquals(7, s.user.id)
        assertEquals("r", s.refreshToken)
    }
}
