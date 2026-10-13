package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/auth"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/parking"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments/fakepay"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/reminders"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/vehicles"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// testEnv - API на настоящей базе для сквозных тестов.
type testEnv struct {
	t    *testing.T
	pool *pgxpool.Pool
	srv  *httptest.Server
	sms  *smsBox
}

// newTestEnv поднимает API поверх TEST_DATABASE_URL, очищая таблицы.
// Без переменной тест пропускается.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE users, login_codes, refresh_tokens, zones CASCADE"); err != nil {
		t.Fatal(err)
	}

	sms := &smsBox{}
	svc := auth.NewService(auth.NewPGStore(pool), sms, auth.NewTokens("test-secret-test-secret-test-secret", time.Minute))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pay := fakepay.New(pool)
	srv := httptest.NewServer(NewHandler(Deps{
		DB:       pool,
		Zones:    zones.NewRepository(pool),
		Auth:     svc,
		Vehicles: vehicles.NewRepository(pool),
		Sessions: parking.NewService(pool, pay),
		Devices:  reminders.NewDevices(pool),
		Cards:    payments.NewCards(pool, pay),
		FakePay:  pay.Handler(log),
		Log:      log,
	}))
	t.Cleanup(srv.Close)
	return &testEnv{t: t, pool: pool, srv: srv, sms: sms}
}

// call делает запрос и возвращает код ответа и разобранный JSON.
func (e *testEnv) call(method, path, token string, body any) (int, map[string]any) {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer resp.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// login входит по номеру и возвращает access-токен.
func (e *testEnv) login(phone string) string {
	e.t.Helper()
	if code, out := e.call("POST", "/api/v1/auth/code", "", map[string]string{"phone": phone}); code != 204 {
		e.t.Fatalf("request code: %d %v", code, out)
	}
	code, out := e.call("POST", "/api/v1/auth/verify", "", map[string]string{"phone": phone, "code": e.sms.code(e.t)})
	if code != 200 {
		e.t.Fatalf("verify: %d %v", code, out)
	}
	return out["accessToken"].(string)
}

// addCard привязывает пользователю тестовую карту так же, как приложение:
// токен у fakepay, потом токен в наш API. Возвращает ответ API.
func (e *testEnv) addCard(token, number string) (int, map[string]any) {
	e.t.Helper()
	code, tok := e.call("POST", "/fakepay/v1/tokens", "", map[string]any{
		"number": number, "expMonth": 12, "expYear": 2030, "cvc": "123",
	})
	if code != 201 {
		e.t.Fatalf("tokenize %s: %d %v", number, code, tok)
	}
	return e.call("POST", "/api/v1/cards", token, map[string]any{"token": tok["token"]})
}

// smsBox вместо звонка выдаёт коды по порядку и запоминает последний.
type smsBox struct {
	mu   sync.Mutex
	n    int
	last string
}

func (b *smsBox) SendCode(context.Context, string) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.n++
	b.last = fmt.Sprintf("%04d", b.n)
	return b.last, nil
}

func (b *smsBox) code(t *testing.T) string {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.last == "" {
		t.Fatal("no code sent")
	}
	return b.last
}
