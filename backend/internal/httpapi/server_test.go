package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

type fakeStore map[int]zones.Zone

func (f fakeStore) Get(_ context.Context, number int) (zones.Zone, error) {
	z, ok := f[number]
	if !ok {
		return zones.Zone{}, zones.ErrNotFound
	}
	return z, nil
}

func (f fakeStore) Nearby(_ context.Context, q zones.NearbyQuery) ([]zones.NearbyZone, error) {
	list := []zones.NearbyZone{}
	for _, z := range f {
		list = append(list, zones.NearbyZone{Zone: z, DistanceMeters: q.RadiusMeters / 2})
	}
	return list, nil
}

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestHandler(t *testing.T) {
	store := fakeStore{1103: {Number: 1103, Kind: zones.KindStreet, PricePerHour: 6000, PaidFrom: "08:00", PaidTo: "19:00"}}
	now = func() time.Time { return time.Date(2026, 10, 13, 18, 30, 0, 0, pricing.Krasnodar) }
	t.Cleanup(func() { now = time.Now })
	h := NewHandler(Deps{DB: fakePinger{}, Zones: store, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/v1/zones/1103", http.StatusOK, `"number":1103`},
		{"/api/v1/zones/1", http.StatusNotFound, `zone not found`},
		{"/api/v1/zones/abc", http.StatusBadRequest, `must be an integer`},
		{"/healthz", http.StatusOK, `"ok"`},
		{"/api/v1/zones/nearby?lat=45.02&lon=38.97", http.StatusOK, `"distanceMeters":250`},
		{"/api/v1/zones/nearby?lat=45.02&lon=38.97&radius=100&kind=street,lot", http.StatusOK, `"distanceMeters":50`},
		{"/api/v1/zones/nearby?lon=38.97", http.StatusBadRequest, `lat is required`},
		{"/api/v1/zones/nearby?lat=95&lon=38.97", http.StatusBadRequest, `lat must be between`},
		{"/api/v1/zones/nearby?lat=45&lon=38&radius=99999", http.StatusBadRequest, `radius must be between`},
		{"/api/v1/zones/nearby?lat=45&lon=38&kind=paid", http.StatusBadRequest, `unknown kind`},
		{"/api/v1/zones/1103/price?minutes=60&start=2026-10-13T10:00:00%2B03:00", http.StatusOK, `"total":6000`},
		{"/api/v1/zones/1103/price?minutes=60&start=2026-10-13T07:00:00Z", http.StatusOK, `"total":6000`},
		{"/api/v1/zones/1103/price?minutes=60", http.StatusOK, `"total":3000,"paidMinutes":30`},
		{"/api/v1/zones/1103/price", http.StatusBadRequest, `minutes is required`},
		{"/api/v1/zones/1103/price?minutes=60&start=завтра", http.StatusBadRequest, `start must be RFC 3339`},
		{"/api/v1/zones/1/price?minutes=60", http.StatusNotFound, `zone not found`},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
			if rec.Code != c.wantStatus {
				t.Errorf("status: want %d, got %d", c.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), c.wantBody) {
				t.Errorf("body %q does not contain %q", rec.Body.String(), c.wantBody)
			}
		})
	}
}
