// Package fakepay - тестовый платёжный провайдер. Ведёт себя как
// настоящий: приложение само отправляет ему номер карты и получает
// токен, а наш API видит только токены.
//
// Карты и списания fakepay хранит в своих таблицах fakepay_cards и
// fakepay_charges. Это как бы база другой компании, которая просто
// оказалась в нашем Postgres: остальной код в них не заглядывает.
package fakepay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

type Provider struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

var _ payments.Provider = (*Provider)(nil)

func New(pool *pgxpool.Pool) *Provider {
	return &Provider{pool: pool, now: time.Now}
}

// TokenResponse - ответ на токенизацию карты.
type TokenResponse struct {
	Token    string `json:"token"`
	Last4    string `json:"last4"`
	Brand    string `json:"brand"`
	ExpMonth int    `json:"expMonth"`
	ExpYear  int    `json:"expYear"`
}

// Tokenize проверяет карту и выдаёт токен, по которому её потом можно
// списывать. Номер не сохраняется, только то, как карта себя ведёт.
func (p *Provider) Tokenize(ctx context.Context, in CardInput) (TokenResponse, error) {
	c, err := validate(in, p.now().In(pricing.Krasnodar))
	if err != nil {
		return TokenResponse{}, err
	}
	resp := TokenResponse{
		Token: "tok_" + randomHex(12), Last4: c.number[12:], Brand: brand(c.number),
		ExpMonth: c.expMonth, ExpYear: c.expYear,
	}
	_, err = p.pool.Exec(ctx, `
INSERT INTO fakepay_cards (token, last4, brand, exp_month, exp_year, behaviour)
VALUES ($1, $2, $3, $4, $5, $6)`,
		resp.Token, resp.Last4, resp.Brand, resp.ExpMonth, resp.ExpYear, behaviour(c.number))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("save card: %w", err)
	}
	return resp, nil
}

func (p *Provider) Verify(ctx context.Context, token string) (payments.Card, error) {
	var c payments.Card
	var b string
	err := p.pool.QueryRow(ctx, `
SELECT last4, brand, exp_month, exp_year, behaviour FROM fakepay_cards WHERE token = $1`, token,
	).Scan(&c.Last4, &c.Brand, &c.ExpMonth, &c.ExpYear, &b)
	if errors.Is(err, pgx.ErrNoRows) {
		return payments.Card{}, payments.ErrUnknownToken
	}
	if err != nil {
		return payments.Card{}, fmt.Errorf("verify card: %w", err)
	}
	if b == behaviourDeclined {
		return payments.Card{}, payments.ErrDeclined
	}
	return c, nil
}

func (p *Provider) Charge(ctx context.Context, token string, amount int64, key string) (string, error) {
	if amount <= 0 {
		return "", fmt.Errorf("charge amount must be positive, got %d", amount)
	}
	// Повтор с тем же ключом - то же списание, второй раз денег не берём.
	var ref string
	err := p.pool.QueryRow(ctx, `SELECT ref FROM fakepay_charges WHERE idem_key = $1`, key).Scan(&ref)
	if err == nil {
		return ref, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("charge: %w", err)
	}

	var b string
	err = p.pool.QueryRow(ctx, `SELECT behaviour FROM fakepay_cards WHERE token = $1`, token).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", payments.ErrUnknownToken
	}
	if err != nil {
		return "", fmt.Errorf("charge: %w", err)
	}
	switch b {
	case behaviourDeclined:
		return "", payments.ErrDeclined
	case behaviourInsufficient:
		return "", payments.ErrInsufficientFunds
	}

	// ON CONFLICT - на случай двух одновременных запросов с одним ключом:
	// выиграет один, второй прочитает его ref.
	_, err = p.pool.Exec(ctx, `
INSERT INTO fakepay_charges (ref, token, amount, idem_key) VALUES ($1, $2, $3, $4)
ON CONFLICT (idem_key) DO NOTHING`, "ch_"+randomHex(12), token, amount, key)
	if err != nil {
		return "", fmt.Errorf("charge: %w", err)
	}
	if err := p.pool.QueryRow(ctx, `SELECT ref FROM fakepay_charges WHERE idem_key = $1`, key).Scan(&ref); err != nil {
		return "", fmt.Errorf("charge: %w", err)
	}
	return ref, nil
}

func (p *Provider) Refund(ctx context.Context, ref string, amount int64) error {
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive, got %d", amount)
	}
	tag, err := p.pool.Exec(ctx, `
UPDATE fakepay_charges SET refunded = refunded + $2 WHERE ref = $1 AND refunded + $2 <= amount`, ref, amount)
	if err != nil {
		return fmt.Errorf("refund: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var exists bool
	if err := p.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM fakepay_charges WHERE ref = $1)`, ref).Scan(&exists); err != nil {
		return fmt.Errorf("refund: %w", err)
	}
	if !exists {
		return payments.ErrUnknownCharge
	}
	return payments.ErrRefundTooBig
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read не возвращает ошибок
	return hex.EncodeToString(b)
}
