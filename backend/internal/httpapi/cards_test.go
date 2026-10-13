package httpapi

import (
	"fmt"
	"testing"
)

func TestCards(t *testing.T) {
	e := newTestEnv(t)
	alice := e.login("+79180000001")
	bob := e.login("+79180000002")

	if code, _ := e.call("GET", "/api/v1/cards", "", nil); code != 401 {
		t.Errorf("without token: want 401, got %d", code)
	}

	code, card := e.addCard(alice, "2200 0000 0000 0004")
	if code != 201 || card["last4"] != "0004" || card["brand"] != "mir" ||
		card["expMonth"] != float64(12) || card["expYear"] != float64(2030) || card["createdAt"] == nil {
		t.Fatalf("add: got %d %v", code, card)
	}
	if _, ok := card["token"]; ok {
		t.Error("token must not leak to the client")
	}
	if code, out := e.addCard(alice, "4000 0000 0000 0002"); code != 402 || out["error"] != "card declined" {
		t.Errorf("declined card: want 402, got %d %v", code, out)
	}
	if code, out := e.call("POST", "/api/v1/cards", alice, map[string]string{"token": "tok_nope"}); code != 400 || out["error"] != "unknown card token" {
		t.Errorf("unknown token: want 400, got %d %v", code, out)
	}

	// Один и тот же токен второй раз не привязать, даже другому пользователю.
	_, tok := e.call("POST", "/fakepay/v1/tokens", "", map[string]any{
		"number": "5555 5555 5555 4444", "expMonth": 1, "expYear": 2031, "cvc": "321",
	})
	if code, _ := e.call("POST", "/api/v1/cards", alice, map[string]any{"token": tok["token"]}); code != 201 {
		t.Errorf("add mastercard: want 201, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/cards", bob, map[string]any{"token": tok["token"]}); code != 409 {
		t.Errorf("same token twice: want 409, got %d", code)
	}

	code, list := e.call("GET", "/api/v1/cards", alice, nil)
	cards, _ := list["cards"].([]any)
	if code != 200 || len(cards) != 2 || cards[0].(map[string]any)["id"] != card["id"] {
		t.Fatalf("list, oldest first: got %d %v", code, list)
	}

	path := fmt.Sprintf("/api/v1/cards/%v", card["id"])
	if code, _ := e.call("DELETE", path, bob, nil); code != 404 {
		t.Errorf("delete someone else's card: want 404, got %d", code)
	}
	if code, _ := e.call("DELETE", path, alice, nil); code != 204 {
		t.Errorf("delete: want 204, got %d", code)
	}
	if code, _ := e.call("DELETE", path, alice, nil); code != 404 {
		t.Errorf("delete twice: want 404, got %d", code)
	}
}
