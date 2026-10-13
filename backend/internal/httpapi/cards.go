package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
)

type CardStore interface {
	Add(ctx context.Context, userID int64, token string) (payments.UserCard, error)
	List(ctx context.Context, userID int64) ([]payments.UserCard, error)
	Delete(ctx context.Context, userID, id int64) error
}

func registerCards(mux *http.ServeMux, svc AuthService, store CardStore, log *slog.Logger) {
	if svc == nil || store == nil {
		return
	}

	mux.Handle("GET /api/v1/cards", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		list, err := store.List(r.Context(), userID)
		if err != nil {
			log.Error("list cards", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cards": list})
	}))

	// Приложение сначала получает токен у провайдера, сюда шлёт только его.
	mux.Handle("POST /api/v1/cards", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		var req struct {
			Token string `json:"token"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		c, err := store.Add(r.Context(), userID, req.Token)
		switch {
		case err == nil:
			writeJSON(w, http.StatusCreated, c)
		case errors.Is(err, payments.ErrUnknownToken):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, payments.ErrDeclined):
			writeError(w, http.StatusPaymentRequired, err.Error())
		case errors.Is(err, payments.ErrDuplicateCard):
			writeError(w, http.StatusConflict, err.Error())
		default:
			log.Error("add card", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}))

	mux.Handle("DELETE /api/v1/cards/{id}", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id must be an integer")
			return
		}
		err = store.Delete(r.Context(), userID, id)
		switch {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, payments.ErrCardNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			log.Error("delete card", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	}))
}
