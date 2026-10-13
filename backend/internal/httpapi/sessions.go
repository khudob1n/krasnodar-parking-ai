package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/parking"
)

type SessionService interface {
	Start(ctx context.Context, userID int64, req parking.StartRequest) (parking.Session, error)
	Extend(ctx context.Context, userID, id int64, minutes int) (parking.Session, error)
	Stop(ctx context.Context, userID, id int64) (parking.Session, error)
	Get(ctx context.Context, userID, id int64) (parking.Session, error)
	List(ctx context.Context, userID int64, limit int) ([]parking.Session, error)
}

func registerSessions(mux *http.ServeMux, svc AuthService, sessions SessionService, log *slog.Logger) {
	if svc == nil || sessions == nil {
		return
	}

	// reply отвечает сессией или переводит ошибку в HTTP-код.
	reply := func(w http.ResponseWriter, okStatus int, s parking.Session, err error) {
		switch {
		case err == nil:
			writeJSON(w, okStatus, s)
		case errors.Is(err, parking.ErrNotFound), errors.Is(err, parking.ErrZoneNotFound),
			errors.Is(err, parking.ErrVehicleNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, parking.ErrAlreadyParked), errors.Is(err, parking.ErrNotActive):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, parking.ErrFreeZone), errors.Is(err, parking.ErrBadDuration),
			errors.Is(err, parking.ErrTooLong):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("parking session", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}

	// withID достаёт {id} из пути.
	withID := func(next func(http.ResponseWriter, *http.Request, int64, int64)) http.Handler {
		return requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
			id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
			if err != nil {
				writeError(w, http.StatusBadRequest, "id must be an integer")
				return
			}
			next(w, r, userID, id)
		})
	}

	mux.Handle("POST /api/v1/sessions", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		var req parking.StartRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s, err := sessions.Start(r.Context(), userID, req)
		reply(w, http.StatusCreated, s, err)
	}))

	mux.Handle("GET /api/v1/sessions", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		limit, err := parseIntOr(r.URL.Query(), "limit", 20, 1, 100)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		list, err := sessions.List(r.Context(), userID, limit)
		if err != nil {
			log.Error("list sessions", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list})
	}))

	mux.Handle("GET /api/v1/sessions/{id}", withID(func(w http.ResponseWriter, r *http.Request, userID, id int64) {
		s, err := sessions.Get(r.Context(), userID, id)
		reply(w, http.StatusOK, s, err)
	}))

	mux.Handle("POST /api/v1/sessions/{id}/extend", withID(func(w http.ResponseWriter, r *http.Request, userID, id int64) {
		var req struct {
			Minutes int `json:"minutes"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s, err := sessions.Extend(r.Context(), userID, id, req.Minutes)
		reply(w, http.StatusOK, s, err)
	}))

	mux.Handle("POST /api/v1/sessions/{id}/stop", withID(func(w http.ResponseWriter, r *http.Request, userID, id int64) {
		s, err := sessions.Stop(r.Context(), userID, id)
		reply(w, http.StatusOK, s, err)
	}))
}
