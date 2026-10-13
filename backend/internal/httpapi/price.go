package httpapi

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

// now вынесен в переменную, чтобы тесты могли «остановить часы».
var now = time.Now

const maxParkingMinutes = 7 * 24 * 60 // неделя

type priceResponse struct {
	Zone  int       `json:"zone"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	pricing.Quote
}

// parsePriceInterval разбирает ?minutes=60&start=2026-10-13T10:00:00+03:00.
// Без start считаем от текущего момента.
func parsePriceInterval(v url.Values, current time.Time) (time.Time, time.Time, error) {
	minutes, err := parseIntOr(v, "minutes", 0, 1, maxParkingMinutes)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if minutes == 0 {
		return time.Time{}, time.Time{}, errors.New("minutes is required")
	}

	start := current
	if raw := v.Get("start"); raw != "" {
		if start, err = time.Parse(time.RFC3339, raw); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("start must be RFC 3339, e.g. 2026-10-13T10:00:00+03:00")
		}
	}
	start = start.In(pricing.Krasnodar).Truncate(time.Second)
	return start, start.Add(time.Duration(minutes) * time.Minute), nil
}
