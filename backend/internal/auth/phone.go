package auth

import (
	"errors"
	"strings"
)

var ErrBadPhone = errors.New("phone must be a Russian mobile number")

// NormalizePhone приводит российский номер к виду +79181234567.
// Принимает «8 (918) 123-45-67», «+7 918 123 45 67», «79181234567».
func NormalizePhone(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range raw {
		switch {
		case r >= '0' && r <= '9':
			digits.WriteRune(r)
		case r == '+' || r == ' ' || r == '-' || r == '(' || r == ')':
			// оформление номера, пропускаем
		default:
			return "", ErrBadPhone
		}
	}
	d := digits.String()
	if len(d) == 11 && (d[0] == '8' || d[0] == '7') {
		d = d[1:]
	}
	// Мобильные номера в России - 10 цифр, начинаются на 9.
	if len(d) != 10 || d[0] != '9' {
		return "", ErrBadPhone
	}
	return "+7" + d, nil
}
