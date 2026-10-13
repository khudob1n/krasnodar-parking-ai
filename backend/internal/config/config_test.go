package config

import "testing"

func TestHTTPAddr(t *testing.T) {
	tests := []struct {
		name, httpAddr, port, want string
	}{
		{"по умолчанию", "", "", ":8080"},
		{"порт от хостинга", "", "10000", ":10000"},
		{"HTTP_ADDR важнее PORT", "127.0.0.1:9000", "10000", "127.0.0.1:9000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", tt.httpAddr)
			t.Setenv("PORT", tt.port)
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.HTTPAddr != tt.want {
				t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, tt.want)
			}
		})
	}
}
