package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.Serializable

/** Машина пользователя: госномер в виде «А123ВС93» и необязательное название («Рабочая»). */
@Serializable
data class Vehicle(val id: Long, val plate: String, val title: String = "")

@Serializable
internal data class VehiclesResponse(val vehicles: List<Vehicle>)

/** Буквы, одинаковые в кириллице и латинице, - только они бывают в российских номерах. */
private val LATIN_TO_CYRILLIC = mapOf(
    'A' to 'А', 'B' to 'В', 'E' to 'Е', 'K' to 'К', 'M' to 'М', 'H' to 'Н',
    'O' to 'О', 'P' to 'Р', 'C' to 'С', 'T' to 'Т', 'Y' to 'У', 'X' to 'Х',
)
private val PLATE = Regex("^[АВЕКМНОРСТУХ]\\d{3}[АВЕКМНОРСТУХ]{2}\\d{2,3}$")

/**
 * Номер, как его хранит бэкенд (NormalizePlate в backend/internal/vehicles/plate.go):
 * «а 123 bc 93» -> «А123ВС93». Не похоже на номер - null.
 */
fun normalizePlate(input: String): String? {
    val plate = input.uppercase().filter { it != ' ' && it != '-' && it != '|' }.map { LATIN_TO_CYRILLIC[it] ?: it }.joinToString("")
    return plate.takeIf { PLATE.matches(it) }
}

/** «А123ВС93» -> «А 123 ВС 93», как на самом номере. */
fun formatPlate(plate: String): String =
    if (plate.length < 8) plate else "${plate[0]} ${plate.substring(1, 4)} ${plate.substring(4, 6)} ${plate.substring(6)}"
