package ru.khudob1n.krasnodar.parking.data

import kotlinx.serialization.Serializable

@Serializable
data class User(val id: Long, val phone: String)

/** Ответ входа и обновления токенов (backend/internal/auth). */
@Serializable
data class AuthSession(val accessToken: String, val refreshToken: String, val expiresIn: Int, val user: User)

/** Чем закончился запрос кода. */
sealed interface CodeResult {
    data object Sent : CodeResult

    /** Код уже отправляли - новый можно запросить через [seconds] секунд. */
    data class TooSoon(val seconds: Int) : CodeResult

    data class Failed(val message: String) : CodeResult
}

/** Цифр в коде из звонка (codeDigits в backend/internal/auth/service.go). */
const val CODE_DIGITS = 4

/**
 * Телефон в виде, который ждёт бэкенд, +79XXXXXXXXX, из того, что ввёл человек:
 * «8 918 123-45-67», «+7 (918) 1234567», «9181234567». Не российский мобильный - null.
 */
fun normalizePhone(input: String): String? {
    var digits = input.filter { it.isDigit() }
    if (digits.length == 11 && (digits[0] == '8' || digits[0] == '7')) digits = digits.substring(1)
    if (digits.length != 10 || digits[0] != '9') return null
    return "+7$digits"
}

/** «+79181234567» -> «+7 918 123-45-67». */
fun formatPhone(phone: String): String {
    val d = phone.removePrefix("+7")
    if (d.length != 10) return phone
    return "+7 ${d.substring(0, 3)} ${d.substring(3, 6)}-${d.substring(6, 8)}-${d.substring(8)}"
}
