package fakepay

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// Handler - публичная HTTP-ручка провайдера, которую приложение зовёт
// напрямую, без нашей авторизации (как SDK с публичным ключом):
// POST /fakepay/v1/tokens.
func (p *Provider) Handler(log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /fakepay/v1/tokens", func(w http.ResponseWriter, r *http.Request) {
		var in CardInput
		dec := json.NewDecoder(io.LimitReader(r.Body, 4<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		resp, err := p.Tokenize(r.Context(), in)
		switch {
		case err == nil:
			writeJSON(w, http.StatusCreated, resp)
		case errors.Is(err, ErrBadNumber), errors.Is(err, ErrBadExpiry),
			errors.Is(err, ErrExpired), errors.Is(err, ErrBadCVC):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error("fakepay tokenize", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
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
