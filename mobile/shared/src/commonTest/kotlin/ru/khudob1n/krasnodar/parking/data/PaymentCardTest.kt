package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class PaymentCardTest {
    @Test
    fun cardNumberChecksLuhn() {
        assertEquals("4242424242424242", normalizeCardNumber("4242 4242 4242 4242"))
        TEST_CARDS.forEach { (number, _) -> assertEquals(number.filter { it.isDigit() }, normalizeCardNumber(number)) }
        assertNull(normalizeCardNumber("4242 4242 4242 4241"), "контрольная сумма не сходится")
        assertNull(normalizeCardNumber("4242 4242"), "короткий")
    }

    @Test
    fun formatsAsTyped() {
        assertEquals("4242 4242 42", formatCardNumber("4242424242"))
        assertEquals("4242 4242 4242 4242", formatCardNumber("4242-4242-4242-4242-99"))
    }

    @Test
    fun expiry() {
        assertEquals(12 to 2030, parseExpiry("12/30"))
        assertNull(parseExpiry("13/30"))
        assertNull(parseExpiry("1/30"))
        assertEquals("Visa •• 4242", PaymentCard(1, "4242", "visa", 1, 2031).label())
        assertEquals("01/31", PaymentCard(1, "4242", "visa", 1, 2031).expiry())
    }
}
