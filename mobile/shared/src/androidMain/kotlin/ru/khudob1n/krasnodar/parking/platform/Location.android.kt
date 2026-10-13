package ru.khudob1n.krasnodar.parking.platform

import android.Manifest
import android.annotation.SuppressLint
import android.content.pm.PackageManager
import android.location.LocationListener
import android.location.LocationManager
import android.os.Looper
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.Composable
import androidx.core.content.ContextCompat
import androidx.core.location.LocationManagerCompat
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import ru.khudob1n.krasnodar.parking.data.LatLng

private val LOCATION_PERMISSIONS = arrayOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION)

actual fun hasLocationPermission(): Boolean = LOCATION_PERMISSIONS.any {
    ContextCompat.checkSelfPermission(Platform.appContext, it) == PackageManager.PERMISSION_GRANTED
}

/** GPS и сеть, что ответит. */
@SuppressLint("MissingPermission")
actual fun locationUpdates(): Flow<LatLng> = callbackFlow {
    val manager = Platform.appContext.getSystemService(LocationManager::class.java)
    val providers = listOf(LocationManager.GPS_PROVIDER, LocationManager.NETWORK_PROVIDER).filter { manager.isProviderEnabled(it) }
    providers.mapNotNull { manager.getLastKnownLocation(it) }
        .filter { System.currentTimeMillis() - it.time < 60_000 }
        .maxByOrNull { it.time }?.let { trySend(LatLng(it.latitude, it.longitude)) }
    val listener = LocationListener { trySend(LatLng(it.latitude, it.longitude)) }
    for (provider in providers) manager.requestLocationUpdates(provider, 2_000L, 5f, listener, Looper.getMainLooper())
    if (providers.isEmpty() || !LocationManagerCompat.isLocationEnabled(manager)) close(IllegalStateException("Геолокация выключена"))
    awaitClose { manager.removeUpdates(listener) }
}

@Composable
actual fun rememberLocationPermissionRequest(onResult: (Boolean) -> Unit): () -> Unit {
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { result ->
        onResult(result.values.any { it })
    }
    return { launcher.launch(LOCATION_PERMISSIONS) }
}
