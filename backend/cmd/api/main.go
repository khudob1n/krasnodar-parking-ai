// Команда api запускает HTTP-сервер.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/auth"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/config"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/httpapi"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/parking"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments/fakepay"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/reminders"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/vehicles"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("api stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ctx отменится по Ctrl+C или по сигналу остановки от Docker.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.RequireJWTSecret(); err != nil {
		return err
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}
	if cfg.ZonesFile != "" {
		n, err := zones.NewRepository(pool).ImportFileIfEmpty(ctx, cfg.ZonesFile)
		if err != nil {
			return err
		}
		if n > 0 {
			log.Info("zones imported on start", "total", n)
		}
	}

	pay := fakepay.New(pool) // TODO: настоящий платёжный провайдер
	sessions := parking.NewService(pool, pay)
	worker := reminders.NewWorker(pool, reminders.LogNotifier{Log: log}, sessions, log) // TODO: настоящие push
	workerDone := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(workerDone)
	}()
	// Перед закрытием пула дожидаемся, пока воркер доделает проход.
	defer func() {
		stop()
		<-workerDone
	}()

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: httpapi.NewHandler(httpapi.Deps{
			DB:    pool,
			Zones: zones.NewRepository(pool),
			Auth: auth.NewService(
				auth.NewPGStore(pool),
				codeSender(cfg, log),
				auth.NewTokens(cfg.JWTSecret, cfg.AccessTokenTTL),
			),
			Vehicles: vehicles.NewRepository(pool),
			Sessions: sessions,
			Devices:  reminders.NewDevices(pool),
			Cards:    payments.NewCards(pool, pay),
			FakePay:  pay.Handler(log),
			Log:      log,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// codeSender - вход по звонку, если заданы ключи uCaller, иначе код пишется в лог.
func codeSender(cfg config.Config, log *slog.Logger) auth.CodeSender {
	if cfg.UCallerServiceID == "" || cfg.UCallerKey == "" {
		log.Warn("UCALLER_SERVICE_ID or UCALLER_KEY is not set, login codes go to the log")
		return auth.LogSender{Log: log}
	}
	return auth.UCaller{ServiceID: cfg.UCallerServiceID, Key: cfg.UCallerKey}
}
