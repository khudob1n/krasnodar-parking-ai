package ru.khudob1n.krasnodar.parking.data

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlin.time.Duration
import kotlin.time.Duration.Companion.minutes
import kotlin.time.Duration.Companion.seconds
import kotlin.time.Instant

class SessionTest {
    private val body = """{"id":5,"zone":1101,"plate":"А123ВС93","tariff":{"pricePerHour":6000,"paidFrom":"08:00","paidTo":"20:00"},
        "startedAt":"2026-10-13T18:00:00+03:00","plannedEnd":"2026-10-13T19:00:00+03:00","endedAt":null,"status":"active","cost":6000}"""

    @Test
    fun parsesActiveSession() {
        val s = json.decodeFromString<ParkingSession>(body)
        assertTrue(s.isActive)
        assertEquals(null, s.endedAt)
        assertEquals(6000, s.cost)
    }

    @Test
    fun parsesPostpaidWithDebt() {
        val s = json.decodeFromString<ParkingSession>(
            """{"id":6,"zone":1103,"plate":"А123ВС93","tariff":{"pricePerHour":6000,"paidFrom":"08:00","paidTo":"20:00"},
            "startedAt":"2026-10-13T18:00:00+03:00","plannedEnd":"2026-10-14T18:00:00+03:00","endedAt":"2026-10-13T18:30:00+03:00",
            "status":"finished","cost":3000,"mode":"postpaid","cardId":2,"paid":0,"debt":3000,"paymentError":"insufficient funds"}""",
        )
        assertTrue(s.isPostpaid)
        assertEquals(3000, s.debt)
        assertEquals("insufficient funds", s.paymentError)
        assertEquals(30.minutes, s.elapsed(Instant.parse("2026-10-13T15:30:00Z")))
        // Старый ответ без новых полей - предоплата без долга.
        assertEquals(PaymentMode.Prepaid, json.decodeFromString<ParkingSession>(body).mode)
    }

    @Test
    fun remainingTime() {
        val s = json.decodeFromString<ParkingSession>(body)
        assertEquals(15.minutes, s.remaining(Instant.parse("2026-10-13T15:45:00Z")))
        assertEquals(Duration.ZERO, s.remaining(Instant.parse("2026-10-13T16:30:00Z")), "после конца - ноль, а не минус")
    }

    @Test
    fun countdown() {
        assertEquals("42:07", formatCountdown(42.minutes + 7.seconds))
        assertEquals("1:05:30", formatCountdown(65.minutes + 30.seconds))
        assertEquals("00:00", formatCountdown(Duration.ZERO))
    }
}

class ParkingRemindersTest {
    @Test
    fun remindersBeforeAndAtEnd() {
        val s = json.decodeFromString<ParkingSession>(
            """{"id":5,"zone":1101,"plate":"А123ВС93","tariff":{"pricePerHour":6000,"paidFrom":"08:00","paidTo":"20:00"},
            "startedAt":"2026-10-13T18:00:00+03:00","plannedEnd":"2026-10-13T19:00:00+03:00","status":"active","cost":6000}""",
        )
        val (warn, end) = s.reminders()
        assertEquals(Instant.parse("2026-10-13T15:50:00Z").toEpochMilliseconds(), warn.atEpochMillis)
        assertEquals(Instant.parse("2026-10-13T16:00:00Z").toEpochMilliseconds(), end.atEpochMillis)
        assertEquals(reminderIds(5), listOf(warn.id, end.id))
        assertTrue(warn.text.startsWith("Зона 1101 · А 123 ВС 93"))
    }
}
