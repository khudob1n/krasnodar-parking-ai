package ru.khudob1n.krasnodar.parking

import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.remember
import ru.khudob1n.krasnodar.parking.data.ApiClient
import ru.khudob1n.krasnodar.parking.data.AuthRepository
import ru.khudob1n.krasnodar.parking.data.CardRepository
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.ZoneCatalog
import ru.khudob1n.krasnodar.parking.map.MapScreen
import ru.khudob1n.krasnodar.parking.platform.AppConfig
import ru.khudob1n.krasnodar.parking.platform.FileStore
import ru.khudob1n.krasnodar.parking.platform.createHttpClient
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme
import ru.khudob1n.krasnodar.parking.ui.theme.ParkingTheme

/** Файл с кэшем всех зон города. */
private const val ZONES_CACHE = "zones.json"

/** Корень интерфейса, общий для Android и iOS: приложение открывается сразу картой зон. */
@Composable
fun App(onDarkChange: (Boolean) -> Unit = {}) {
    ParkingTheme {
        val dark = AppTheme.colors.isDark
        LaunchedEffect(dark) { onDarkChange(dark) }
        val api = remember { ApiClient(createHttpClient(), AppConfig.apiBaseUrl) }
        val catalog = remember { ZoneCatalog(api, readCache = { FileStore.read(ZONES_CACHE) }, writeCache = { FileStore.write(ZONES_CACHE, it) }) }
        val auth = remember { AuthRepository(api) }
        val vehicles = remember { VehicleRepository(api, auth) }
        val cards = remember { CardRepository(api, auth) }
        val sessions = remember { SessionRepository(api, auth) }
        MapScreen(api, catalog, auth, vehicles, cards, sessions)
    }
}
