package ru.khudob1n.krasnodar.parking.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * Карты пользователя. Новую карту сначала отдаём платёжному провайдеру (он вернёт токен),
 * и только токен - нашему серверу: номер карты к нам не попадает.
 */
class CardRepository(private val api: ApiClient, private val auth: AuthRepository) {
    private val state = MutableStateFlow<List<PaymentCard>>(emptyList())
    val cards: StateFlow<List<PaymentCard>> = state.asStateFlow()

    suspend fun refresh() {
        state.value = auth.withToken { api.cards(it) }
    }

    suspend fun add(number: String, expMonth: Int, expYear: Int, cvc: String): PaymentCard {
        val token = api.tokenizeCard(number, expMonth, expYear, cvc)
        val card = auth.withToken { api.addCard(it, token) }
        state.value = state.value + card
        return card
    }

    suspend fun delete(id: Long) {
        auth.withToken { api.deleteCard(it, id) }
        state.value = state.value.filterNot { it.id == id }
    }

    fun clear() {
        state.value = emptyList()
    }
}
