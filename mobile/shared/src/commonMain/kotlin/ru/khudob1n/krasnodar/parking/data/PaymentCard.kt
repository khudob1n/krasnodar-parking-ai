package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.Serializable

/** Привязанная карта (GET /api/v1/cards). Номера карты у нас нет - только последние 4 цифры. */
@Serializable
data class PaymentCard(val id: Long, val last4: String, val brand: String, val expMonth: Int, val expYear: Int)

@Serializable
internal data class CardsResponse(val cards: List<PaymentCard>)

/** Ответ платёжного провайдера на данные карты: токен, по которому наш сервер её привяжет. */
@Serializable
internal data class CardToken(val token: String)

/** «Visa •• 4242». */
fun PaymentCard.label(): String = "${brandName(brand)} •• $last4"

/** «12/30». */
fun PaymentCard.expiry(): String = "${expMonth.toString().padStart(2, '0')}/${(expYear % 100).toString().padStart(2, '0')}"

fun brandName(brand: String): String = when (brand) {
    "visa" -> "Visa"
    "mastercard" -> "Mastercard"
    "mir" -> "Мир"
    else -> "Карта"
}

/** Номер карты без пробелов, если в нём 16 цифр и сходится контрольная сумма (алгоритм Луна); иначе null. */
fun normalizeCardNumber(input: String): String? {
    val digits = input.filter { it.isDigit() }
    if (digits.length != 16) return null
    val sum = digits.reversed().mapIndexed { i, ch ->
        val d = ch - '0'
        if (i % 2 == 1) (d * 2).let { if (it > 9) it - 9 else it } else d
    }.sum()
    return digits.takeIf { sum % 10 == 0 }
}

/** «4242424242424242» -> «4242 4242 4242 4242» (и для недописанного номера). */
fun formatCardNumber(input: String): String = input.filter { it.isDigit() }.take(16).chunked(4).joinToString(" ")

/** «1230» или «12/30» -> месяц и полный год; не похоже на срок - null. */
fun parseExpiry(input: String): Pair<Int, Int>? {
    val digits = input.filter { it.isDigit() }
    if (digits.length != 4) return null
    val month = digits.take(2).toInt()
    val year = 2000 + digits.takeLast(2).toInt()
    return (month to year).takeIf { month in 1..12 }
}

/** Тестовые карты провайдера fakepay (backend/README.md): что с ними будет при оплате. */
val TEST_CARDS = listOf(
    "4242 4242 4242 4242" to "всё проходит",
    "4000 0000 0000 9995" to "не хватает денег",
    "4000 0000 0000 0002" to "банк отклоняет",
)
