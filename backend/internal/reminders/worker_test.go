package reminders

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

type sentPush struct {
	tokens []string
	title  string
	body   string
}

type fakeNotifier struct {
	mu   sync.Mutex
	sent []sentPush
}

func (f *fakeNotifier) Notify(_ context.Context, tokens []string, title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, sentPush{tokens, title, body})
	return nil
}

func (f *fakeNotifier) take() []sentPush {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := f.sent
	f.sent = nil
	return out
}

type noopCloser struct{}

func (noopCloser) CloseExpired(context.Context) (int, error) { return 0, nil }

func TestWorker(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE users, zones CASCADE"); err != nil {
		t.Fatal(err)
	}

	start := time.Date(2026, 10, 13, 10, 0, 0, 0, pricing.Krasnodar)
	var userID int64
	err = pool.QueryRow(ctx, `INSERT INTO users (phone) VALUES ('+79180000001') RETURNING id`).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
INSERT INTO zones (number, kind, address, location, price_per_hour, paid_from, paid_to,
                   total_places, disabled_places, in_building, tariff_text)
VALUES (1103, 'street', 'ул. Комсомольская', ST_MakePoint(38.97, 45.02)::geography, 6000,
        '08:00', '19:00', 10, 1, false, '')`)
	if err != nil {
		t.Fatal(err)
	}
	// Парковка с 10:00 до 11:00.
	_, err = pool.Exec(ctx, `
INSERT INTO parking_sessions (user_id, zone_number, plate, price_per_hour, paid_from, paid_to, started_at, planned_end)
VALUES ($1, 1103, 'А123ВС93', 6000, '08:00', '19:00', $2, $3)`, userID, start, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	// Постоплатная с тем же planned_end: о ней не напоминаем.
	_, err = pool.Exec(ctx, `
INSERT INTO parking_sessions (user_id, zone_number, plate, price_per_hour, paid_from, paid_to, started_at, planned_end, mode)
VALUES ($1, 1103, 'В456ОР93', 6000, '08:00', '19:00', $2, $3, 'postpaid')`, userID, start, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	devices := NewDevices(pool)
	if err := devices.Register(ctx, userID, "phone-token", "android"); err != nil {
		t.Fatal(err)
	}
	if err := devices.Register(ctx, userID, "x", "windows"); err == nil {
		t.Error("unknown platform must be rejected")
	}

	notifier := &fakeNotifier{}
	w := NewWorker(pool, notifier, noopCloser{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	clock := start
	w.now = func() time.Time { return clock }

	step := func(at time.Duration) []sentPush {
		t.Helper()
		clock = start.Add(at)
		if err := w.Tick(ctx); err != nil {
			t.Fatal(err)
		}
		return notifier.take()
	}

	if got := step(40 * time.Minute); len(got) != 0 {
		t.Errorf("10:40, 20 min left: want nothing, got %+v", got)
	}
	got := step(52 * time.Minute)
	if len(got) != 1 || got[0].title != "Парковка скоро закончится" || !strings.Contains(got[0].body, "осталось 8 мин") {
		t.Fatalf("10:52: want reminder, got %+v", got)
	}
	if got[0].tokens[0] != "phone-token" {
		t.Errorf("wrong tokens: %v", got[0].tokens)
	}
	if got := step(55 * time.Minute); len(got) != 0 {
		t.Errorf("reminder must be sent once, got %+v", got)
	}
	got = step(61 * time.Minute)
	if len(got) != 1 || got[0].title != "Время парковки вышло" || !strings.Contains(got[0].body, "оплачено до 11:00") {
		t.Fatalf("11:01: want expiry notice, got %+v", got)
	}
	if got := step(62 * time.Minute); len(got) != 0 {
		t.Errorf("expiry notice must be sent once, got %+v", got)
	}

	// После выхода из аккаунта на устройство ничего не шлём.
	if err := devices.Unregister(ctx, userID, "phone-token"); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM push_tokens`).Scan(&n)
	if n != 0 {
		t.Errorf("want 0 tokens after unregister, got %d", n)
	}
}
