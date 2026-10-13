package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/reminders"
)

type DeviceStore interface {
	Register(ctx context.Context, userID int64, token, platform string) error
	Unregister(ctx context.Context, userID int64, token string) error
}

func registerDevices(mux *http.ServeMux, svc AuthService, devices DeviceStore, log *slog.Logger) {
	if svc == nil || devices == nil {
		return
	}
	type deviceRequest struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}

	mux.Handle("POST /api/v1/devices", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		var req deviceRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		err := devices.Register(r.Context(), userID, req.Token, req.Platform)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, reminders.ErrBadDevice):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("register device", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}))

	mux.Handle("POST /api/v1/devices/unregister", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		var req deviceRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := devices.Unregister(r.Context(), userID, req.Token); err != nil {
			log.Error("unregister device", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}
