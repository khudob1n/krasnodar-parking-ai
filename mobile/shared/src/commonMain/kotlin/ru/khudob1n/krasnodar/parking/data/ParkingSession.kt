package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlin.time.Duration
import kotlin.time.Instant

@Serializable
data class Tariff(val pricePerHour: Long, val paidFrom: String, val paidTo: String)

/** Как платим: заранее за выбранное время или по факту, когда остановили. */
@Serializable
enum class PaymentMode {
    @SerialName("prepaid") Prepaid,
    @SerialName("postpaid") Postpaid,
}

@Serializable
enum class SessionStatus {
    @SerialName("active") Active,
    @SerialName("finished") Finished,
}

/**
 * Парковка (backend/internal/parking/session.go). Время - строки RFC 3339.
 * cost - копейки: у завершённой - итог; у активной с предоплатой - сколько выйдет, если стоять до
 * plannedEnd, с постоплатой - сколько набежало к моменту ответа сервера.
 * paid - сколько уже списано (за вычетом возвратов), debt - сколько не удалось списать в конце
 * постоплатной парковки, paymentError - почему.
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
    val mode: PaymentMode = PaymentMode.Prepaid,
    val cardId: Long? = null,
    val paid: Long = 0,
    val debt: Long = 0,
    val paymentError: String? = null,
) {
    val isActive: Boolean get() = status == SessionStatus.Active
    val isPostpaid: Boolean get() = mode == PaymentMode.Postpaid

    /** Сколько стоим (для постоплаты: таймер идёт вперёд). */
    fun elapsed(now: Instant): Duration = (now - Instant.parse(startedAt)).coerceAtLeast(Duration.ZERO)

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
