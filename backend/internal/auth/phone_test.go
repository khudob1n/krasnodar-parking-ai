package auth

import "testing"

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"8 (918) 123-45-67": "+79181234567",
		"+7 918 123 45 67":  "+79181234567",
		"79181234567":       "+79181234567",
		"9181234567":        "+79181234567",
	}
	for in, want := range ok {
		got, err := NormalizePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "12345", "+7 861 123 45 67", "+1 918 123 45 67", "8918123456a", "+7918123456789"} {
		if got, err := NormalizePhone(in); err == nil {
			t.Errorf("NormalizePhone(%q) = %q, want error", in, got)
		}
	}
}
