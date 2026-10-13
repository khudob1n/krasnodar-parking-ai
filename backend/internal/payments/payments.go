// Package payments - оплата картой: интерфейс платёжного провайдера и
// карты пользователя.
//
// Номер карты к нам не попадает: приложение отдаёт его провайдеру (у
// настоящих - через их SDK) и получает токен. Мы храним только токен и
// то, что можно показать: последние 4 цифры, платёжную систему, срок.
package payments

import (
	"context"
	"errors"
)

var (
	ErrUnknownToken      = errors.New("unknown card token")
	ErrDeclined          = errors.New("card declined")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrRefundTooBig      = errors.New("refund is bigger than the charge")
	ErrUnknownCharge     = errors.New("unknown charge")
)

// Card - то, что провайдер сообщает о карте.
type Card struct {
	Last4    string `json:"last4"`
	Brand    string `json:"brand"` // mir, visa, mastercard или card
	ExpMonth int    `json:"expMonth"`
	ExpYear  int    `json:"expYear"`
}

type Provider interface {
	// Verify проверяет карту по токену (у настоящих провайдеров - списание и возврат 1 ₽ при привязке).
	Verify(ctx context.Context, token string) (Card, error)
	// Charge списывает amount; key - ключ идемпотентности: повтор с тем же ключом не списывает второй раз и возвращает тот же ref.
	Charge(ctx context.Context, token string, amount int64, key string) (ref string, err error)
	// Refund возвращает часть или всё списание ref.
	Refund(ctx context.Context, ref string, amount int64) error
}

// IsPaymentError - отказ провайдера, о котором надо сказать пользователю
// (в отличие от сбоя базы или сети).
func IsPaymentError(err error) bool {
	return errors.Is(err, ErrUnknownToken) || errors.Is(err, ErrDeclined) ||
		errors.Is(err, ErrInsufficientFunds)
}
