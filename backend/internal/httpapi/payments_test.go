package httpapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// Деньги при парковке: без карты не начать, отказ банка - парковки нет,
// при остановке лишнее возвращается.
func TestSessionPayments(t *testing.T) {
	e := newTestEnv(t)
	err := zones.NewRepository(e.pool).Upsert(context.Background(), []zones.Zone{
		{Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская", PricePerHour: 6000,
			PaidFrom: "00:00", PaidTo: "00:00", Location: zones.Point{Lat: 45.02, Lon: 38.97}},
	})
	if err != nil {
		t.Fatal(err)
	}
	alice := e.login("+79180000001")
	bob := e.login("+79180000002")
	_, car := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "А123ВС93"})
	_, bobCard := e.addCard(bob, "4242 4242 4242 4242")

	start := func(body map[string]any) (int, map[string]any) {
		body["zone"], body["vehicleId"] = 1103, car["id"]
		return e.call("POST", "/api/v1/sessions", alice, body)
	}
	sessions := func() int {
		_, list := e.call("GET", "/api/v1/sessions", alice, nil)
		return len(list["sessions"].([]any))
	}

	if code, out := start(map[string]any{"minutes": 60}); code != 402 || out["error"] != "add a payment card first" {
		t.Errorf("no cards: want 402, got %d %v", code, out)
	}

	_, poor := e.addCard(alice, "4000 0000 0000 9995")
	if code, out := start(map[string]any{"minutes": 60}); code != 402 || out["error"] != "insufficient funds" {
		t.Errorf("insufficient funds: want 402, got %d %v", code, out)
	}
	if n := sessions(); n != 0 {
		t.Errorf("failed charge must not create a session, got %d", n)
	}

	_, good := e.addCard(alice, "4242 4242 4242 4242")
	if code, out := start(map[string]any{"minutes": 60, "cardId": bobCard["id"]}); code != 404 || out["error"] != "card not found" {
		t.Errorf("someone else's card: want 404, got %d %v", code, out)
	}
	if code, _ := start(map[string]any{"minutes": 60, "cardId": poor["id"]}); code != 402 {
		t.Errorf("chosen card is poor: want 402, got %d", code)
	}
	if code, _ := start(map[string]any{"minutes": 60, "mode": "later"}); code != 400 {
		t.Errorf("unknown mode: want 400, got %d", code)
	}

	// Без cardId - последняя добавленная карта.
	code, sess := start(map[string]any{"minutes": 60, "mode": "prepaid"})
	if code != 201 || sess["cardId"] != good["id"] || sess["paid"] != float64(6000) {
		t.Fatalf("start: got %d %v", code, sess)
	}
	path := fmt.Sprintf("/api/v1/sessions/%v", sess["id"])

	// Карту удалили: продлить нечем, остановить можно.
	if code, _ := e.call("DELETE", fmt.Sprintf("/api/v1/cards/%v", good["id"]), alice, nil); code != 204 {
		t.Fatalf("delete card in use: want 204, got %d", code)
	}
	code, out := e.call("POST", path+"/extend", alice, map[string]int{"minutes": 30})
	if code != 402 || out["error"] != "add a payment card first" {
		t.Errorf("extend without card: want 402, got %d %v", code, out)
	}
	_, got := e.call("GET", path, alice, nil)
	if got["cardId"] != nil || got["paid"] != float64(6000) || got["plannedEnd"] != sess["plannedEnd"] {
		t.Errorf("failed extend must not change the session: %v", got)
	}

	code, stopped := e.call("POST", path+"/stop", alice, nil)
	if code != 200 || stopped["paid"] != stopped["cost"] {
		t.Fatalf("stop: got %d %v", code, stopped)
	}

	// В payments: списание при старте и возврат при остановке.
	var charged, refunded int64
	err = e.pool.QueryRow(context.Background(), `
SELECT COALESCE(sum(amount) FILTER (WHERE kind = 'charge'), 0),
       COALESCE(sum(amount) FILTER (WHERE kind = 'refund'), 0)
FROM payments WHERE session_id = $1`, sess["id"]).Scan(&charged, &refunded)
	if err != nil {
		t.Fatal(err)
	}
	if charged != 6000 || float64(charged-refunded) != stopped["paid"] {
		t.Errorf("payments: charged %d, refunded %d, paid %v", charged, refunded, stopped["paid"])
	}
}
