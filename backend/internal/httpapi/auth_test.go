package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/auth"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
)

// smsBox запоминает последнее «отправленное» SMS.
type smsBox struct {
	mu   sync.Mutex
	last string
}

func (b *smsBox) Send(_ context.Context, _, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.last = text
	return nil
}

func (b *smsBox) code(t *testing.T) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	m := regexp.MustCompile(`\d{4}$`).FindString(b.last)
	if m == "" {
		t.Fatalf("no code in sms %q", b.last)
	}
	return m
}

// Полный сценарий входа через HTTP на настоящей базе.
func TestAuthFlow(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE users, login_codes, refresh_tokens CASCADE"); err != nil {
		t.Fatal(err)
	}

	sms := &smsBox{}
	svc := auth.NewService(auth.NewPGStore(pool), sms, auth.NewTokens("test-secret-test-secret-test-secret", time.Minute))
	srv := httptest.NewServer(NewHandler(Deps{DB: pool, Auth: svc, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}))
	defer srv.Close()

	call := func(method, path, token string, body any) (int, map[string]any) {
		t.Helper()
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req, _ := http.NewRequest(method, srv.URL+path, &buf)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		out := map[string]any{}
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	phone := map[string]string{"phone": "8 (918) 123-45-67"}

	if code, _ := call("POST", "/api/v1/auth/code", "", map[string]string{"phone": "123"}); code != 400 {
		t.Errorf("bad phone: want 400, got %d", code)
	}
	if code, _ := call("POST", "/api/v1/auth/code", "", phone); code != 204 {
		t.Fatalf("request code: want 204, got %d", code)
	}
	if code, _ := call("POST", "/api/v1/auth/code", "", phone); code != 429 {
		t.Errorf("second code within a minute: want 429, got %d", code)
	}
	if code, _ := call("POST", "/api/v1/auth/verify", "", map[string]string{"phone": phone["phone"], "code": "wrong"}); code != 401 {
		t.Errorf("wrong code: want 401, got %d", code)
	}

	code, session := call("POST", "/api/v1/auth/verify", "", map[string]string{"phone": "+79181234567", "code": sms.code(t)})
	if code != 200 {
		t.Fatalf("verify: want 200, got %d %v", code, session)
	}
	access, refresh := session["accessToken"].(string), session["refreshToken"].(string)

	if code, _ := call("GET", "/api/v1/me", "", nil); code != 401 {
		t.Errorf("me without token: want 401, got %d", code)
	}
	code, me := call("GET", "/api/v1/me", access, nil)
	if code != 200 || me["phone"] != "+79181234567" {
		t.Errorf("me: got %d %v", code, me)
	}

	code, renewed := call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": refresh})
	if code != 200 || renewed["refreshToken"] == refresh {
		t.Fatalf("refresh: got %d %v", code, renewed)
	}
	if code, _ := call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": refresh}); code != 401 {
		t.Errorf("reused refresh token: want 401, got %d", code)
	}

	newRefresh := renewed["refreshToken"].(string)
	if code, _ := call("POST", "/api/v1/auth/logout", "", map[string]string{"refreshToken": newRefresh}); code != 204 {
		t.Errorf("logout: want 204, got %d", code)
	}
	if code, _ := call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": newRefresh}); code != 401 {
		t.Errorf("refresh after logout: want 401, got %d", code)
	}
}
