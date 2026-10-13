package ru.khudob1n.krasnodar.parking.map

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.em
import androidx.compose.ui.unit.sp
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import org.maplibre.compose.expressions.dsl.asString
import org.maplibre.compose.expressions.dsl.const
import org.maplibre.compose.expressions.dsl.convertToColor
import org.maplibre.compose.expressions.dsl.feature
import org.maplibre.compose.expressions.dsl.interpolate
import org.maplibre.compose.expressions.dsl.linear
import org.maplibre.compose.expressions.dsl.textOffset
import org.maplibre.compose.expressions.dsl.zoom
import org.maplibre.compose.expressions.value.LineCap
import org.maplibre.compose.expressions.value.LineJoin
import org.maplibre.compose.interaction.ClickResult
import org.maplibre.compose.layers.CircleLayer
import org.maplibre.compose.layers.LineLayer
import org.maplibre.compose.layers.SymbolLayer
import org.maplibre.compose.sources.GeoJsonData
import org.maplibre.compose.sources.rememberGeoJsonSource
import org.maplibre.spatialk.geojson.Feature
import org.maplibre.spatialk.geojson.FeatureCollection
import org.maplibre.spatialk.geojson.Geometry
import org.maplibre.spatialk.geojson.LineString
import org.maplibre.spatialk.geojson.Point
import org.maplibre.spatialk.geojson.Position
import ru.khudob1n.krasnodar.parking.data.LatLng
import ru.khudob1n.krasnodar.parking.data.Zone
import ru.khudob1n.krasnodar.parking.data.ZoneKind

private fun geoFeature(geometry: Geometry, properties: JsonObject): Feature<Geometry, JsonObject> = Feature(geometry, properties)

private fun LatLng.position() = Position(longitude = lon, latitude = lat)

/** "#RRGGBB" для выражений стиля. */
private fun Color.hex(): String {
    val argb = (alpha * 255).toInt() shl 24 or ((red * 255).toInt() shl 16) or ((green * 255).toInt() shl 8) or (blue * 255).toInt()
    return "#" + (argb and 0xFFFFFF).toString(16).padStart(6, '0').uppercase()
}

/**
 * Зоны на карте: уличные - линиями вдоль участков улиц, площадки и бесплатные - кружками в точке
 * зоны. Платные - синие, бесплатные - зелёные; выбранная зона - толще и с обводкой. Вблизи
 * у каждой зоны подписан номер - тот же, что на табличке у парковки.
 */
@Composable
fun ZoneLayers(
    zones: List<Zone>,
    selected: Zone?,
    user: LatLng?,
    paid: Color,
    free: Color,
    background: Color,
    onSelect: (Int) -> Unit,
) {
    val lines = remember(zones, paid, free) {
        val list = zones.flatMap { zone ->
            val props = buildJsonObject {
                put("number", zone.number.toString())
                put("color", (if (zone.kind == ZoneKind.Free) free else paid).hex())
            }
            zone.segments.map { seg -> geoFeature(LineString(listOf(seg.start.position(), seg.stop.position())), props) }
        }
        FeatureCollection(list)
    }
    // Зона без участков улиц (площадка или бесплатная) - кружок в её точке.
    val points = remember(zones, paid, free) {
        FeatureCollection(zones.filter { it.segments.isEmpty() }.map { zone ->
            geoFeature(Point(zone.location.position()), buildJsonObject {
                put("number", zone.number.toString())
                put("color", (if (zone.kind == ZoneKind.Free) free else paid).hex())
            })
        })
    }
    // Подпись с номером - в точке зоны, цветом зоны.
    val labels = remember(zones, paid, free) {
        FeatureCollection(zones.map { zone ->
            geoFeature(Point(zone.location.position()), buildJsonObject {
                put("number", zone.number.toString())
                put("color", (if (zone.kind == ZoneKind.Free) free else paid).hex())
            })
        })
    }
    val selectedZone = selected
    val selectedLines = remember(selectedZone) {
        FeatureCollection(selectedZone?.segments.orEmpty().map { seg -> geoFeature(LineString(listOf(seg.start.position(), seg.stop.position())), JsonObject(emptyMap())) })
    }
    val selectedPoint = remember(selectedZone) {
        FeatureCollection(listOfNotNull(selectedZone?.takeIf { it.segments.isEmpty() }?.let { geoFeature(Point(it.location.position()), JsonObject(emptyMap())) }))
    }
    val userPoint = remember(user) { FeatureCollection(listOfNotNull(user?.let { geoFeature(Point(it.position()), JsonObject(emptyMap())) })) }

    val open: (List<Feature<Geometry, JsonObject?>>) -> ClickResult = { hits ->
        hits.firstNotNullOfOrNull { it.properties?.get("number")?.jsonPrimitive?.content?.toIntOrNull() }
            ?.let { onSelect(it); ClickResult.Consume } ?: ClickResult.Pass
    }

    val lineSource = rememberGeoJsonSource(GeoJsonData.Features(lines))
    val pointSource = rememberGeoJsonSource(GeoJsonData.Features(points))
    val selectedLineSource = rememberGeoJsonSource(GeoJsonData.Features(selectedLines))
    val selectedPointSource = rememberGeoJsonSource(GeoJsonData.Features(selectedPoint))

    // Подсветка выбранной зоны - под самой зоной, широкой полупрозрачной полосой.
    LineLayer(
        "zone-selected-halo", selectedLineSource, color = const(paid), opacity = const(0.3f), width = const(16.dp),
        cap = const(LineCap.Round), join = const(LineJoin.Round),
    )
    LineLayer(
        "zone-lines", lineSource, color = feature["color"].asString().convertToColor(),
        // Издалека тонкие, чтобы город не превращался в сплошную синеву; вблизи - толще, в них легко попасть.
        width = interpolate(linear(), zoom(), 11 to const(1.5.dp), 14 to const(3.dp), 17 to const(6.dp)),
        cap = const(LineCap.Round), join = const(LineJoin.Round), onClick = open, hitPadding = 12.dp,
    )
    CircleLayer(
        "zone-selected-point", selectedPointSource, color = const(paid), opacity = const(0.3f), radius = const(18.dp),
    )
    CircleLayer(
        "zone-points", pointSource, color = feature["color"].asString().convertToColor(),
        radius = interpolate(linear(), zoom(), 11 to const(3.dp), 14 to const(5.dp), 17 to const(8.dp)),
        strokeColor = const(background), strokeWidth = const(2.dp), onClick = open, hitPadding = 10.dp,
    )

    SymbolLayer(
        "zone-labels", rememberGeoJsonSource(GeoJsonData.Features(labels)), minZoom = ZONE_LABELS_MIN_ZOOM,
        textField = feature["number"].asString(),
        textFont = const(listOf("JetBrains Mono Regular")),
        textSize = const(13.sp),
        textColor = feature["color"].asString().convertToColor(),
        textHaloColor = const(background), textHaloWidth = const(2.dp),
        textOffset = textOffset(0f.em, (-1.2f).em),
        onClick = open, hitPadding = 8.dp,
    )

    // «Я здесь»: синяя точка с белой обводкой.
    CircleLayer(
        "user-location", rememberGeoJsonSource(GeoJsonData.Features(userPoint)),
        color = const(Color(0xFF1A73E8)), radius = const(7.dp), strokeColor = const(Color.White), strokeWidth = const(3.dp),
    )
}
