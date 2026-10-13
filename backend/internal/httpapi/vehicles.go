package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/vehicles"
)

type VehicleStore interface {
	Add(ctx context.Context, userID int64, plate, title string) (vehicles.Vehicle, error)
	List(ctx context.Context, userID int64) ([]vehicles.Vehicle, error)
	Delete(ctx context.Context, userID, id int64) error
}

func registerVehicles(mux *http.ServeMux, svc AuthService, store VehicleStore, log *slog.Logger) {
	if svc == nil || store == nil {
		return
	}

	mux.Handle("GET /api/v1/vehicles", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		list, err := store.List(r.Context(), userID)
		if err != nil {
			log.Error("list vehicles", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"vehicles": list})
	}))

	mux.Handle("POST /api/v1/vehicles", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		var req struct {
			Plate string `json:"plate"`
			Title string `json:"title"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		v, err := store.Add(r.Context(), userID, req.Plate, req.Title)
		switch {
		case err == nil:
			writeJSON(w, http.StatusCreated, v)
		case errors.Is(err, vehicles.ErrDuplicate):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, vehicles.ErrBadPlate), errors.Is(err, vehicles.ErrBadTitle):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("add vehicle", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}))

	mux.Handle("DELETE /api/v1/vehicles/{id}", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id must be an integer")
			return
		}
		err = store.Delete(r.Context(), userID, id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, vehicles.ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			log.Error("delete vehicle", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}))
}
