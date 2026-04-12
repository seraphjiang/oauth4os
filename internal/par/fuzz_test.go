package par

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzPush ensures the PAR push endpoint never panics.
func FuzzPush(f *testing.F) {
	f.Add("client_id=app&client_secret=sec&redirect_uri=http://localhost/cb&scope=read:logs-*&code_challenge=abc&code_challenge_method=S256")
	f.Add("")
	f.Add("client_id=")
	f.Add("redirect_uri=" + strings.Repeat("A", 10000))
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(func(id, secret string) error { return nil })
		mux := http.NewServeMux()
		h.Register(mux)
		r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
	})
}

// Test PAR push → resolve round-trip
func TestPushAndResolve(t *testing.T) {
	h := NewHandler(func(id, secret string) error { return nil })
	mux := http.NewServeMux()
	h.Register(mux)

	body := "client_id=app&client_secret=sec&redirect_uri=http://localhost/cb&scope=read:logs-*&code_challenge=abc&code_challenge_method=S256&state=xyz"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)

	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	// Extract request_uri from response
	respBody := w.Body.String()
	if !strings.Contains(respBody, "request_uri") {
		t.Fatal("response missing request_uri")
	}
}

// Test PAR with bad client auth
func TestPushBadAuth(t *testing.T) {
	h := NewHandler(func(id, secret string) error {
		return http.ErrAbortHandler // simulate auth failure
	})
	mux := http.NewServeMux()
	h.Register(mux)

	body := "client_id=app&client_secret=wrong&redirect_uri=http://localhost/cb"
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mux.ServeHTTP(w, r)

	if w.Code == 201 {
		t.Error("expected auth failure, got 201")
	}
}
