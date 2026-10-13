package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlin.time.Duration
import kotlin.time.Instant

@Serializable
data class Tariff(val pricePerHour: Long, val paidFrom: String, val paidTo: String)

@Serializable
enum class SessionStatus {
    @SerialName("active") Active,
    @SerialName("finished") Finished,
}

/**
 * Парковка (backend/internal/parking/session.go). Время - строки RFC 3339 по Краснодару.
 * cost - копейки: у активной - сколько выйдет, если стоять до plannedEnd, у завершённой - итог.
 */
@Serializable
data class ParkingSession(
    val id: Long,
    val zone: Int,
    val plate: String,
    val tariff: Tariff,
    val startedAt: String,
    val plannedEnd: String,
    val endedAt: String? = null,
    val status: SessionStatus,
    val cost: Long,
) {
    val isActive: Boolean get() = status == SessionStatus.Active

    /** Сколько осталось до конца оплаченного времени (не меньше нуля). */
    fun remaining(now: Instant): Duration = (Instant.parse(plannedEnd) - now).coerceAtLeast(Duration.ZERO)
}

@Serializable
internal data class SessionsResponse(val sessions: List<ParkingSession>)

/** Обратный отсчёт: «42:07», «1:05:30». */
fun formatCountdown(d: Duration): String {
    val total = d.inWholeSeconds
    val h = total / 3600
    val m = (total % 3600) / 60
    val s = total % 60
    fun two(n: Long) = n.toString().padStart(2, '0')
    return if (h > 0) "$h:${two(m)}:${two(s)}" else "${two(m)}:${two(s)}"
}
