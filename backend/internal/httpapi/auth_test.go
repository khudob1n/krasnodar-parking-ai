package httpapi

import "testing"

// Полный сценарий входа через HTTP на настоящей базе.
func TestAuthFlow(t *testing.T) {
	e := newTestEnv(t)
	phone := map[string]string{"phone": "8 (918) 123-45-67"}

	if code, _ := e.call("POST", "/api/v1/auth/code", "", map[string]string{"phone": "123"}); code != 400 {
		t.Errorf("bad phone: want 400, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/auth/code", "", phone); code != 204 {
		t.Fatalf("request code: want 204, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/auth/code", "", phone); code != 429 {
		t.Errorf("second code within a minute: want 429, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/auth/verify", "", map[string]string{"phone": phone["phone"], "code": "wrong"}); code != 401 {
		t.Errorf("wrong code: want 401, got %d", code)
	}

	code, session := e.call("POST", "/api/v1/auth/verify", "", map[string]string{"phone": "+79181234567", "code": e.sms.code(t)})
	if code != 200 {
		t.Fatalf("verify: want 200, got %d %v", code, session)
	}
	access, refresh := session["accessToken"].(string), session["refreshToken"].(string)

	if code, _ := e.call("GET", "/api/v1/me", "", nil); code != 401 {
		t.Errorf("me without token: want 401, got %d", code)
	}
	code, me := e.call("GET", "/api/v1/me", access, nil)
	if code != 200 || me["phone"] != "+79181234567" {
		t.Errorf("me: got %d %v", code, me)
	}

	code, renewed := e.call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": refresh})
	if code != 200 || renewed["refreshToken"] == refresh {
		t.Fatalf("refresh: got %d %v", code, renewed)
	}
	if code, _ := e.call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": refresh}); code != 401 {
		t.Errorf("reused refresh token: want 401, got %d", code)
	}

	newRefresh := renewed["refreshToken"].(string)
	if code, _ := e.call("POST", "/api/v1/auth/logout", "", map[string]string{"refreshToken": newRefresh}); code != 204 {
		t.Errorf("logout: want 204, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/auth/refresh", "", map[string]string{"refreshToken": newRefresh}); code != 401 {
		t.Errorf("refresh after logout: want 401, got %d", code)
	}
}
