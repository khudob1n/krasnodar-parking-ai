package pricing

import (
	"testing"
	"time"
)

func at(day, hour, min int) time.Time {
	return time.Date(2026, time.October, day, hour, min, 0, 0, Krasnodar)
}

func TestCalculate(t *testing.T) {
	day := Tariff{PricePerHour: 6000, PaidFrom: "08:00", PaidTo: "19:00"}
	allDay := Tariff{PricePerHour: 10000, PaidFrom: "00:00", PaidTo: "00:00"}
	night := Tariff{PricePerHour: 3000, PaidFrom: "20:00", PaidTo: "08:00"}
	free := Tariff{PricePerHour: 0, PaidFrom: "00:00", PaidTo: "00:00"}

	cases := []struct {
		name        string
		tariff      Tariff
		start, end  time.Time
		wantTotal   int64
		wantMinutes int
	}{
		{"час днём", day, at(13, 10, 0), at(13, 11, 0), 6000, 60},
		{"15 минут", day, at(13, 10, 0), at(13, 10, 15), 1500, 15},
		{"заехал до начала платного", day, at(13, 7, 30), at(13, 8, 30), 3000, 30},
		{"уехал после конца платного", day, at(13, 18, 30), at(13, 20, 0), 3000, 30},
		{"ночь бесплатно", day, at(13, 19, 0), at(14, 8, 0), 0, 0},
		{"сутки", day, at(13, 8, 0), at(14, 8, 0), 66000, 660},
		{"круглосуточно 90 минут", allDay, at(13, 23, 0), at(14, 0, 30), 15000, 90},
		{"окно через полночь", night, at(13, 7, 0), at(13, 21, 0), 6000, 120},
		{"бесплатная зона", free, at(13, 10, 0), at(13, 12, 0), 0, 120},
		{"округление вверх до копейки", Tariff{PricePerHour: 100, PaidFrom: "00:00", PaidTo: "00:00"}, at(13, 10, 0), at(13, 10, 1), 2, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Calculate(c.tariff, c.start, c.end)
			if err != nil {
				t.Fatal(err)
			}
			if got.Total != c.wantTotal || got.PaidMinutes != c.wantMinutes {
				t.Errorf("got %d коп./%d мин, want %d коп./%d мин", got.Total, got.PaidMinutes, c.wantTotal, c.wantMinutes)
			}
		})
	}
}

func TestCalculateUsesKrasnodarTime(t *testing.T) {
	day := Tariff{PricePerHour: 6000, PaidFrom: "08:00", PaidTo: "19:00"}
	// 05:00 UTC - это 08:00 в Краснодаре, начало платного времени.
	start := time.Date(2026, time.October, 13, 5, 0, 0, 0, time.UTC)
	got, err := Calculate(day, start, start.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 6000 {
		t.Errorf("want 6000, got %d", got.Total)
	}
}

func TestCalculateErrors(t *testing.T) {
	day := Tariff{PricePerHour: 6000, PaidFrom: "08:00", PaidTo: "19:00"}
	if _, err := Calculate(day, at(13, 10, 0), at(13, 10, 0)); err == nil {
		t.Error("empty interval: want error")
	}
	if _, err := Calculate(Tariff{PaidFrom: "8", PaidTo: "19:00"}, at(13, 10, 0), at(13, 11, 0)); err == nil {
		t.Error("bad time: want error")
	}
}
