package ru.khudob1n.krasnodar.parking.platform

import androidx.compose.runtime.Composable
import kotlinx.coroutines.flow.Flow
import ru.khudob1n.krasnodar.parking.data.LatLng

/** Есть ли разрешение на геолокацию. */
expect fun hasLocationPermission(): Boolean

/**
 * Положение пользователя, пока на него подписаны: сначала
 * последнее известное, если свежее минуты. Геолокация выключена - поток завершается ошибкой.
 */
expect fun locationUpdates(): Flow<LatLng>

/** Запрос разрешения на геолокацию; [onResult] - дали или нет. Возвращает функцию «спросить». */
@Composable
expect fun rememberLocationPermissionRequest(onResult: (Boolean) -> Unit): () -> Unit
