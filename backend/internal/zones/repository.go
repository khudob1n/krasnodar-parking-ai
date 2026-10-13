package zones

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound - зоны с таким номером нет.
var ErrNotFound = errors.New("zone not found")

// Repository читает и пишет зоны в Postgres.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const upsertSQL = `
INSERT INTO zones (
    number, kind, address, description, location, segments,
    price_per_hour, paid_from, paid_to,
    total_places, disabled_places, in_building, tariff_text, updated_at
) VALUES (
    $1, $2, $3, $4,
    ST_SetSRID(ST_MakePoint($5, $6), 4326)::geography,
    ST_GeogFromText($7),
    $8, $9::time, $10::time,
    $11, $12, $13, $14, now()
)
ON CONFLICT (number) DO UPDATE SET
    kind = EXCLUDED.kind,
    address = EXCLUDED.address,
    description = EXCLUDED.description,
    location = EXCLUDED.location,
    segments = EXCLUDED.segments,
    price_per_hour = EXCLUDED.price_per_hour,
    paid_from = EXCLUDED.paid_from,
    paid_to = EXCLUDED.paid_to,
    total_places = EXCLUDED.total_places,
    disabled_places = EXCLUDED.disabled_places,
    in_building = EXCLUDED.in_building,
    tariff_text = EXCLUDED.tariff_text,
    updated_at = now()`

// Upsert добавляет зоны или обновляет уже существующие с тем же номером.
// Всё выполняется в одной транзакции: либо загрузятся все зоны, либо ни одной.
func (r *Repository) Upsert(ctx context.Context, zones []Zone) error {
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		batch := &pgx.Batch{}
		for _, z := range zones {
			batch.Queue(upsertSQL,
				z.Number, z.Kind, z.Address, z.Description,
				z.Location.Lon, z.Location.Lat,
				segmentsWKT(z.Segments),
				z.PricePerHour, z.PaidFrom, z.PaidTo,
				z.TotalPlaces, z.DisabledPlaces, z.InBuilding, z.TariffText,
			)
		}
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("upsert zones: %w", err)
		}
		return nil
	})
}

// zoneColumns - поля зоны в том порядке, в котором их читает scanZone.
const zoneColumns = `
    number, kind, address, description,
    ST_Y(location::geometry), ST_X(location::geometry),
    COALESCE(ST_AsText(segments), ''),
    price_per_hour, to_char(paid_from, 'HH24:MI'), to_char(paid_to, 'HH24:MI'),
    total_places, disabled_places, in_building, tariff_text`

// scanZone читает одну зону из строки результата. extra - дополнительные
// колонки после zoneColumns (например, расстояние).
func scanZone(row pgx.Row, extra ...any) (Zone, error) {
	var z Zone
	var wkt string
	dest := []any{
		&z.Number, &z.Kind, &z.Address, &z.Description,
		&z.Location.Lat, &z.Location.Lon,
		&wkt,
		&z.PricePerHour, &z.PaidFrom, &z.PaidTo,
		&z.TotalPlaces, &z.DisabledPlaces, &z.InBuilding, &z.TariffText,
	}
	if err := row.Scan(append(dest, extra...)...); err != nil {
		return Zone{}, err
	}
	segments, err := parseSegmentsWKT(wkt)
	if err != nil {
		return Zone{}, fmt.Errorf("zone %d segments: %w", z.Number, err)
	}
	z.Segments = segments
	return z, nil
}

// Get возвращает зону по номеру с таблички.
func (r *Repository) Get(ctx context.Context, number int) (Zone, error) {
	q := `SELECT` + zoneColumns + ` FROM zones WHERE number = $1`
	z, err := scanZone(r.pool.QueryRow(ctx, q, number))
	if errors.Is(err, pgx.ErrNoRows) {
		return Zone{}, ErrNotFound
	}
	if err != nil {
		return Zone{}, fmt.Errorf("get zone %d: %w", number, err)
	}
	return z, nil
}

// NearbyQuery - параметры поиска парковок рядом с точкой.
type NearbyQuery struct {
	Point        Point
	RadiusMeters int
	Limit        int
	Kinds        []Kind // пусто - любые типы
}

// NearbyZone - зона и расстояние до неё.
type NearbyZone struct {
	Zone
	DistanceMeters int `json:"distanceMeters"`
}

// Nearby ищет зоны в радиусе от точки, ближайшие первыми. Расстояние
// считается до ближайшего участка улицы, а если участков нет - до точки зоны.
func (r *Repository) Nearby(ctx context.Context, q NearbyQuery) ([]NearbyZone, error) {
	kinds := make([]string, len(q.Kinds))
	for i, k := range q.Kinds {
		kinds[i] = string(k)
	}

	sql := `
WITH me AS (SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS p)
SELECT` + zoneColumns + `, round(ST_Distance(z.shape, me.p))::int AS distance
FROM zones z, me
WHERE ST_DWithin(z.shape, me.p, $3)
  AND (cardinality($4::text[]) = 0 OR z.kind = ANY($4))
ORDER BY distance, z.number
LIMIT $5`

	rows, err := r.pool.Query(ctx, sql, q.Point.Lon, q.Point.Lat, q.RadiusMeters, kinds, q.Limit)
	if err != nil {
		return nil, fmt.Errorf("nearby zones: %w", err)
	}
	defer rows.Close()

	result := []NearbyZone{}
	for rows.Next() {
		var n NearbyZone
		if n.Zone, err = scanZone(rows, &n.DistanceMeters); err != nil {
			return nil, fmt.Errorf("nearby zones: %w", err)
		}
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("nearby zones: %w", err)
	}
	return result, nil
}

// segmentsWKT записывает участки в текстовый формат WKT, который понимает
// PostGIS: MULTILINESTRING((lon lat, lon lat), ...). Порядок именно такой:
// сначала долгота, потом широта. Без участков возвращает nil, то есть NULL.
func segmentsWKT(segments []Segment) *string {
	if len(segments) == 0 {
		return nil
	}
	parts := make([]string, len(segments))
	for i, s := range segments {
		parts[i] = fmt.Sprintf("(%s %s, %s %s)",
			fmtCoord(s.Start.Lon), fmtCoord(s.Start.Lat),
			fmtCoord(s.Stop.Lon), fmtCoord(s.Stop.Lat))
	}
	wkt := "MULTILINESTRING(" + strings.Join(parts, ", ") + ")"
	return &wkt
}

func fmtCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// parseSegmentsWKT разбирает ответ ST_AsText обратно в участки.
// Каждый наш участок - линия ровно из двух точек.
func parseSegmentsWKT(wkt string) ([]Segment, error) {
	segments := []Segment{}
	if wkt == "" {
		return segments, nil
	}
	body, ok := strings.CutPrefix(wkt, "MULTILINESTRING(")
	if !ok {
		return nil, fmt.Errorf("unexpected wkt %q", wkt)
	}
	body = strings.TrimSuffix(body, ")")
	for _, line := range strings.Split(body, "),(") {
		line = strings.Trim(line, "()")
		var s Segment
		_, err := fmt.Sscanf(line, "%g %g,%g %g", &s.Start.Lon, &s.Start.Lat, &s.Stop.Lon, &s.Stop.Lat)
		if err != nil {
			return nil, fmt.Errorf("parse line %q: %w", line, err)
		}
		segments = append(segments, s)
	}
	return segments, nil
}
