package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/auth"
)

type AuthService interface {
	RequestCode(ctx context.Context, phone string) error
	Verify(ctx context.Context, phone, code string) (auth.Session, error)
	Refresh(ctx context.Context, refreshToken string) (auth.Session, error)
	Logout(ctx context.Context, refreshToken string) error
	Authenticate(accessToken string) (int64, error)
	User(ctx context.Context, id int64) (auth.User, error)
}

func registerAuth(mux *http.ServeMux, svc AuthService, log *slog.Logger) {
	if svc == nil {
		return
	}

	mux.HandleFunc("POST /api/v1/auth/code", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Phone string `json:"phone"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		err := svc.RequestCode(r.Context(), req.Phone)
		var tooSoon auth.TooSoonError
		switch {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, auth.ErrBadPhone):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.As(err, &tooSoon):
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(tooSoon.RetryAfter.Seconds()))))
			writeError(w, http.StatusTooManyRequests, err.Error())
		default:
			log.Error("request code", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
		}
	})

	mux.HandleFunc("POST /api/v1/auth/verify", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Phone string `json:"phone"`
			Code  string `json:"code"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		session, err := svc.Verify(r.Context(), req.Phone, req.Code)
		writeSession(w, log, session, err)
	})

	mux.HandleFunc("POST /api/v1/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		session, err := svc.Refresh(r.Context(), req.RefreshToken)
		writeSession(w, log, session, err)
	})

	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refreshToken"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := svc.Logout(r.Context(), req.RefreshToken); err != nil {
			log.Error("logout", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.Handle("GET /api/v1/me", requireUser(svc, func(w http.ResponseWriter, r *http.Request, userID int64) {
		user, err := svc.User(r.Context(), userID)
		if errors.Is(err, auth.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "user not found")
			return
		}
		if err != nil {
			log.Error("get user", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, user)
	}))
}

func writeSession(w http.ResponseWriter, log *slog.Logger, s auth.Session, err error) {
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, s)
	case errors.Is(err, auth.ErrBadPhone):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, auth.ErrCodeInvalid), errors.Is(err, auth.ErrBadToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, auth.ErrTooManyAttempts):
		writeError(w, http.StatusTooManyRequests, err.Error())
	default:
		log.Error("auth", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// requireUser пропускает запрос дальше, только если в заголовке
// Authorization: Bearer <token> лежит действующий access-токен.
func requireUser(svc AuthService, next func(http.ResponseWriter, *http.Request, int64)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		userID, err := svc.Authenticate(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r, userID)
	})
}
