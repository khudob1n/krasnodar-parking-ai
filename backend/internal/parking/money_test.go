package parking

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

// Предоплата: списание при старте, доплата при продлении, возврат при
// остановке - сначала по последнему списанию.
func TestPrepaidMoney(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 10, 0, 0, 0, pricing.Krasnodar)
	svc := NewService(f.pool, f.pay)
	svc.now = func() time.Time { return clock }

	poor := f.addCard(t, "4000 0000 0000 9995")
	if _, err := svc.Start(ctx, f.userID, StartRequest{Zone: 1103, VehicleID: f.vehicleID, Minutes: 60, CardID: poor}); !errors.Is(err, payments.ErrInsufficientFunds) {
		t.Fatalf("poor card: want ErrInsufficientFunds, got %v", err)
	}

	sess, err := svc.Start(ctx, f.userID, StartRequest{Zone: 1103, VehicleID: f.vehicleID, Minutes: 60, CardID: f.cardID})
	if err != nil {
		t.Fatal(err)
	}
	if sess.Paid != 6000 || sess.Mode != ModePrepaid || sess.CardID == nil || *sess.CardID != f.cardID {
		t.Fatalf("start: %+v", sess)
	}

	// 10:10, продлили на час: доплата 60 ₽.
	clock = clock.Add(10 * time.Minute)
	sess, err = svc.Extend(ctx, f.userID, sess.ID, 60)
	if err != nil || sess.Paid != 12000 || sess.Cost != 12000 {
		t.Fatalf("extend: %+v, %v", sess, err)
	}

	// Сменили карту сессии на бедную: продление не проходит и ничего не меняет.
	if _, err := f.pool.Exec(ctx, `UPDATE parking_sessions SET card_id = $1 WHERE id = $2`, poor, sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Extend(ctx, f.userID, sess.ID, 30); !errors.Is(err, payments.ErrInsufficientFunds) {
		t.Errorf("extend with poor card: want ErrInsufficientFunds, got %v", err)
	}
	if got, _ := svc.Get(ctx, f.userID, sess.ID); got.Paid != 12000 || !got.PlannedEnd.Equal(sess.PlannedEnd) {
		t.Errorf("failed extend changed the session: %+v", got)
	}

	// 10:30, остановили: стоимость 30 ₽, вернуть 90 ₽ - 60 с продления и 30 со старта.
	clock = clock.Add(20 * time.Minute)
	sess, err = svc.Stop(ctx, f.userID, sess.ID)
	if err != nil || sess.Cost != 3000 || sess.Paid != 3000 {
		t.Fatalf("stop: %+v, %v", sess, err)
	}
	rows, err := f.pool.Query(ctx, `SELECT kind, amount FROM payments WHERE session_id = $1 ORDER BY id`, sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var kind string
		var amount int64
		if err := rows.Scan(&kind, &amount); err != nil {
			t.Fatal(err)
		}
		got = append(got, fmt.Sprintf("%s %d", kind, amount))
	}
	want := []string{"charge 6000", "charge 6000", "refund 6000", "refund 3000"}
	if len(got) != len(want) {
		t.Fatalf("payments: want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("payments: want %v, got %v", want, got)
			break
		}
	}
}
