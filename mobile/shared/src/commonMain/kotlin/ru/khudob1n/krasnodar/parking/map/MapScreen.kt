package ru.khudob1n.krasnodar.parking.map

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.safeDrawingPadding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeoutOrNull
import org.maplibre.compose.camera.CameraAnimation
import org.maplibre.compose.camera.CameraPosition
import org.maplibre.compose.camera.CameraUpdate
import org.maplibre.compose.interaction.ClickResult
import org.maplibre.compose.interaction.MapInteractions
import org.maplibre.compose.map.CameraConstraints
import org.maplibre.compose.map.MapEvent
import org.maplibre.compose.map.MapUiOptions
import org.maplibre.compose.map.MaplibreMap
import org.maplibre.compose.map.rememberMapState
import org.maplibre.compose.style.BaseStyle
import org.maplibre.spatialk.geojson.BoundingBox
import org.maplibre.spatialk.geojson.Position
import ru.khudob1n.krasnodar.parking.data.ApiClient
import ru.khudob1n.krasnodar.parking.data.AuthRepository
import ru.khudob1n.krasnodar.parking.data.SessionRepository
import ru.khudob1n.krasnodar.parking.data.VehicleRepository
import ru.khudob1n.krasnodar.parking.data.LatLng
import ru.khudob1n.krasnodar.parking.data.Zone
import ru.khudob1n.krasnodar.parking.data.ZoneKind
import ru.khudob1n.krasnodar.parking.data.formatDistance
import ru.khudob1n.krasnodar.parking.data.reminderIds
import ru.khudob1n.krasnodar.parking.data.reminders
import ru.khudob1n.krasnodar.parking.data.priceLabel
import ru.khudob1n.krasnodar.parking.platform.hasLocationPermission
import ru.khudob1n.krasnodar.parking.platform.Reminders
import ru.khudob1n.krasnodar.parking.platform.locationUpdates
import ru.khudob1n.krasnodar.parking.platform.rememberNotificationPermissionRequest
import ru.khudob1n.krasnodar.parking.platform.rememberLocationPermissionRequest
import ru.khudob1n.krasnodar.parking.ui.account.AccountSheet
import ru.khudob1n.krasnodar.parking.ui.parking.ActiveParkingCard
import ru.khudob1n.krasnodar.parking.ui.parking.ParkingActions
import ru.khudob1n.krasnodar.parking.ui.theme.AppTheme
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds

/** Сколько ближайших зон показывать списком внизу. */
private const val NEAREST_COUNT = 5
private const val ZONE_ZOOM = 17.0

/**
 * Главный экран: карта с зонами вокруг центра карты, кнопка «Где я» и список ближайших зон.
 * Зоны грузятся, когда карта остановилась (MapEvent.Idle), - в радиусе, который накрывает экран.
 */
@Composable
fun MapScreen(api: ApiClient, auth: AuthRepository, vehicles: VehicleRepository, sessions: SessionRepository) {
    val colors = AppTheme.colors
    val scope = rememberCoroutineScope()

    var zones by remember { mutableStateOf<List<Zone>>(emptyList()) }
    var selected by remember { mutableStateOf<Zone?>(null) }
    var user by remember { mutableStateOf<LatLng?>(null) }
    var error by remember { mutableStateOf<String?>(null) }
    var tooFar by remember { mutableStateOf(false) }
    var accountOpen by remember { mutableStateOf(false) }
    val session by auth.session.collectAsState()
    val parkings by sessions.sessions.collectAsState()
    // Парковка, которую показывает карточка: остановили - итог висит, пока карточку не закроют.
    var shownParking by remember { mutableStateOf<Long?>(null) }
    val askNotifications = rememberNotificationPermissionRequest()
    val parkingActions = remember {
        ParkingActions(
            auth, vehicles, sessions,
            openAccount = { accountOpen = true; selected = null },
            onStarted = { started -> selected = null; shownParking = started.id; askNotifications() },
        )
    }

    // Напоминания на устройстве: у идущих парковок - за 10 минут и в конце, у завершённых - отменить.
    LaunchedEffect(parkings) {
        parkings.forEach { p ->
            if (p.isActive) p.reminders().forEach(Reminders::schedule) else reminderIds(p.id).forEach(Reminders::cancel)
        }
    }

    // Вошёл - подтянуть машины и парковки; вышел - забыть их.
    LaunchedEffect(session?.user?.id) {
        if (session == null) {
            vehicles.clear()
            sessions.clear()
            return@LaunchedEffect
        }
        try {
            vehicles.refresh()
            sessions.refresh()
        } catch (e: CancellationException) {
            throw e
        } catch (_: Exception) {
            // Нет связи - покажем, что есть; карточки сами повторят запрос.
        }
    }

    val mapState = rememberMapState(
        baseStyle = BaseStyle.Json(if (colors.isDark) MapStyles.dark else MapStyles.light),
        initialCameraPosition = CameraPosition(target = Position(longitude = City.center.lon, latitude = City.center.lat), zoom = City.DEFAULT_ZOOM),
    ) {
        ZoneLayers(
            zones = zones, selected = selected, user = user,
            paid = colors.parkingPaid, free = colors.parkingFree, background = colors.backgroundPrimary,
            onSelect = { number -> selected = zones.firstOrNull { it.number == number }; accountOpen = false },
        )
    }

    fun flyTo(point: LatLng, zoom: Double) {
        scope.launch {
            runCatching {
                mapState.animateCamera(
                    CameraUpdate(target = Position(longitude = point.lon, latitude = point.lat), zoom = zoom),
                    CameraAnimation.Ease(700.milliseconds),
                )
            }
        }
    }

    // Карта остановилась - подгрузить зоны вокруг центра кадра.
    LaunchedEffect(mapState) {
        mapState.events.collect { event ->
            if (event != MapEvent.Idle) return@collect
            val camera = mapState.cameraPosition
            tooFar = camera.zoom < ZONES_MIN_ZOOM
            if (tooFar) return@collect
            val center = LatLng(camera.target.latitude, camera.target.longitude)
            try {
                zones = api.nearby(center, radiusForZoom(camera.zoom, center.lat))
                error = null
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                error = "Не удалось загрузить зоны: ${e.message ?: "нет связи с сервером"}"
            }
        }
    }

    suspend fun locate() {
        val point = withTimeoutOrNull(15.seconds) { runCatching { locationUpdates().first() }.getOrNull() }
        if (point == null) {
            error = "Не получилось определить, где вы. Проверьте, включена ли геолокация."
            return
        }
        user = point
        flyTo(point, 16.0)
    }

    val askPermission = rememberLocationPermissionRequest { granted ->
        if (granted) scope.launch { locate() } else error = "Без доступа к геолокации покажу зоны в центре города."
    }
    // Разрешение уже есть - сразу к пользователю: парковку ищут рядом с собой.
    LaunchedEffect(Unit) { if (hasLocationPermission()) locate() }

    Box(Modifier.fillMaxSize().background(colors.mapBg)) {
        MaplibreMap(
            modifier = Modifier.fillMaxSize(),
            state = mapState,
            cameraConstraints = CameraConstraints(minZoom = 10.0, boundingBox = BoundingBox(west = City.WEST, south = City.SOUTH, east = City.EAST, north = City.NORTH)),
            interactions = MapInteractions {
                camera {
                    rotate { enabled = false }
                    tilt { enabled = false }
                }
                callbacks {
                    click { onUnhandled { selected = null; ClickResult.Consume } }
                }
            },
            uiOptions = MapUiOptions { loadColor = colors.mapBg },
        )

        Column(Modifier.fillMaxSize().safeDrawingPadding().padding(12.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                ZoneSearch(Modifier.weight(1f)) { number ->
                    scope.launch {
                        try {
                            val found = api.zone(number)
                            if (found == null) {
                                error = "Зоны $number нет. Номер написан на табличке у парковки."
                                return@launch
                            }
                            error = null
                            if (zones.none { it.number == found.number }) zones = zones + found
                            selected = found
                            flyTo(found.location, ZONE_ZOOM)
                        } catch (e: CancellationException) {
                            throw e
                        } catch (_: Exception) {
                            error = "Не удалось найти зону: нет связи с сервером"
                        }
                    }
                }
                Spacer(Modifier.size(8.dp))
                MapButton(if (session == null) "Войти" else "Профиль") { accountOpen = !accountOpen; if (accountOpen) selected = null }
                Spacer(Modifier.size(8.dp))
                MapButton("Где я") { if (hasLocationPermission()) scope.launch { locate() } else askPermission() }
            }
            error?.let { Banner(it, Modifier.padding(top = 8.dp)) }
            Spacer(Modifier.weight(1f))
            val zone = selected
            val parking = parkings.firstOrNull { it.isActive } ?: parkings.firstOrNull { it.id == shownParking }
            when {
                accountOpen -> AccountSheet(auth, vehicles, onClose = { accountOpen = false })
                zone != null -> ZoneSheet(zone, api, parkingActions, onClose = { selected = null })
                parking != null -> {
                    LaunchedEffect(parking.id) { shownParking = parking.id }
                    ActiveParkingCard(parking, sessions, onClose = { shownParking = null })
                }
                tooFar -> Banner("Приблизьте карту, чтобы увидеть парковки")
                zones.isNotEmpty() -> NearestList(nearestTo(zones, user), nearUser = user != null, onPick = { picked ->
                    selected = picked
                    flyTo(picked.location, ZONE_ZOOM)
                })
            }
        }
    }
}

/**
 * Ближайшие зоны с расстоянием: к пользователю, если знаем, где он, иначе - к центру карты (как их
 * отсортировал сервер).
 */
private fun nearestTo(zones: List<Zone>, user: LatLng?): List<Pair<Zone, Int?>> =
    if (user == null) zones.take(NEAREST_COUNT).map { it to it.distanceMeters }
    else zones.map { it to distanceMeters(user, it) }.sortedBy { it.second }.take(NEAREST_COUNT)

/** До ближайшей точки зоны: концы участков улиц или точка площадки (сервер меряет точнее, но от центра карты). */
private fun distanceMeters(from: LatLng, zone: Zone): Int {
    val points = zone.segments.flatMap { listOf(it.start, it.stop) }.ifEmpty { listOf(zone.location) }
    return points.minOf { haversineMeters(from, it) }
}

@Composable
private fun MapButton(label: String, onClick: () -> Unit) {
    Box(
        Modifier.clip(AppTheme.shapes.mapButton).background(AppTheme.colors.backgroundPrimary).clickable(onClick = onClick)
            .padding(horizontal = 14.dp, vertical = 12.dp),
    ) {
        Text(label, style = AppTheme.type.button, color = AppTheme.colors.textPrimary)
    }
}

@Composable
private fun Banner(text: String, modifier: Modifier = Modifier) {
    Box(modifier.clip(AppTheme.shapes.pill).background(AppTheme.colors.backgroundPrimary).padding(12.dp)) {
        Text(text, style = AppTheme.type.small, color = AppTheme.colors.textSecondary)
    }
}

/** Цветная точка вида зоны: синяя - платная, зелёная - бесплатная. */
@Composable
private fun KindDot(zone: Zone) {
    val c = AppTheme.colors
    val free = zone.kind == ZoneKind.Free || zone.pricePerHour <= 0
    Box(Modifier.size(10.dp).background(if (free) c.parkingFree else c.parkingPaid, CircleShape))
}

@Composable
private fun NearestList(zones: List<Pair<Zone, Int?>>, nearUser: Boolean, onPick: (Zone) -> Unit) {
    val c = AppTheme.colors
    Column(
        Modifier.fillMaxWidth().widthIn(max = 560.dp).clip(AppTheme.shapes.card).background(c.backgroundPrimary).padding(vertical = 8.dp),
    ) {
        Text(if (nearUser) "Рядом с вами" else "У центра карты", style = AppTheme.type.caption, color = c.textPrimary, modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp))
        zones.forEach { (zone, distance) ->
            Row(
                Modifier.fillMaxWidth().clickable { onPick(zone) }.padding(horizontal = 16.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                KindDot(zone)
                Text(zone.number.toString(), style = AppTheme.type.routeNumber, color = c.textPrimary)
                Text(zone.address, style = AppTheme.type.small, color = c.textSecondary, maxLines = 1, overflow = TextOverflow.Ellipsis, modifier = Modifier.weight(1f))
                Text(zone.priceLabel(), style = AppTheme.type.small, color = c.textPrimary)
                distance?.let { Text(formatDistance(it), style = AppTheme.type.small, color = c.functional) }
            }
        }
    }
}
