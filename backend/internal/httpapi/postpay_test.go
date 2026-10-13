package httpapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

func TestPostpaidAPI(t *testing.T) {
	e := newTestEnv(t)
	err := zones.NewRepository(e.pool).Upsert(context.Background(), []zones.Zone{
		{Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская", PricePerHour: 6000,
			PaidFrom: "00:00", PaidTo: "00:00", Location: zones.Point{Lat: 45.02, Lon: 38.97}},
	})
	if err != nil {
		t.Fatal(err)
	}
	alice := e.login("+79180000001")
	_, car := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "А123ВС93"})
	_, good := e.addCard(alice, "4242 4242 4242 4242")
	_, poor := e.addCard(alice, "4000 0000 0000 9995")

	code, sess := e.call("POST", "/api/v1/sessions", alice, map[string]any{
		"zone": 1103, "vehicleId": car["id"], "mode": "postpaid",
	})
	if code != 201 || sess["mode"] != "postpaid" || sess["cardId"] != poor["id"] || sess["debt"] != float64(0) {
		t.Fatalf("start postpaid: got %d %v", code, sess)
	}
	if _, ok := sess["paymentError"]; ok {
		t.Errorf("paymentError must be absent without debt: %v", sess)
	}
	path := fmt.Sprintf("/api/v1/sessions/%v", sess["id"])
	if code, out := e.call("POST", path+"/extend", alice, map[string]int{"minutes": 30}); code != 400 || out["error"] != "postpaid parking has no end time" {
		t.Errorf("extend postpaid: want 400, got %d %v", code, out)
	}
	if code, out := e.call("POST", path+"/pay", alice, nil); code != 409 || out["error"] != "nothing to pay" {
		t.Errorf("pay active: want 409, got %d %v", code, out)
	}

	// Делаем вид, что стоим уже час.
	if _, err := e.pool.Exec(context.Background(),
		`UPDATE parking_sessions SET started_at = started_at - interval '1 hour' WHERE id = $1`, sess["id"]); err != nil {
		t.Fatal(err)
	}
	code, stopped := e.call("POST", path+"/stop", alice, nil)
	if code != 200 || stopped["status"] != "finished" || stopped["paid"] != float64(0) ||
		stopped["debt"] != stopped["cost"] || stopped["paymentError"] != "insufficient funds" {
		t.Fatalf("stop with poor card: got %d %v", code, stopped)
	}
	if stopped["cost"].(float64) < 6000 {
		t.Errorf("an hour must cost at least 60 ₽, got %v", stopped["cost"])
	}

	code, out := e.call("POST", "/api/v1/sessions", alice, map[string]any{"zone": 1103, "vehicleId": car["id"], "minutes": 60})
	if code != 402 || out["error"] != "pay for the previous parking first" {
		t.Errorf("start with debt: want 402, got %d %v", code, out)
	}
	if code, _ := e.call("POST", path+"/pay", alice, map[string]any{"cardId": poor["id"]}); code != 402 {
		t.Errorf("pay with poor card: want 402, got %d", code)
	}
	code, paid := e.call("POST", path+"/pay", alice, map[string]any{"cardId": good["id"]})
	if code != 200 || paid["debt"] != float64(0) || paid["paid"] != stopped["cost"] {
		t.Fatalf("pay: got %d %v", code, paid)
	}
	if code, _ := e.call("POST", path+"/pay", alice, nil); code != 409 {
		t.Errorf("pay twice: want 409, got %d", code)
	}
	// Последняя добавленная карта - бедная: предоплата не пройдёт, но уже не из-за долга.
	code, out = e.call("POST", "/api/v1/sessions", alice, map[string]any{"zone": 1103, "vehicleId": car["id"], "minutes": 60})
	if code != 402 || out["error"] != "insufficient funds" {
		t.Errorf("start with poor default card: want 402 insufficient funds, got %d %v", code, out)
	}
	code, out = e.call("POST", "/api/v1/sessions", alice, map[string]any{
		"zone": 1103, "vehicleId": car["id"], "minutes": 60, "cardId": good["id"],
	})
	if code != 201 {
		t.Errorf("start after paying the debt: want 201, got %d %v", code, out)
	}
}
