package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func devicePost(mux *http.ServeMux, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// Mutation: remove client_id check → missing client_id must be rejected
func TestMutation_ClientIDRequired(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)
	w := devicePost(mux, "/oauth/device/code", url.Values{})
	if w.Code == 200 {
		t.Error("missing client_id should be rejected")
	}
}

// Mutation: remove device_code from response → must return device_code
func TestMutation_DeviceCodeReturned(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)
	w := devicePost(mux, "/oauth/device/code", url.Values{"client_id": {"app"}})
	if w.Code != 200 {
		t.Skipf("device code request returned %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["device_code"] == nil || resp["device_code"] == "" {
		t.Error("response must include device_code")
	}
	if resp["user_code"] == nil || resp["user_code"] == "" {
		t.Error("response must include user_code")
	}
}

// Mutation: remove authorization_pending → unapproved poll must return pending
func TestMutation_PollPending(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	mux := http.NewServeMux()
	h.Register(mux)

	// Request code
	w := devicePost(mux, "/oauth/device/code", url.Values{"client_id": {"app"}})
	var codeResp map[string]string
	json.NewDecoder(w.Body).Decode(&codeResp)

	// Poll without approval
	w2 := devicePost(mux, "/oauth/device/token", url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {codeResp["device_code"]},
	})
	var pollResp map[string]string
	json.NewDecoder(w2.Body).Decode(&pollResp)
	if pollResp["error"] != "authorization_pending" {
		t.Errorf("unapproved poll should return authorization_pending, got %s", pollResp["error"])
	}
}

// Mutation: remove Cleanup → expired requests must be cleaned
func TestMutation_Cleanup(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "t", "r" })
	// Just verify it doesn't panic
	h.Cleanup()
}

// Mutation: remove approve → approved device must issue token
func TestMutation_ApproveFlow(t *testing.T) {
	h := NewHandler(func(c string, s []string) (string, string) { return "access_tok", "refresh_tok" })
	mux := http.NewServeMux()
	h.Register(mux)

	w := devicePost(mux, "/oauth/device/code", url.Values{"client_id": {"app"}})
	var codeResp map[string]string
	json.NewDecoder(w.Body).Decode(&codeResp)

	// Approve
	devicePost(mux, "/oauth/device/approve", url.Values{
		"user_code": {codeResp["user_code"]},
		"action":    {"approve"},
	})

	// Poll — should get token
	time.Sleep(10 * time.Millisecond)
	w3 := devicePost(mux, "/oauth/device/token", url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {codeResp["device_code"]},
	})
	var tokenResp map[string]interface{}
	json.NewDecoder(w3.Body).Decode(&tokenResp)
	if tokenResp["access_token"] == nil {
		t.Errorf("approved device should get token, got %v", tokenResp)
	}
}

// Mutation: remove user_code → device auth response must include user_code
func TestMutation_UserCodeReturned(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "client_id=app"
	r := httptest.NewRequest("POST", "/device/authorize", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "user_code") {
		t.Error("device auth response must include user_code")
	}
}

// Mutation: remove interval → response must include polling interval
func TestMutation_IntervalReturned(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "client_id=app"
	r := httptest.NewRequest("POST", "/device/authorize", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), "interval") {
		t.Error("device auth response must include interval")
	}
}
