package httpapi

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

const (
	defaultRadius = 500  // метров
	maxRadius     = 5000 // дальше искать парковку пешком бессмысленно
	defaultLimit  = 20
	maxLimit      = 100
)

// parseNearbyQuery разбирает ?lat=45.03&lon=38.97&radius=500&limit=20&kind=street,lot
func parseNearbyQuery(v url.Values) (zones.NearbyQuery, error) {
	q := zones.NearbyQuery{RadiusMeters: defaultRadius, Limit: defaultLimit}

	var err error
	if q.Point.Lat, err = parseFloat(v, "lat", -90, 90); err != nil {
		return q, err
	}
	if q.Point.Lon, err = parseFloat(v, "lon", -180, 180); err != nil {
		return q, err
	}
	if q.RadiusMeters, err = parseIntOr(v, "radius", defaultRadius, 1, maxRadius); err != nil {
		return q, err
	}
	if q.Limit, err = parseIntOr(v, "limit", defaultLimit, 1, maxLimit); err != nil {
		return q, err
	}

	if raw := v.Get("kind"); raw != "" {
		for _, k := range strings.Split(raw, ",") {
			kind := zones.Kind(strings.TrimSpace(k))
			switch kind {
			case zones.KindStreet, zones.KindLot, zones.KindFree:
				q.Kinds = append(q.Kinds, kind)
			default:
				return q, fmt.Errorf("unknown kind %q", k)
			}
		}
	}
	return q, nil
}

func parseFloat(v url.Values, name string, lo, hi float64) (float64, error) {
	raw := v.Get(name)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number", name)
	}
	if f < lo || f > hi {
		return 0, fmt.Errorf("%s must be between %v and %v", name, lo, hi)
	}
	return f, nil
}

func parseIntOr(v url.Values, name string, fallback, lo, hi int) (int, error) {
	raw := v.Get(name)
	if raw == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(name + " must be an integer")
	}
	if n < lo || n > hi {
		return 0, fmt.Errorf("%s must be between %d and %d", name, lo, hi)
	}
	return n, nil
}
