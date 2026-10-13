package fakepay

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBadNumber = errors.New("card number is invalid")
	ErrBadExpiry = errors.New("expiry date is invalid")
	ErrExpired   = errors.New("card is expired")
	ErrBadCVC    = errors.New("cvc must be 3 digits")
)

// Поведение тестовой карты определяется её номером, как у настоящих
// провайдеров в тестовом режиме.
const (
	behaviourOK           = "ok"
	behaviourDeclined     = "declined"     // отказ и при проверке, и при списании
	behaviourInsufficient = "insufficient" // проверка проходит, списания - нет
)

var testCards = map[string]string{
	"4000000000000002": behaviourDeclined,
	"4000000000009995": behaviourInsufficient,
}

// CardInput - данные карты, которые вводит пользователь.
type CardInput struct {
	Number   string `json:"number"`
	ExpMonth int    `json:"expMonth"`
	ExpYear  int    `json:"expYear"`
	CVC      string `json:"cvc"`
}

// card - проверенная карта: номер без пробелов.
type card struct {
	number   string
	expMonth int
	expYear  int
}

// validate проверяет введённую карту на момент now.
func validate(in CardInput, now time.Time) (card, error) {
	number := strings.NewReplacer(" ", "", "-", "").Replace(in.Number)
	if len(number) != 16 || !digitsOnly(number) || !luhn(number) {
		return card{}, ErrBadNumber
	}
	if in.ExpMonth < 1 || in.ExpMonth > 12 || in.ExpYear < 1000 || in.ExpYear > 9999 {
		return card{}, ErrBadExpiry
	}
	// Карта действует до конца месяца, указанного на ней.
	if in.ExpYear < now.Year() || in.ExpYear == now.Year() && in.ExpMonth < int(now.Month()) {
		return card{}, ErrExpired
	}
	if len(in.CVC) != 3 || !digitsOnly(in.CVC) {
		return card{}, ErrBadCVC
	}
	return card{number: number, expMonth: in.ExpMonth, expYear: in.ExpYear}, nil
}

// brand определяет платёжную систему по первым цифрам номера.
func brand(number string) string {
	p2, _ := strconv.Atoi(number[:2])
	p4, _ := strconv.Atoi(number[:4])
	switch {
	case p4 >= 2200 && p4 <= 2204:
		return "mir"
	case number[0] == '4':
		return "visa"
	case p2 >= 51 && p2 <= 55, p4 >= 2221 && p4 <= 2720:
		return "mastercard"
	default:
		return "card"
	}
}

func behaviour(number string) string {
	if b, ok := testCards[number]; ok {
		return b
	}
	return behaviourOK
}

// luhn - контрольная сумма номера карты: ловит опечатки в одной цифре и
// перестановки соседних.
func luhn(number string) bool {
	sum := 0
	for i := range len(number) {
		d := int(number[len(number)-1-i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

func digitsOnly(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
