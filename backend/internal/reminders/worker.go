// Package reminders - фоновая задача, которая предупреждает водителя,
// что оплаченное время скоро закончится, и сообщает, когда оно вышло.
package reminders

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

// Notifier доставляет уведомление на устройства. Настоящий (RuStore Push,
// FCM) появится вместе с клиентом, пока есть LogNotifier.
type Notifier interface {
	Notify(ctx context.Context, tokens []string, title, body string) error
}

// SessionCloser закрывает истёкшие парковки (parking.Service).
type SessionCloser interface {
	CloseExpired(ctx context.Context) (int, error)
}

type Worker struct {
	pool     *pgxpool.Pool
	notifier Notifier
	closer   SessionCloser
	log      *slog.Logger
	now      func() time.Time
	Interval time.Duration // как часто проверять
	Before   time.Duration // за сколько до конца напоминать
}

func NewWorker(pool *pgxpool.Pool, n Notifier, c SessionCloser, log *slog.Logger) *Worker {
	return &Worker{
		pool: pool, notifier: n, closer: c, log: log, now: time.Now,
		Interval: 30 * time.Second,
		Before:   10 * time.Minute,
	}
}

// Run проверяет парковки каждые Interval, пока ctx не отменят.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		if err := w.Tick(ctx); err != nil && ctx.Err() == nil {
			w.log.Error("reminders tick", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// due - парковка, о которой пора сообщить.
type due struct {
	userID     int64
	zone       int
	plate      string
	plannedEnd time.Time
}

// Tick - один проход: напомнить, сообщить об окончании, закрыть истёкшие.
func (w *Worker) Tick(ctx context.Context) error {
	now := w.now()

	// UPDATE ... RETURNING сразу помечает строки как отправленные. Если
	// запущено несколько копий API, одну парковку заберёт только одна.
	soon, err := w.claim(ctx, `
UPDATE parking_sessions SET reminded_at = $1
WHERE ended_at IS NULL AND reminded_at IS NULL
  AND planned_end > $1 AND planned_end <= $2
RETURNING user_id, zone_number, plate, planned_end`, now, now.Add(w.Before))
	if err != nil {
		return err
	}
	for _, d := range soon {
		left := d.plannedEnd.Sub(now).Round(time.Minute)
		w.send(ctx, d.userID, "Парковка скоро закончится",
			fmt.Sprintf("Зона %d, %s: осталось %d мин. Продлите, чтобы не получить штраф.", d.zone, d.plate, int(left.Minutes())))
	}

	// Сообщаем об окончании только по свежим парковкам (за последний час),
	// чтобы после долгого простоя сервера не завалить людей старыми.
	expired, err := w.claim(ctx, `
UPDATE parking_sessions SET expired_notified_at = $1
WHERE expired_notified_at IS NULL
  AND planned_end <= $1 AND planned_end > $2
  AND (ended_at IS NULL OR ended_at >= planned_end)
RETURNING user_id, zone_number, plate, planned_end`, now, now.Add(-time.Hour))
	if err != nil {
		return err
	}
	for _, d := range expired {
		w.send(ctx, d.userID, "Время парковки вышло",
			fmt.Sprintf("Зона %d, %s: оплачено до %s.", d.zone, d.plate, d.plannedEnd.In(pricing.Krasnodar).Format("15:04")))
	}

	if _, err := w.closer.CloseExpired(ctx); err != nil {
		return fmt.Errorf("close expired sessions: %w", err)
	}
	return nil
}

func (w *Worker) claim(ctx context.Context, sql string, args ...any) ([]due, error) {
	rows, err := w.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("claim reminders: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (due, error) {
		var d due
		err := row.Scan(&d.userID, &d.zone, &d.plate, &d.plannedEnd)
		return d, err
	})
	if err != nil {
		return nil, fmt.Errorf("claim reminders: %w", err)
	}
	return list, nil
}

// send отправляет уведомление на все устройства пользователя. Ошибку
// только логируем: одно недоставленное уведомление не должно ломать
// остальные.
func (w *Worker) send(ctx context.Context, userID int64, title, body string) {
	rows, err := w.pool.Query(ctx, `SELECT token FROM push_tokens WHERE user_id = $1`, userID)
	if err != nil {
		w.log.Error("load push tokens", "user", userID, "err", err)
		return
	}
	tokens, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		w.log.Error("load push tokens", "user", userID, "err", err)
		return
	}
	if len(tokens) == 0 {
		return
	}
	if err := w.notifier.Notify(ctx, tokens, title, body); err != nil {
		w.log.Error("send notification", "user", userID, "err", err)
	}
}

// LogNotifier пишет уведомления в лог. Только для разработки.
type LogNotifier struct{ Log *slog.Logger }

func (n LogNotifier) Notify(_ context.Context, tokens []string, title, body string) error {
	n.Log.Warn("push not sent, dev mode", "tokens", len(tokens), "title", title, "body", body)
	return nil
}
