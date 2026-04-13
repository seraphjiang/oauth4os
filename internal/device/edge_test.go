package device

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: device auth returns user_code and device_code
func TestEdge_DeviceAuthReturnsCode(t *testing.T) {
	h := NewHandler(nil)
	body := "client_id=app&scope=read"
	r := httptest.NewRequest("POST", "/oauth/device/code", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.RequestCode(w, r)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "device_code") {
		t.Error("response should contain device_code")
	}
	if !strings.Contains(w.Body.String(), "user_code") {
		t.Error("response should contain user_code")
	}
}

// Edge: poll with unknown device_code returns error
func TestEdge_PollUnknownCode(t *testing.T) {
	h := NewHandler(nil)
	body := "grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=unknown&client_id=app"
	r := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.PollToken(w, r)
	if w.Code == 200 {
		t.Error("unknown device_code should not return 200")
	}
}

// Edge: device auth requires POST
func TestEdge_RequiresPOST(t *testing.T) {
	h := NewHandler(nil)
	w := httptest.NewRecorder()
	h.RequestCode(w, httptest.NewRequest("GET", "/oauth/device/code", nil))
	if w.Code == 200 {
		t.Error("GET should not be accepted")
	}
}
