// Package httpapi - HTTP-ручки API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// ZoneStore - то, что серверу нужно от хранилища зон. Интерфейс вместо
// конкретного типа позволяет подменить базу в тестах.
type ZoneStore interface {
	Get(ctx context.Context, number int) (zones.Zone, error)
	All(ctx context.Context) ([]zones.Zone, error)
	Nearby(ctx context.Context, q zones.NearbyQuery) ([]zones.NearbyZone, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

// Deps - всё, что нужно ручкам API.
type Deps struct {
	DB       Pinger
	Zones    ZoneStore
	Auth     AuthService
	Vehicles VehicleStore
	Sessions SessionService
	Devices  DeviceStore
	Cards    CardStore
	// FakePay - ручки тестового платёжного провайдера (/fakepay/...).
	// Живут на нашем сервере, но к нашему API не относятся.
	FakePay http.Handler
	Log     *slog.Logger
}

// NewHandler собирает все маршруты API.
func NewHandler(d Deps) http.Handler {
	db, store, log := d.DB, d.Zones, d.Log
	mux := http.NewServeMux()
	registerAuth(mux, d.Auth, log)
	registerVehicles(mux, d.Auth, d.Vehicles, log)
	registerSessions(mux, d.Auth, d.Sessions, log)
	registerDevices(mux, d.Auth, d.Devices, log)
	registerCards(mux, d.Auth, d.Cards, log)
	if d.FakePay != nil {
		mux.Handle("/fakepay/", d.FakePay)
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			log.Error("healthz: database unavailable", "err", err)
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("GET /api/v1/zones", &allZones{store: store, log: log})

	mux.HandleFunc("GET /api/v1/zones/{number}", func(w http.ResponseWriter, r *http.Request) {
		number, err := strconv.Atoi(r.PathValue("number"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "zone number must be an integer")
			return
		}
		z, err := store.Get(r.Context(), number)
		if errors.Is(err, zones.ErrNotFound) {
			writeError(w, http.StatusNotFound, "zone not found")
			return
		}
		if err != nil {
			log.Error("get zone", "number", number, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, z)
	})

	mux.HandleFunc("GET /api/v1/zones/{number}/price", func(w http.ResponseWriter, r *http.Request) {
		number, err := strconv.Atoi(r.PathValue("number"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "zone number must be an integer")
			return
		}
		start, end, err := parsePriceInterval(r.URL.Query(), now())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		z, err := store.Get(r.Context(), number)
		if errors.Is(err, zones.ErrNotFound) {
			writeError(w, http.StatusNotFound, "zone not found")
			return
		}
		if err != nil {
			log.Error("get zone", "number", number, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		quote, err := pricing.Calculate(pricing.Tariff{
			PricePerHour: z.PricePerHour, PaidFrom: z.PaidFrom, PaidTo: z.PaidTo,
		}, start, end)
		if err != nil {
			log.Error("calculate price", "number", number, "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, priceResponse{
			Zone: z.Number, Start: start, End: end, Quote: quote,
		})
	})

	mux.HandleFunc("GET /api/v1/zones/nearby", func(w http.ResponseWriter, r *http.Request) {
		q, err := parseNearbyQuery(r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		list, err := store.Nearby(r.Context(), q)
		if err != nil {
			log.Error("nearby zones", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"zones": list})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// readJSON читает тело запроса в v. Тело ограничено 64 КБ, неизвестные
// поля - ошибка: так опечатки в клиенте видны сразу.
func readJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.New("invalid JSON body")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
