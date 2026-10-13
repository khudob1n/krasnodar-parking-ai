package ru.khudob1n.krasnodar.parking.data

/** Рубли из копеек: 6000 -> "60 ₽", 4550 -> "45,50 ₽". */
fun formatRub(kopecks: Long): String {
    val rub = kopecks / 100
    val kop = kopecks % 100
    return if (kop == 0L) "$rub ₽" else "$rub,${kop.toString().padStart(2, '0')} ₽"
}

/** Цена часа для карточки: «60 ₽/ч» или «Бесплатно». */
fun Zone.priceLabel(): String =
    if (kind == ZoneKind.Free || pricePerHour <= 0) "Бесплатно" else "${formatRub(pricePerHour)}/ч"

/** Когда платно: «круглосуточно» или «платно 08:00–20:00»; у бесплатных - пусто. */
fun Zone.paidHoursLabel(): String = when {
    kind == ZoneKind.Free || pricePerHour <= 0 -> ""
    paidFrom.isEmpty() || paidFrom == paidTo -> "круглосуточно"
    else -> "платно $paidFrom–$paidTo"
}

/** Расстояние пешком: «350 м», «1,2 км». */
fun formatDistance(meters: Int): String = when {
    meters < 1000 -> "$meters м"
    else -> {
        val tenths = (meters + 50) / 100
        if (tenths % 10 == 0) "${tenths / 10} км" else "${tenths / 10},${tenths % 10} км"
    }
}

/** Длительность: «15 мин», «1 ч», «1 ч 30 мин». */
fun formatMinutes(minutes: Int): String {
    val h = minutes / 60
    val m = minutes % 60
    return when {
        h == 0 -> "$m мин"
        m == 0 -> "$h ч"
        else -> "$h ч $m мин"
    }
}

/** Время «ЧЧ:ММ» из строки RFC 3339 бэкенда (он отдаёт время Краснодара): "2026-10-13T18:30:00+03:00" -> "18:30". */
fun clockOf(rfc3339: String): String = rfc3339.substringAfter('T').take(5)

/** Номер зоны из того, что ввёл человек: «1101», « 1101 », «№1101». Не число - null. */
fun parseZoneNumber(input: String): Int? = input.trim().removePrefix("№").trim().toIntOrNull()?.takeIf { it > 0 }
