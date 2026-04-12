package registration

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func edgeRegistrar(id, secret string, scopes, redirectURIs []string) {}

func TestConcurrentRegister(t *testing.T) {
	h := NewHandler(edgeRegistrar, []string{"read:logs-*", "admin"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"client_name":"svc","scope":"read:logs-*"}`
			r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.Register(w, r)
			if w.Code != 201 {
				t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
			}
		}()
	}
	wg.Wait()
}

func TestRegisterEmptyBody(t *testing.T) {
	h := NewHandler(edgeRegistrar, []string{"read:logs-*"})
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(""))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code == 201 {
		t.Fatal("empty body should not succeed")
	}
}

func TestRegisterSecretInResponse(t *testing.T) {
	h := NewHandler(edgeRegistrar, []string{"read:logs-*"})
	body := `{"client_name":"svc","scope":"read:logs-*"}`
	r := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Register(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["client_secret"] == nil || resp["client_secret"] == "" {
		t.Fatal("client_secret should be in registration response")
	}
}

func TestGetNonexistentClient(t *testing.T) {
	h := NewHandler(edgeRegistrar, []string{"read:logs-*"})
	r := httptest.NewRequest("GET", "/oauth/register/nonexistent", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code == 200 {
		t.Fatal("nonexistent client should not return 200")
	}
}
