package httpapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

func TestSessions(t *testing.T) {
	e := newTestEnv(t)
	err := zones.NewRepository(e.pool).Upsert(context.Background(), []zones.Zone{
		{Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская", PricePerHour: 6000,
			PaidFrom: "00:00", PaidTo: "00:00", Location: zones.Point{Lat: 45.02, Lon: 38.97}},
		{Number: 12001, Kind: zones.KindFree, Address: "ул. Московская",
			PaidFrom: "00:00", PaidTo: "00:00", Location: zones.Point{Lat: 45.05, Lon: 39.0}},
	})
	if err != nil {
		t.Fatal(err)
	}

	alice := e.login("+79180000001")
	bob := e.login("+79180000002")
	_, car := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "А123ВС93"})
	_, bobCar := e.call("POST", "/api/v1/vehicles", bob, map[string]string{"plate": "А123ВС93"})
	_, card := e.addCard(alice, "4242 4242 4242 4242")
	e.addCard(bob, "4242 4242 4242 4242")

	start := func(token string, zone int, vehicle any, minutes int) (int, map[string]any) {
		return e.call("POST", "/api/v1/sessions", token, map[string]any{"zone": zone, "vehicleId": vehicle, "minutes": minutes})
	}

	if code, _ := start(alice, 12001, car["id"], 60); code != 400 {
		t.Errorf("free zone: want 400, got %d", code)
	}
	if code, _ := start(alice, 999, car["id"], 60); code != 404 {
		t.Errorf("unknown zone: want 404, got %d", code)
	}
	if code, _ := start(alice, 1103, bobCar["id"], 60); code != 404 {
		t.Errorf("someone else's car: want 404, got %d", code)
	}
	if code, _ := start(alice, 1103, car["id"], 0); code != 400 {
		t.Errorf("zero minutes: want 400, got %d", code)
	}

	code, sess := start(alice, 1103, car["id"], 60)
	if code != 201 || sess["status"] != "active" || sess["cost"] != float64(6000) ||
		sess["paid"] != float64(6000) || sess["mode"] != "prepaid" || sess["cardId"] != card["id"] {
		t.Fatalf("start: got %d %v", code, sess)
	}
	// Тот же номер нельзя поставить второй раз, даже из другого аккаунта.
	if code, _ := start(bob, 1103, bobCar["id"], 30); code != 409 {
		t.Errorf("same plate twice: want 409, got %d", code)
	}

	path := fmt.Sprintf("/api/v1/sessions/%v", sess["id"])
	if code, _ := e.call("GET", path, bob, nil); code != 404 {
		t.Errorf("someone else's session: want 404, got %d", code)
	}
	code, ext := e.call("POST", path+"/extend", alice, map[string]int{"minutes": 30})
	if code != 200 || ext["cost"] != float64(9000) || ext["paid"] != float64(9000) {
		t.Errorf("extend: got %d %v", code, ext)
	}
	if code, _ := e.call("POST", path+"/extend", alice, map[string]int{"minutes": 1440}); code != 400 {
		t.Errorf("extend beyond 24h: want 400, got %d", code)
	}

	code, stopped := e.call("POST", path+"/stop", alice, nil)
	if code != 200 || stopped["status"] != "finished" || stopped["endedAt"] == nil {
		t.Fatalf("stop: got %d %v", code, stopped)
	}
	if cost := stopped["cost"].(float64); cost > 100 { // простояли секунды - копейки
		t.Errorf("stopped right away: want almost free, got %v", cost)
	}
	if stopped["paid"] != stopped["cost"] {
		t.Errorf("stop must refund the rest: paid %v, cost %v", stopped["paid"], stopped["cost"])
	}
	if code, _ := e.call("POST", path+"/stop", alice, nil); code != 409 {
		t.Errorf("stop twice: want 409, got %d", code)
	}

	code, list := e.call("GET", "/api/v1/sessions", alice, nil)
	if code != 200 || len(list["sessions"].([]any)) != 1 {
		t.Errorf("list: got %d %v", code, list)
	}
	// После остановки машину снова можно поставить.
	if code, _ := start(bob, 1103, bobCar["id"], 30); code != 201 {
		t.Errorf("start after stop: want 201, got %d", code)
	}
}
