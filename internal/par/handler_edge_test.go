package par

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func post(mux *http.ServeMux, path string, vals url.Values) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestOneTimeUse(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	w := post(mux, "/oauth/par", url.Values{"client_id": {"app-1"}, "scope": {"read:logs-*"}})
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	uri := resp["request_uri"].(string)

	// First resolve succeeds
	_, _, _, _, _, _, ok := h.Resolve(uri)
	if !ok {
		t.Fatal("first resolve should succeed")
	}
	// Second resolve fails
	_, _, _, _, _, _, ok2 := h.Resolve(uri)
	if ok2 {
		t.Fatal("second resolve should fail (one-time use)")
	}
}

func TestExpiry(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	w := post(mux, "/oauth/par", url.Values{"client_id": {"app-1"}})
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	uri := resp["request_uri"].(string)

	// Expire it
	h.mu.Lock()
	h.requests[uri].ExpiresAt = time.Now().Add(-1 * time.Second)
	h.mu.Unlock()

	_, _, _, _, _, _, ok := h.Resolve(uri)
	if ok {
		t.Fatal("expired request should not resolve")
	}
}

func TestCleanup(t *testing.T) {
	h := NewHandler(nil)
	h.mu.Lock()
	h.requests["urn:test:expired"] = &request{ExpiresAt: time.Now().Add(-1 * time.Minute)}
	h.requests["urn:test:valid"] = &request{ExpiresAt: time.Now().Add(5 * time.Minute)}
	h.mu.Unlock()

	h.Cleanup()

	h.mu.Lock()
	_, expired := h.requests["urn:test:expired"]
	_, valid := h.requests["urn:test:valid"]
	h.mu.Unlock()

	if expired {
		t.Fatal("expired request should be cleaned up")
	}
	if !valid {
		t.Fatal("valid request should remain")
	}
}

func TestConcurrentPush(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	var wg sync.WaitGroup
	uris := make([]string, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			w := post(mux, "/oauth/par", url.Values{"client_id": {fmt.Sprintf("app-%d", n)}})
			var resp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&resp)
			uris[n] = resp["request_uri"].(string)
		}(i)
	}
	wg.Wait()

	// All URIs should be unique and resolvable
	seen := map[string]bool{}
	for _, uri := range uris {
		if uri == "" {
			t.Fatal("empty URI")
		}
		if seen[uri] {
			t.Fatal("duplicate URI")
		}
		seen[uri] = true
	}
}

func TestPKCEParams(t *testing.T) {
	h := NewHandler(nil)
	mux := http.NewServeMux()
	h.Register(mux)

	w := post(mux, "/oauth/par", url.Values{
		"client_id":             {"app-1"},
		"code_challenge":        {"challenge123"},
		"code_challenge_method": {"S256"},
		"redirect_uri":         {"https://app.example.com/cb"},
		"state":                 {"state-xyz"},
	})
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	_, _, redirect, state, cc, cm, ok := h.Resolve(resp["request_uri"].(string))
	if !ok {
		t.Fatal("resolve failed")
	}
	if redirect != "https://app.example.com/cb" || state != "state-xyz" || cc != "challenge123" || cm != "S256" {
		t.Fatalf("params mismatch: redirect=%s state=%s cc=%s cm=%s", redirect, state, cc, cm)
	}
}
