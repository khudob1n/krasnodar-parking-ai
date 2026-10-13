package parking

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// Тест управляет часами сервиса, чтобы проверить, что бывает, когда
// оплаченное время выходит.
func TestSessionExpiry(t *testing.T) {
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
	// Дневной тариф 60 ₽/ч с 08:00 до 19:00.
	err = zones.NewRepository(pool).Upsert(ctx, []zones.Zone{{
		Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская", PricePerHour: 6000,
		PaidFrom: "08:00", PaidTo: "19:00", Location: zones.Point{Lat: 45.02, Lon: 38.97},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var userID, vehicleID int64
	if err := pool.QueryRow(ctx, `INSERT INTO users (phone) VALUES ('+79180000001') RETURNING id`).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO vehicles (user_id, plate) VALUES ($1, 'А123ВС93') RETURNING id`, userID).Scan(&vehicleID); err != nil {
		t.Fatal(err)
	}

	clock := time.Date(2026, 10, 13, 18, 0, 0, 0, pricing.Krasnodar)
	svc := NewService(pool)
	svc.now = func() time.Time { return clock }

	// 18:00, ставим на 2 часа. Платный только час до 19:00.
	sess, err := svc.Start(ctx, userID, StartRequest{Zone: 1103, VehicleID: vehicleID, Minutes: 120})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Cost != 6000 {
		t.Errorf("estimated cost: want 6000, got %d", sess.Cost)
	}

	// 20:30: время вышло. Сессия показывается завершённой в 20:00.
	clock = clock.Add(150 * time.Minute)
	got, err := svc.Get(ctx, userID, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusFinished || got.EndedAt == nil || !got.EndedAt.Equal(sess.PlannedEnd) {
		t.Errorf("expired session: got %+v", got)
	}
	if _, err := svc.Stop(ctx, userID, sess.ID); !errors.Is(err, ErrNotActive) {
		t.Errorf("stop expired: want ErrNotActive, got %v", err)
	}

	// Новая парковка той же машины закрывает старую в базе и не упирается
	// в «машина уже стоит».
	next, err := svc.Start(ctx, userID, StartRequest{Zone: 1103, VehicleID: vehicleID, Minutes: 60})
	if err != nil {
		t.Fatalf("start after expiry: %v", err)
	}
	var cost int64
	if err := pool.QueryRow(ctx, `SELECT cost FROM parking_sessions WHERE id = $1`, sess.ID).Scan(&cost); err != nil {
		t.Fatal(err)
	}
	if cost != 6000 {
		t.Errorf("closed session cost: want 6000, got %d", cost)
	}

	// Фоновое закрытие: в 22:00 вторая парковка тоже истекла.
	clock = clock.Add(90 * time.Minute)
	n, err := svc.CloseExpired(ctx)
	if err != nil || n != 1 {
		t.Errorf("CloseExpired: want 1, got %d, %v", n, err)
	}
	got, err = svc.Get(ctx, userID, next.ID)
	if err != nil || got.Status != StatusFinished || got.Cost != 0 {
		t.Errorf("night session must be free and finished: %+v, %v", got, err)
	}
}
