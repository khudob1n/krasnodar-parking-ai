package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUCaller(t *testing.T) {
	var got struct {
		Phone int64  `json:"phone"`
		Code  string `json:"code"`
	}
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1.0/initCall" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"status":true,"ucaller_id":103000,"phone":"7918***4567","code":"` + got.Code + `"}`))
	}))
	defer srv.Close()

	code, err := UCaller{ServiceID: "42", Key: "secret", BaseURL: srv.URL}.SendCode(context.Background(), "+79181234567")
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != codeDigits || code != got.Code {
		t.Errorf("code = %q, sent %q", code, got.Code)
	}
	if got.Phone != 79181234567 || gotAuth != "Bearer secret.42" {
		t.Errorf("request: phone=%d auth=%q", got.Phone, gotAuth)
	}
}

func TestUCallerErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"ошибка API", http.StatusOK, `{"status":false,"error":"Invalid phone","code":3}`},
		{"ошибка HTTP", http.StatusInternalServerError, `{"status":false,"error":"oops","code":0}`},
		{"не JSON", http.StatusOK, `oops`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			if _, err := (UCaller{BaseURL: srv.URL}).SendCode(context.Background(), "+79181234567"); err == nil {
				t.Error("want error")
			}
		})
	}
}
