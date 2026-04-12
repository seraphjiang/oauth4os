package registration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// helper: create a mux wired to all handler methods
func testMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /oauth/register", h.Register)
	mux.HandleFunc("GET /oauth/register", h.List)
	mux.HandleFunc("GET /oauth/register/{client_id}", h.Get)
	mux.HandleFunc("PUT /oauth/register/{client_id}", h.Update)
	mux.HandleFunc("DELETE /oauth/register/{client_id}", h.Delete)
	mux.HandleFunc("POST /oauth/register/{client_id}/rotate", h.RotateSecret)
	return mux
}

func noopRegistrar(id, secret string, scopes, redirectURIs []string) {}

func registerApp(t *testing.T, mux *http.ServeMux, name, scope string) Response {
	t.Helper()
	body, _ := json.Marshal(Request{ClientName: name, Scope: scope})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/oauth/register", bytes.NewReader(body)))
	if w.Code != 201 {
		t.Fatalf("register %s: expected 201, got %d: %s", name, w.Code, w.Body.String())
	}
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	return resp
}

// Full developer portal flow: register → list → rotate → delete
func TestDevPortalFlow(t *testing.T) {
	h := NewHandler(noopRegistrar, nil)
	mux := testMux(h)

	// Register
	app := registerApp(t, mux, "my-dashboard", "read:logs-*")
	if app.ClientID == "" || app.ClientSecret == "" {
		t.Fatal("register should return client_id and client_secret")
	}
	origSecret := app.ClientSecret

	// List — should contain our app
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register", nil))
	if w.Code != 200 {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	var list []Response
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("list: expected 1 app, got %d", len(list))
	}
	if list[0].ClientSecret != "" {
		t.Fatal("list should redact secrets")
	}

	// Rotate secret
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/oauth/register/"+app.ClientID+"/rotate", nil))
	if w.Code != 200 {
		t.Fatalf("rotate: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var rotated map[string]string
	json.NewDecoder(w.Body).Decode(&rotated)
	if rotated["client_secret"] == "" {
		t.Fatal("rotate should return new secret")
	}
	if rotated["client_secret"] == origSecret {
		t.Fatal("rotated secret should differ from original")
	}

	// Delete
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/oauth/register/"+app.ClientID, nil))
	if w.Code != 204 {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Verify deleted — GET returns 404
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register/"+app.ClientID, nil))
	if w.Code != 404 {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}

	// List should be empty
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register", nil))
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("list after delete: expected 0, got %d", len(list))
	}
}

// Multiple apps — register 3, list all, delete one, verify count
func TestDevPortalMultipleApps(t *testing.T) {
	h := NewHandler(noopRegistrar, nil)
	mux := testMux(h)

	a1 := registerApp(t, mux, "app-1", "read:logs-*")
	registerApp(t, mux, "app-2", "admin")
	registerApp(t, mux, "app-3", "read:logs-* write:logs-*")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register", nil))
	var list []Response
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(list))
	}

	// Delete first
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/oauth/register/"+a1.ClientID, nil))
	if w.Code != 204 {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register", nil))
	json.NewDecoder(w.Body).Decode(&list)
	if len(list) != 2 {
		t.Fatalf("expected 2 apps after delete, got %d", len(list))
	}
}

// Rotate on nonexistent client → 404
func TestRotateNonexistent(t *testing.T) {
	h := NewHandler(noopRegistrar, nil)
	mux := testMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("POST", "/oauth/register/client_bogus/rotate", nil))
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// Delete nonexistent → 404
func TestDeleteNonexistent(t *testing.T) {
	h := NewHandler(noopRegistrar, nil)
	mux := testMux(h)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("DELETE", "/oauth/register/client_bogus", nil))
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// Persistence: register on handler A, create handler B with same state, verify client exists.
// Simulates proxy restart by transferring the in-memory map.
func TestPersistenceAcrossRestart(t *testing.T) {
	h1 := NewHandler(noopRegistrar, nil)
	mux1 := testMux(h1)

	app := registerApp(t, mux1, "persistent-app", "admin")

	// Simulate restart: new handler, copy clients from old
	h2 := NewHandler(noopRegistrar, nil)
	h1.mu.RLock()
	for k, v := range h1.clients {
		h2.clients[k] = v
	}
	h1.mu.RUnlock()

	mux2 := testMux(h2)

	// Verify client survives "restart"
	w := httptest.NewRecorder()
	mux2.ServeHTTP(w, httptest.NewRequest("GET", "/oauth/register/"+app.ClientID, nil))
	if w.Code != 200 {
		t.Fatalf("get after restart: expected 200, got %d", w.Code)
	}
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ClientName != "persistent-app" {
		t.Fatalf("expected persistent-app, got %s", resp.ClientName)
	}

	// Rotate should still work
	w = httptest.NewRecorder()
	mux2.ServeHTTP(w, httptest.NewRequest("POST", "/oauth/register/"+app.ClientID+"/rotate", nil))
	if w.Code != 200 {
		t.Fatalf("rotate after restart: expected 200, got %d", w.Code)
	}
}

// Update flow: register → update name → verify
func TestUpdateClient(t *testing.T) {
	h := NewHandler(noopRegistrar, nil)
	mux := testMux(h)

	app := registerApp(t, mux, "old-name", "read:logs-*")

	body, _ := json.Marshal(Request{ClientName: "new-name"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("PUT", "/oauth/register/"+app.ClientID, bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("update: expected 200, got %d", w.Code)
	}
	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.ClientName != "new-name" {
		t.Fatalf("expected new-name, got %s", resp.ClientName)
	}
	if resp.ClientSecret != "" {
		t.Fatal("update response should not expose secret")
	}
}
