package ru.khudob1n.krasnodar.parking.platform

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberUpdatedState
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.useContents
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import platform.CoreLocation.CLLocation
import platform.CoreLocation.CLLocationManager
import platform.CoreLocation.CLLocationManagerDelegateProtocol
import platform.CoreLocation.kCLAuthorizationStatusAuthorizedAlways
import platform.CoreLocation.kCLAuthorizationStatusAuthorizedWhenInUse
import platform.CoreLocation.kCLAuthorizationStatusNotDetermined
import platform.CoreLocation.kCLLocationAccuracyBest
import platform.Foundation.NSDate
import platform.Foundation.NSError
import platform.Foundation.timeIntervalSinceDate
import platform.darwin.NSObject
import ru.khudob1n.krasnodar.parking.data.LatLng

private fun authorized(manager: CLLocationManager) = manager.authorizationStatus.let {
    it == kCLAuthorizationStatusAuthorizedWhenInUse || it == kCLAuthorizationStatusAuthorizedAlways
}

actual fun hasLocationPermission(): Boolean = authorized(CLLocationManager())

@OptIn(ExperimentalForeignApi::class)
private fun CLLocation.point() = coordinate.useContents { LatLng(latitude, longitude) }

@OptIn(ExperimentalForeignApi::class)
actual fun locationUpdates(): Flow<LatLng> = callbackFlow {
    val manager = CLLocationManager()
    val delegate = object : NSObject(), CLLocationManagerDelegateProtocol {
        override fun locationManager(manager: CLLocationManager, didUpdateLocations: List<*>) {
            (didUpdateLocations.lastOrNull() as? CLLocation)?.let { trySend(it.point()) }
        }

        override fun locationManager(manager: CLLocationManager, didFailWithError: NSError) {
            if (!CLLocationManager.locationServicesEnabled()) close(IllegalStateException("Геолокация выключена"))
        }
    }
    manager.delegate = delegate
    manager.desiredAccuracy = kCLLocationAccuracyBest
    manager.distanceFilter = 5.0
    manager.location?.takeIf { NSDate().timeIntervalSinceDate(it.timestamp) < 60 }?.let { trySend(it.point()) }
    if (!CLLocationManager.locationServicesEnabled()) close(IllegalStateException("Геолокация выключена"))
    else manager.startUpdatingLocation()
    awaitClose {
        manager.stopUpdatingLocation()
        manager.delegate = null
    }
}

@Composable
actual fun rememberLocationPermissionRequest(onResult: (Boolean) -> Unit): () -> Unit {
    val result = rememberUpdatedState(onResult)
    val state = remember {
        object {
            val manager = CLLocationManager()
            var waiting = false
            val delegate = object : NSObject(), CLLocationManagerDelegateProtocol {
                override fun locationManagerDidChangeAuthorization(manager: CLLocationManager) {
                    if (!waiting || manager.authorizationStatus == kCLAuthorizationStatusNotDetermined) return
                    waiting = false
                    result.value(authorized(manager))
                }
            }.also { manager.delegate = it }
        }
    }
    return {
        if (state.manager.authorizationStatus == kCLAuthorizationStatusNotDetermined) {
            state.waiting = true
            state.manager.requestWhenInUseAuthorization()
        } else {
            result.value(authorized(state.manager))
        }
    }
}
