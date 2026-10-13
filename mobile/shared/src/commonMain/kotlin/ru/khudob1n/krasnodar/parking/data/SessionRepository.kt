package ru.khudob1n.krasnodar.parking.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Парковки пользователя: последние с сервера, активные - сверху. */
class SessionRepository(private val api: ApiClient, private val auth: AuthRepository) {
    private val state = MutableStateFlow<List<ParkingSession>>(emptyList())
    val sessions: StateFlow<List<ParkingSession>> = state.asStateFlow()

    suspend fun refresh() {
        state.value = auth.withToken { api.sessions(it) }
    }

    suspend fun start(zone: Int, vehicleId: Long, minutes: Int, cardId: Long, mode: PaymentMode): ParkingSession =
        auth.withToken { api.startSession(it, zone, vehicleId, minutes, cardId, mode) }.also(::put)

    suspend fun pay(id: Long, cardId: Long): ParkingSession =
        auth.withToken { api.paySession(it, id, cardId) }.also(::put)

    suspend fun extend(id: Long, minutes: Int): ParkingSession =
        auth.withToken { api.extendSession(it, id, minutes) }.also(::put)

    suspend fun stop(id: Long): ParkingSession =
        auth.withToken { api.stopSession(it, id) }.also(::put)

    fun clear() {
        state.value = emptyList()
    }

    /** Заменить сессию по id или добавить новую в начало. */
    private fun put(session: ParkingSession) {
        val list = state.value
        state.value = if (list.any { it.id == session.id }) list.map { if (it.id == session.id) session else it } else listOf(session) + list
    }
}
