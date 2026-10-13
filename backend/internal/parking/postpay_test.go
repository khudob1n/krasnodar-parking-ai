package parking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments/fakepay"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

// Постоплата: списание по факту при остановке, долг при отказе банка,
// оплата долга, автозакрытие через 24 часа.
func TestPostpaid(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	clock := time.Date(2026, 10, 13, 10, 0, 0, 0, pricing.Krasnodar)
	svc := NewService(f.pool, f.pay)
	svc.now = func() time.Time { return clock }
	postpaid := func(cardID int64) (Session, error) {
		return svc.Start(ctx, f.userID, StartRequest{Zone: 1103, VehicleID: f.vehicleID, CardID: cardID, Mode: ModePostpaid})
	}

	// 10:00-11:00 хорошей картой: списан итог 60 ₽.
	sess, err := postpaid(f.cardID)
	if err != nil {
		t.Fatal(err)
	}
	if sess.Mode != ModePostpaid || sess.Cost != 0 || sess.Paid != 0 || !sess.PlannedEnd.Equal(clock.Add(24*time.Hour)) {
		t.Fatalf("start: %+v", sess)
	}
	clock = clock.Add(30 * time.Minute)
	if got, _ := svc.Get(ctx, f.userID, sess.ID); got.Cost != 3000 {
		t.Errorf("active postpaid cost is cost so far: want 3000, got %d", got.Cost)
	}
	if _, err := svc.Extend(ctx, f.userID, sess.ID, 30); !errors.Is(err, ErrNoEndTime) {
		t.Errorf("extend postpaid: want ErrNoEndTime, got %v", err)
	}
	clock = clock.Add(30 * time.Minute)
	sess, err = svc.Stop(ctx, f.userID, sess.ID)
	if err != nil || sess.Cost != 6000 || sess.Paid != 6000 || sess.Debt != 0 {
		t.Fatalf("stop: %+v, %v", sess, err)
	}

	// Забыли остановить: через 24 часа закрывается сама и списывает итог.
	// С 11:00 до 11:00 следующего дня платных часов 8 + 3.
	sess, err = postpaid(0)
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(25 * time.Hour)
	if n, err := svc.CloseExpired(ctx); err != nil || n != 1 {
		t.Fatalf("CloseExpired: want 1, got %d, %v", n, err)
	}
	sess, _ = svc.Get(ctx, f.userID, sess.ID)
	if sess.Status != StatusFinished || sess.Cost != 66000 || sess.Paid != 66000 {
		t.Errorf("auto-closed postpaid: %+v", sess)
	}

	// Карта без денег: привязывается, парковка начинается, а при остановке
	// итог становится долгом.
	poor := f.addCard(t, "4000 0000 0000 9995")
	sess, err = postpaid(poor)
	if err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(30 * time.Minute) // 12:00-12:30
	sess, err = svc.Stop(ctx, f.userID, sess.ID)
	if err != nil || sess.Status != StatusFinished || sess.Debt != 3000 || sess.Paid != 0 ||
		sess.PaymentError != "insufficient funds" {
		t.Fatalf("stop with poor card: %+v, %v", sess, err)
	}
	if _, err := svc.Start(ctx, f.userID, StartRequest{Zone: 1103, VehicleID: f.vehicleID, Minutes: 60}); !errors.Is(err, ErrHasDebt) {
		t.Errorf("start with debt: want ErrHasDebt, got %v", err)
	}
	if _, err := svc.Pay(ctx, f.userID, sess.ID, poor); !errors.Is(err, payments.ErrInsufficientFunds) {
		t.Errorf("pay with poor card: want ErrInsufficientFunds, got %v", err)
	}
	if got, _ := svc.Get(ctx, f.userID, sess.ID); got.Debt != 3000 {
		t.Errorf("failed pay must keep the debt: %+v", got)
	}
	paid, err := svc.Pay(ctx, f.userID, sess.ID, f.cardID)
	if err != nil || paid.Debt != 0 || paid.Paid != 3000 || paid.PaymentError != "" || *paid.CardID != f.cardID {
		t.Fatalf("pay: %+v, %v", paid, err)
	}
	if _, err := svc.Pay(ctx, f.userID, sess.ID, 0); !errors.Is(err, ErrNothingToPay) {
		t.Errorf("pay twice: want ErrNothingToPay, got %v", err)
	}

	// Карта, на которой банк отказывает, не годится и для постоплаты.
	// Через API её не привязать, поэтому кладём в базу напрямую.
	tok, err := f.pay.Tokenize(ctx, fakepay.CardInput{Number: "4000 0000 0000 0002", ExpMonth: 12, ExpYear: 2030, CVC: "123"})
	if err != nil {
		t.Fatal(err)
	}
	var declined int64
	err = f.pool.QueryRow(ctx, `
INSERT INTO cards (user_id, token, last4, brand, exp_month, exp_year)
VALUES ($1, $2, '0002', 'visa', 12, 2030) RETURNING id`, f.userID, tok.Token).Scan(&declined)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := postpaid(declined); !errors.Is(err, payments.ErrDeclined) {
		t.Errorf("postpaid with declined card: want ErrDeclined, got %v", err)
	}
}
