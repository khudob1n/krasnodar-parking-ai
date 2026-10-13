package vehicles

import "testing"

func TestNormalizePlate(t *testing.T) {
	ok := map[string]string{
		"А123ВС93":     "А123ВС93",
		"а 123 вс 93":  "А123ВС93",
		"a123bc93":     "А123ВС93", // латиница
		"Х777ХХ123":    "Х777ХХ123",
		"x 777 xx|123": "Х777ХХ123",
	}
	for in, want := range ok {
		if got, err := NormalizePlate(in); err != nil || got != want {
			t.Errorf("NormalizePlate(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "Ж123ВС93", "А12ВС93", "А123ВС9", "А123ВС1234", "AB123C93"} {
		if got, err := NormalizePlate(in); err == nil {
			t.Errorf("NormalizePlate(%q) = %q, want error", in, got)
		}
	}
}
