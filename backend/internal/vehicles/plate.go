// Package vehicles - автомобили пользователя.
package vehicles

import (
	"errors"
	"regexp"
	"strings"
)

var ErrBadPlate = errors.New("plate must look like А123ВС93")

// В российских номерах только 12 букв, которые выглядят одинаково
// в кириллице и латинице. Люди часто набирают их латиницей.
var latinToCyrillic = strings.NewReplacer(
	"A", "А", "B", "В", "E", "Е", "K", "К", "M", "М", "H", "Н",
	"O", "О", "P", "Р", "C", "С", "T", "Т", "Y", "У", "X", "Х",
)

var platePattern = regexp.MustCompile(`^[АВЕКМНОРСТУХ]\d{3}[АВЕКМНОРСТУХ]{2}\d{2,3}$`)

// NormalizePlate приводит «а 123 bc 93» к «А123ВС93».
func NormalizePlate(raw string) (string, error) {
	p := strings.ToUpper(raw)
	p = strings.NewReplacer(" ", "", "-", "", "|", "").Replace(p)
	p = latinToCyrillic.Replace(p)
	if !platePattern.MatchString(p) {
		return "", ErrBadPlate
	}
	return p, nil
}
