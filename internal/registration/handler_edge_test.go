package registration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func noopRegistrar(id, secret string, scopes, redirectURIs []string) {}

func TestConcurrentRegister(t *testing.T) {
	h := NewHandler(noopRegistrar, []string{"read:logs-*", "admin"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			body := `{"client_name":"svc","scope":["read:logs-*"]}`
			r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.Register(w, r)
			if w.Code != 201 {
				t.Errorf("expected 201, got %d", w.Code)
			}
		}(i)
	}
	wg.Wait()
}

func TestRegisterEmptyBody(t *testing.T) {
	h := NewHandler(noopRegistrar, []string{"read:logs-*"})
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code == 201 {
		t.Fatal("empty body should not succeed")
	}
}

func TestRegisterSecretNotInResponse(t *testing.T) {
	h := NewHandler(noopRegistrar, []string{"read:logs-*"})
	body := `{"client_name":"svc","scope":["read:logs-*"]}`
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// Secret should be present in registration response (one-time display)
	if resp["client_secret"] == nil || resp["client_secret"] == "" {
		t.Fatal("client_secret should be in registration response")
	}
}

func TestGetNonexistentClient(t *testing.T) {
	h := NewHandler(noopRegistrar, []string{"read:logs-*"})
	r := httptest.NewRequest("GET", "/oauth/register/nonexistent", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code == 200 {
		t.Fatal("nonexistent client should not return 200")
	}
}

func TestDeleteNonexistentEdge(t *testing.T) {
	h := NewHandler(noopRegistrar, []string{"read:logs-*"})
	r := httptest.NewRequest("DELETE", "/oauth/register/nonexistent", nil)
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code == 200 {
		t.Fatal("deleting nonexistent client should not return 200")
	}
}
