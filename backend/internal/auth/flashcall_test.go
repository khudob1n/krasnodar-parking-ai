package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFlashCall(t *testing.T) {
	var gotPhone, gotAuth, gotClient string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/flash-call/call" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotPhone = body["phone"]
		gotAuth = r.Header.Get("Authorization")
		gotClient = r.Header.Get("Client")
		_, _ = w.Write([]byte(`{"data":{"pin":"4721"}}`))
	}))
	defer srv.Close()

	code, err := FlashCall{Token: "secret", BaseURL: srv.URL}.SendCode(context.Background(), "+79181234567")
	if err != nil {
		t.Fatal(err)
	}
	if code != "4721" {
		t.Errorf("code = %q, want 4721", code)
	}
	if gotPhone != "79181234567" || gotAuth != "Bearer secret" || gotClient != plusofonClient {
		t.Errorf("request: phone=%q auth=%q client=%q", gotPhone, gotAuth, gotClient)
	}
}

func TestFlashCallErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"ошибка API", http.StatusUnauthorized, `{"message":"Unauthenticated."}`},
		{"нет кода в ответе", http.StatusOK, `{"data":{}}`},
		{"не JSON", http.StatusOK, `oops`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			if _, err := (FlashCall{BaseURL: srv.URL}).SendCode(context.Background(), "+79181234567"); err == nil {
				t.Error("want error")
			}
		})
	}
}
