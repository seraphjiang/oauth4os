package device

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestDeviceFlowConcurrent verifies multiple device flows don't interfere.
func TestDeviceFlowConcurrent(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			clientID := "cli-" + string(rune('A'+n))

			// Request code
			w := formPost(mux, "/oauth/device/code", url.Values{"client_id": {clientID}})
			if w.Code != 200 {
				t.Errorf("client %s: expected 200, got %d", clientID, w.Code)
				return
			}
			var cr map[string]interface{}
			json.NewDecoder(w.Body).Decode(&cr)
			dc := cr["device_code"].(string)
			uc := cr["user_code"].(string)

			// Approve
			formPost(mux, "/oauth/device/approve", url.Values{"user_code": {uc}, "action": {"approve"}})

			// Poll
			w2 := formPost(mux, "/oauth/device/token", url.Values{
				"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				"device_code": {dc},
			})
			var tok map[string]interface{}
			json.NewDecoder(w2.Body).Decode(&tok)
			if tok["access_token"] == nil {
				t.Errorf("client %s: no access token", clientID)
			}
		}(i)
	}
	wg.Wait()
}

// TestDeviceFlowExpiry verifies codes expire after timeout.
func TestDeviceFlowExpiry(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	w := formPost(mux, "/oauth/device/code", url.Values{"client_id": {"cli-exp"}})
	var cr map[string]interface{}
	json.NewDecoder(w.Body).Decode(&cr)
	dc := cr["device_code"].(string)

	// Manually expire
	h.mu.Lock()
	for _, c := range h.codes {
		c.ExpiresAt = time.Now().Add(-1 * time.Second)
	}
	h.mu.Unlock()

	w2 := formPost(mux, "/oauth/device/token", url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {dc},
	})
	var err map[string]string
	json.NewDecoder(w2.Body).Decode(&err)
	if err["error"] != "expired_token" {
		t.Fatalf("expected expired_token, got %s", err["error"])
	}
}

// TestDeviceFlowCleanup verifies expired codes are cleaned up.
func TestDeviceFlowCleanup(t *testing.T) {
	h := NewHandler(testIssuer)
	formPost(http.NewServeMux(), "/oauth/device/code", url.Values{"client_id": {"cli-1"}})

	// Add a code and expire it
	h.mu.Lock()
	h.codes["test-expired"] = &code{
		DeviceCode: "test-expired",
		UserCode:   "AAAA-BBBB",
		ExpiresAt:  time.Now().Add(-1 * time.Minute),
	}
	h.byUser["AAAA-BBBB"] = h.codes["test-expired"]
	h.mu.Unlock()

	h.Cleanup()

	h.mu.Lock()
	_, exists := h.codes["test-expired"]
	h.mu.Unlock()
	if exists {
		t.Fatal("expected expired code to be cleaned up")
	}
}

// TestInvalidGrantType verifies wrong grant_type is rejected.
func TestInvalidGrantType(t *testing.T) {
	h := NewHandler(testIssuer)
	mux := http.NewServeMux()
	h.Register(mux)

	w := formPost(mux, "/oauth/device/token", url.Values{
		"grant_type":  {"client_credentials"},
		"device_code": {"anything"},
	})
	var err map[string]string
	json.NewDecoder(w.Body).Decode(&err)
	if err["error"] != "unsupported_grant_type" {
		t.Fatalf("expected unsupported_grant_type, got %s", err["error"])
	}
}

func formPost(mux *http.ServeMux, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}
