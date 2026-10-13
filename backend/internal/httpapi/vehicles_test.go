package httpapi

import (
	"fmt"
	"testing"
)

func TestVehicles(t *testing.T) {
	e := newTestEnv(t)
	alice := e.login("+79180000001")
	bob := e.login("+79180000002")

	if code, _ := e.call("GET", "/api/v1/vehicles", "", nil); code != 401 {
		t.Errorf("without token: want 401, got %d", code)
	}

	code, car := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "a 123 bc 93", "title": "Киа"})
	if code != 201 || car["plate"] != "А123ВС93" {
		t.Fatalf("add: got %d %v", code, car)
	}
	if code, _ := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "А123ВС93"}); code != 409 {
		t.Errorf("duplicate: want 409, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/vehicles", alice, map[string]string{"plate": "Ж000ЖЖ00"}); code != 400 {
		t.Errorf("bad plate: want 400, got %d", code)
	}
	// Тот же номер у другого пользователя - можно: машиной пользуется семья.
	if code, _ := e.call("POST", "/api/v1/vehicles", bob, map[string]string{"plate": "А123ВС93"}); code != 201 {
		t.Errorf("same plate for another user: want 201, got %d", code)
	}

	code, list := e.call("GET", "/api/v1/vehicles", alice, nil)
	if code != 200 || len(list["vehicles"].([]any)) != 1 {
		t.Fatalf("list: got %d %v", code, list)
	}

	path := fmt.Sprintf("/api/v1/vehicles/%v", car["id"])
	if code, _ := e.call("DELETE", path, bob, nil); code != 404 {
		t.Errorf("delete someone else's car: want 404, got %d", code)
	}
	if code, _ := e.call("DELETE", path, alice, nil); code != 204 {
		t.Errorf("delete: want 204, got %d", code)
	}
	if code, _ := e.call("DELETE", path, alice, nil); code != 404 {
		t.Errorf("delete twice: want 404, got %d", code)
	}
}
