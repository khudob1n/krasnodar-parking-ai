package ru.khudob1n.krasnodar.parking.map

import ru.khudob1n.krasnodar.parking.data.LatLng

/** Центр и рамка карты - как у «Транспорта Краснодара». */
object City {
    val center = LatLng(45.0355, 38.9753)
    const val DEFAULT_ZOOM = 13.0

    const val NORTH = 45.35
    const val EAST = 39.45
    const val SOUTH = 44.85
    const val WEST = 38.6
}

/** С этого масштаба у зон подписаны номера: раньше подписи налезают друг на друга. */
const val ZONE_LABELS_MIN_ZOOM = 15f
