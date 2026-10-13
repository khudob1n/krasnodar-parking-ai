package fakepay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
)

func newTestProvider(t *testing.T) *Provider {
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
	return New(pool)
}

func TestProvider(t *testing.T) {
	p := newTestProvider(t)
	ctx := context.Background()
	tokenize := func(number string) string {
		t.Helper()
		resp, err := p.Tokenize(ctx, CardInput{Number: number, ExpMonth: 12, ExpYear: 2030, CVC: "123"})
		if err != nil {
			t.Fatal(err)
		}
		return resp.Token
	}
	ok := tokenize("4242 4242 4242 4242")
	declined := tokenize("4000 0000 0000 0002")
	poor := tokenize("4000 0000 0000 9995")

	if c, err := p.Verify(ctx, ok); err != nil || c.Last4 != "4242" || c.Brand != "visa" {
		t.Errorf("verify ok card: %+v, %v", c, err)
	}
	if _, err := p.Verify(ctx, declined); !errors.Is(err, payments.ErrDeclined) {
		t.Errorf("verify declined: want ErrDeclined, got %v", err)
	}
	if _, err := p.Verify(ctx, poor); err != nil {
		t.Errorf("verify insufficient card must pass, got %v", err)
	}
	if _, err := p.Verify(ctx, "tok_nope"); !errors.Is(err, payments.ErrUnknownToken) {
		t.Errorf("verify unknown: want ErrUnknownToken, got %v", err)
	}

	if _, err := p.Charge(ctx, declined, 100, "k-"+declined); !errors.Is(err, payments.ErrDeclined) {
		t.Errorf("charge declined: got %v", err)
	}
	if _, err := p.Charge(ctx, poor, 100, "k-"+poor); !errors.Is(err, payments.ErrInsufficientFunds) {
		t.Errorf("charge insufficient: got %v", err)
	}

	key := "k-" + ok
	ref, err := p.Charge(ctx, ok, 1000, key)
	if err != nil {
		t.Fatal(err)
	}
	again, err := p.Charge(ctx, ok, 1000, key)
	if err != nil || again != ref {
		t.Errorf("same key must return the same charge: %q vs %q, %v", ref, again, err)
	}
	var n int
	_ = p.pool.QueryRow(ctx, `SELECT count(*) FROM fakepay_charges WHERE token = $1`, ok).Scan(&n)
	if n != 1 {
		t.Errorf("want 1 charge, got %d", n)
	}

	if err := p.Refund(ctx, ref, 600); err != nil {
		t.Fatal(err)
	}
	if err := p.Refund(ctx, ref, 500); !errors.Is(err, payments.ErrRefundTooBig) {
		t.Errorf("refund over the charge: want ErrRefundTooBig, got %v", err)
	}
	if err := p.Refund(ctx, ref, 400); err != nil {
		t.Errorf("refund the rest: %v", err)
	}
	if err := p.Refund(ctx, "ch_nope", 1); !errors.Is(err, payments.ErrUnknownCharge) {
		t.Errorf("refund unknown: want ErrUnknownCharge, got %v", err)
	}
}

func TestTokensHTTP(t *testing.T) {
	p := newTestProvider(t)
	srv := httptest.NewServer(p.Handler(slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()

	post := func(body string) (int, map[string]any) {
		t.Helper()
		resp, err := http.Post(srv.URL+"/fakepay/v1/tokens", "application/json", bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out := map[string]any{}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}

	code, out := post(`{"number":"2200 0000 0000 0004","expMonth":12,"expYear":2030,"cvc":"123"}`)
	token, _ := out["token"].(string)
	if code != 201 || out["last4"] != "0004" || out["brand"] != "mir" || out["expYear"] != float64(2030) {
		t.Fatalf("tokenize: %d %v", code, out)
	}
	if !strings.HasPrefix(token, "tok_") || len(token) != 4+24 {
		t.Errorf("token must be tok_ + 24 hex, got %q", token)
	}

	tests := []struct{ body, want string }{
		{`{"number":"1234 5678 9012 3456","expMonth":12,"expYear":2030,"cvc":"123"}`, "card number is invalid"},
		{`{"number":"4242 4242 4242 4242","expMonth":1,"expYear":2020,"cvc":"123"}`, "card is expired"},
		{`{"number":"4242 4242 4242 4242","expMonth":12,"expYear":2030,"cvc":"12"}`, "cvc must be 3 digits"},
		{`{"number":`, "invalid JSON body"},
	}
	for _, tt := range tests {
		if code, out := post(tt.body); code != 400 || out["error"] != tt.want {
			t.Errorf("%s: want 400 %q, got %d %v", tt.body, tt.want, code, out)
		}
	}
}
