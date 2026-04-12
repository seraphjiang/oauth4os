package par

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func postForm(mux *http.ServeMux, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// Mutation: remove one-time use delete → Resolve must consume the request
func TestMutation_OneTimeUse(t *testing.T) {
	h := NewHandler(nil)
	h.mu.Lock()
	h.requests["urn:test:1"] = &request{ClientID: "app", ExpiresAt: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	_, _, _, _, _, _, ok := h.Resolve("urn:test:1")
	if !ok {
		t.Fatal("first resolve should succeed")
	}
	_, _, _, _, _, _, ok2 := h.Resolve("urn:test:1")
	if ok2 {
		t.Error("second resolve must fail — request_uri is one-time use")
	}
}

// Mutation: remove expiry check → expired requests must not resolve
func TestMutation_Expiry(t *testing.T) {
	h := NewHandler(nil)
	h.mu.Lock()
	h.requests["urn:test:2"] = &request{ClientID: "app", ExpiresAt: time.Now().Add(-time.Second)}
	h.mu.Unlock()

	_, _, _, _, _, _, ok := h.Resolve("urn:test:2")
	if ok {
		t.Error("expired request must not resolve")
	}
}

// Mutation: remove client_id check → empty client_id must be rejected
func TestMutation_ClientIDRequired(t *testing.T) {
	h := NewHandler(nil)
	mux := setupMux(h)
	w := postForm(mux, "/oauth/par", "scope=read")
	if w.Code != 400 {
		t.Errorf("missing client_id should return 400, got %d", w.Code)
	}
}

// Mutation: change 201 to 200 → PAR must return 201 Created
func TestMutation_201Status(t *testing.T) {
	h := NewHandler(nil)
	mux := setupMux(h)
	w := postForm(mux, "/oauth/par", "client_id=app&redirect_uri=http://localhost/cb")
	if w.Code != 201 {
		t.Errorf("PAR push must return 201, got %d", w.Code)
	}
}

// Mutation: remove Cache-Control header
func TestMutation_CacheControl(t *testing.T) {
	h := NewHandler(nil)
	mux := setupMux(h)
	w := postForm(mux, "/oauth/par", "client_id=app&redirect_uri=http://localhost/cb")
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Error("PAR response must have Cache-Control: no-store")
	}
}

// Mutation: Cleanup must remove expired entries
func TestMutation_Cleanup(t *testing.T) {
	h := NewHandler(nil)
	h.mu.Lock()
	h.requests["urn:expired"] = &request{ExpiresAt: time.Now().Add(-time.Second)}
	h.requests["urn:valid"] = &request{ExpiresAt: time.Now().Add(time.Minute)}
	h.mu.Unlock()

	h.Cleanup()

	h.mu.Lock()
	_, expiredExists := h.requests["urn:expired"]
	_, validExists := h.requests["urn:valid"]
	h.mu.Unlock()

	if expiredExists {
		t.Error("Cleanup must remove expired entries")
	}
	if !validExists {
		t.Error("Cleanup must keep valid entries")
	}
}

// Mutation: remove request_uri prefix → must start with urn:ietf:params:oauth:request_uri:
func TestMutation_RequestURIFormat(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)
	body := "client_id=app&redirect_uri=http://localhost/cb&response_type=code&scope=openid"
	r := httptest.NewRequest("POST", "/par", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == 201 && !strings.Contains(w.Body.String(), "urn:ietf:params:oauth:request_uri:") {
		t.Error("request_uri must use urn:ietf:params:oauth:request_uri: prefix")
	}
}
