package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

func TestGetZone(t *testing.T) {
	store := fakeStore{1103: {Number: 1103, Kind: zones.KindStreet, PricePerHour: 6000}}
	h := NewHandler(fakePinger{}, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	cases := []struct {
		path       string
		wantStatus int
		wantBody   string
	}{
		{"/api/v1/zones/1103", http.StatusOK, `"number":1103`},
		{"/api/v1/zones/1", http.StatusNotFound, `zone not found`},
		{"/api/v1/zones/abc", http.StatusBadRequest, `must be an integer`},
		{"/healthz", http.StatusOK, `"ok"`},
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
