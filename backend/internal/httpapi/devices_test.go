package httpapi

import "testing"

func TestDevices(t *testing.T) {
	e := newTestEnv(t)
	token := e.login("+79180000001")

	body := map[string]string{"token": "push-token", "platform": "android"}
	if code, _ := e.call("POST", "/api/v1/devices", "", body); code != 401 {
		t.Errorf("without auth: want 401, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/devices", token, body); code != 204 {
		t.Errorf("register: want 204, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/devices", token, map[string]string{"token": "t", "platform": "nokia"}); code != 400 {
		t.Errorf("bad platform: want 400, got %d", code)
	}
	if code, _ := e.call("POST", "/api/v1/devices/unregister", token, map[string]string{"token": "push-token"}); code != 204 {
		t.Errorf("unregister: want 204, got %d", code)
	}
}
