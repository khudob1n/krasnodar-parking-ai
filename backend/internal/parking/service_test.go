package parking

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments/fakepay"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// Тест управляет часами сервиса, чтобы проверить, что бывает, когда
// оплаченное время выходит.
// fixture - база с зоной 1103 (60 ₽/ч с 08:00 до 19:00), пользователем,
// его машиной и картой 4242 4242 4242 4242.
type fixture struct {
	pool      *pgxpool.Pool
	pay       *fakepay.Provider
	userID    int64
	vehicleID int64
	cardID    int64
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE users, zones CASCADE"); err != nil {
		t.Fatal(err)
	}
	err = zones.NewRepository(pool).Upsert(ctx, []zones.Zone{{
		Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская", PricePerHour: 6000,
		PaidFrom: "08:00", PaidTo: "19:00", Location: zones.Point{Lat: 45.02, Lon: 38.97},
	}})
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{pool: pool, pay: fakepay.New(pool)}
	if err := pool.QueryRow(ctx, `INSERT INTO users (phone) VALUES ('+79180000001') RETURNING id`).Scan(&f.userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO vehicles (user_id, plate) VALUES ($1, 'А123ВС93') RETURNING id`, f.userID).Scan(&f.vehicleID); err != nil {
		t.Fatal(err)
	}
	f.cardID = f.addCard(t, "4242 4242 4242 4242")
	return f
}

// addCard привязывает пользователю тестовую карту и возвращает её id.
func (f fixture) addCard(t *testing.T, number string) int64 {
	t.Helper()
	ctx := context.Background()
	tok, err := f.pay.Tokenize(ctx, fakepay.CardInput{Number: number, ExpMonth: 12, ExpYear: 2030, CVC: "123"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := payments.NewCards(f.pool, f.pay).Add(ctx, f.userID, tok.Token)
	if err != nil {
		t.Fatal(err)
	}
	return c.ID
}

func TestSessionExpiry(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	pool, userID, vehicleID := f.pool, f.userID, f.vehicleID

	clock := time.Date(2026, 10, 13, 18, 0, 0, 0, pricing.Krasnodar)
	svc := NewService(pool, f.pay)
	svc.now = func() time.Time { return clock }

	// 18:00, ставим на 2 часа. Платный только час до 19:00.
	sess, err := svc.Start(ctx, userID, StartRequest{Zone: 1103, VehicleID: vehicleID, Minutes: 120})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Cost != 6000 || sess.Paid != 6000 {
		t.Errorf("estimated cost and prepaid: want 6000, got %d and %d", sess.Cost, sess.Paid)
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
