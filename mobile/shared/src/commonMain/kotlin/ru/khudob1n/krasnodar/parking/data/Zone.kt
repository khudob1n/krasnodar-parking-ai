package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Точка на карте в градусах. */
@Serializable
data class LatLng(val lat: Double, val lon: Double)

/** Участок улицы, вдоль которого можно парковаться. */
@Serializable
data class Segment(val start: LatLng, val stop: LatLng)

@Serializable
enum class ZoneKind {
    @SerialName("street") Street,
    @SerialName("lot") Lot,
    @SerialName("free") Free,
}

/**
 * Парковочная зона - как её отдаёт бэкенд (backend/internal/zones/zone.go). Цена - в копейках,
 * платные часы - "ЧЧ:ММ", одинаковые paidFrom и paidTo - круглосуточно.
 */
@Serializable
data class Zone(
    val number: Int,
    val kind: ZoneKind,
    val address: String,
    val description: String = "",
    val location: LatLng,
    val segments: List<Segment> = emptyList(),
    val pricePerHour: Long = 0,
    val paidFrom: String = "",
    val paidTo: String = "",
    val totalPlaces: Int = 0,
    val disabledPlaces: Int = 0,
    val inBuilding: Boolean = false,
    val tariffText: String = "",
    /** Есть только в ответе «рядом»: метров до ближайшего участка зоны. */
    val distanceMeters: Int? = null,
)

/**
 * Расчёт цены: total - копейки к оплате, paidMinutes - сколько минут попало в платные часы
 * (оплата поминутная, бесплатные часы не считаются). start и end - строки RFC 3339.
 */
@Serializable
data class Quote(val zone: Int, val start: String, val end: String, val total: Long, val paidMinutes: Int)

@Serializable
internal data class NearbyResponse(val zones: List<Zone>)
