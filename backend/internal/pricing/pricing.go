// Package pricing считает стоимость парковки по тарифу зоны.
package pricing

import (
	"errors"
	"fmt"
	"time"
	_ "time/tzdata" // база часовых поясов внутри бинарника: в Docker-образе её может не быть
)

// Krasnodar - часовой пояс города. Платные часы на табличках указаны
// по местному времени, поэтому считаем всё в нём.
var Krasnodar = mustLoad("Europe/Moscow")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// Tariff - тариф зоны.
type Tariff struct {
	PricePerHour int64  `json:"pricePerHour"` // копейки за час в платное время
	PaidFrom     string `json:"paidFrom"`     // "08:00"
	PaidTo       string `json:"paidTo"`       // "19:00"; равное PaidFrom значит «круглосуточно»
}

// Quote - результат расчёта.
type Quote struct {
	Total       int64 `json:"total"`       // копейки к оплате
	PaidMinutes int   `json:"paidMinutes"` // сколько минут из интервала попало в платное время
}

// Calculate считает стоимость стоянки с start до end.
// Оплата поминутная: платные секунды * цену часа / 3600, с округлением
// вверх до копейки.
func Calculate(t Tariff, start, end time.Time) (Quote, error) {
	if !end.After(start) {
		return Quote{}, errors.New("end must be after start")
	}
	from, err := minutesOfDay(t.PaidFrom)
	if err != nil {
		return Quote{}, err
	}
	to, err := minutesOfDay(t.PaidTo)
	if err != nil {
		return Quote{}, err
	}

	start, end = start.In(Krasnodar), end.In(Krasnodar)
	var paid time.Duration
	// Идём по календарным дням, захватив и предыдущий день: его платное окно
	// может переходить через полночь (например, 20:00-08:00).
	day := midnight(start).AddDate(0, 0, -1)
	for day.Before(end) {
		for _, w := range paidWindows(day, from, to) {
			paid += overlap(start, end, w[0], w[1])
		}
		day = day.AddDate(0, 0, 1)
	}

	seconds := int64(paid / time.Second)
	return Quote{
		Total:       ceilDiv(seconds*t.PricePerHour, 3600),
		PaidMinutes: int(paid / time.Minute),
	}, nil
}

// paidWindows возвращает платные интервалы, которые начинаются в этот день.
func paidWindows(day time.Time, from, to int) [][2]time.Time {
	at := func(min int) time.Time {
		return time.Date(day.Year(), day.Month(), day.Day(), min/60, min%60, 0, 0, Krasnodar)
	}
	switch {
	case from == to: // круглосуточно
		return [][2]time.Time{{at(0), day.AddDate(0, 0, 1)}}
	case from < to:
		return [][2]time.Time{{at(from), at(to)}}
	default: // окно через полночь
		next := day.AddDate(0, 0, 1)
		return [][2]time.Time{{at(from), time.Date(next.Year(), next.Month(), next.Day(), to/60, to%60, 0, 0, Krasnodar)}}
	}
}

func overlap(aStart, aEnd, bStart, bEnd time.Time) time.Duration {
	s, e := maxTime(aStart, bStart), minTime(aEnd, bEnd)
	if e.After(s) {
		return e.Sub(s)
	}
	return 0
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func minutesOfDay(s string) (int, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, fmt.Errorf("bad time %q", s)
	}
	return t.Hour()*60 + t.Minute(), nil
}

func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
