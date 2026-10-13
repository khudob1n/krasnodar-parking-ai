// Package httpapi - HTTP-ручки API.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// ZoneStore - то, что серверу нужно от хранилища зон. Интерфейс вместо
// конкретного типа позволяет подменить базу в тестах.
type ZoneStore interface {
	Get(ctx context.Context, number int) (zones.Zone, error)
}

type Pinger interface {
	Ping(ctx context.Context) error
}

// NewHandler собирает все маршруты API.
func NewHandler(db Pinger, store ZoneStore, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(r.Context()); err != nil {
			log.Error("healthz: database unavailable", "err", err)
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

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

	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
