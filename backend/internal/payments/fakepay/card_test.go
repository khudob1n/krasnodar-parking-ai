package fakepay

import (
	"errors"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

func TestValidate(t *testing.T) {
	now := time.Date(2026, 10, 13, 12, 0, 0, 0, pricing.Krasnodar)
	ok := CardInput{Number: "4242 4242 4242 4242", ExpMonth: 12, ExpYear: 2030, CVC: "123"}
	with := func(f func(*CardInput)) CardInput {
		in := ok
		f(&in)
		return in
	}
	tests := []struct {
		name string
		in   CardInput
		want error
	}{
		{"valid", ok, nil},
		{"dashes", with(func(c *CardInput) { c.Number = "4242-4242-4242-4242" }), nil},
		{"this month", with(func(c *CardInput) { c.ExpMonth, c.ExpYear = 10, 2026 }), nil},
		{"luhn", with(func(c *CardInput) { c.Number = "4242 4242 4242 4241" }), ErrBadNumber},
		{"short", with(func(c *CardInput) { c.Number = "4242 4242 4242 424" }), ErrBadNumber},
		{"letters", with(func(c *CardInput) { c.Number = "4242 4242 4242 42a2" }), ErrBadNumber},
		{"month 13", with(func(c *CardInput) { c.ExpMonth = 13 }), ErrBadExpiry},
		{"2-digit year", with(func(c *CardInput) { c.ExpYear = 30 }), ErrBadExpiry},
		{"last month", with(func(c *CardInput) { c.ExpMonth, c.ExpYear = 9, 2026 }), ErrExpired},
		{"last year", with(func(c *CardInput) { c.ExpYear = 2025 }), ErrExpired},
		{"cvc 4 digits", with(func(c *CardInput) { c.CVC = "1234" }), ErrBadCVC},
		{"cvc letters", with(func(c *CardInput) { c.CVC = "12a" }), ErrBadCVC},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := validate(tt.in, now); !errors.Is(err, tt.want) {
				t.Errorf("want %v, got %v", tt.want, err)
			}
		})
	}
}

func TestBrandAndBehaviour(t *testing.T) {
	tests := []struct {
		number, brand, behaviour string
	}{
		{"4242424242424242", "visa", behaviourOK},
		{"2200000000000004", "mir", behaviourOK},
		{"2204000000000000", "mir", behaviourOK},
		{"5555555555554444", "mastercard", behaviourOK},
		{"2221000000000009", "mastercard", behaviourOK},
		{"3530111333300000", "card", behaviourOK},
		{"4000000000000002", "visa", behaviourDeclined},
		{"4000000000009995", "visa", behaviourInsufficient},
	}
	for _, tt := range tests {
		if !luhn(tt.number) {
			t.Errorf("%s: test number must pass luhn", tt.number)
		}
		if got := brand(tt.number); got != tt.brand {
			t.Errorf("%s: brand want %s, got %s", tt.number, tt.brand, got)
		}
		if got := behaviour(tt.number); got != tt.behaviour {
			t.Errorf("%s: behaviour want %s, got %s", tt.number, tt.behaviour, got)
		}
	}
}
