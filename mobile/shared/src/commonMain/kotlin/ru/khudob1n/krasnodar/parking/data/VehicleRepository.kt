package ru.khudob1n.krasnodar.parking.data

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/** Машины пользователя: список держим в памяти, чтобы оплата открывалась сразу с выбранной машиной. */
class VehicleRepository(private val api: ApiClient, private val auth: AuthRepository) {
    private val state = MutableStateFlow<List<Vehicle>>(emptyList())
    val vehicles: StateFlow<List<Vehicle>> = state.asStateFlow()

    suspend fun refresh() {
        state.value = auth.withToken { api.vehicles(it) }
    }

    suspend fun add(plate: String, title: String): Vehicle {
        val vehicle = auth.withToken { api.addVehicle(it, plate, title) }
        state.value = state.value + vehicle
        return vehicle
    }

    suspend fun delete(id: Long) {
        auth.withToken { api.deleteVehicle(it, id) }
        state.value = state.value.filterNot { it.id == id }
    }

    /** После выхода чужие машины в памяти не нужны. */
    fun clear() {
        state.value = emptyList()
    }
}
