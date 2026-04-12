package keyring

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// Mutation: remove key generation → Current must return a valid key
func TestMutation_CurrentKey(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k == nil || k.Private == nil {
		t.Error("Current must return a valid key")
	}
	if k.ID == "" {
		t.Error("key must have a KID")
	}
}

// Mutation: remove JWKS handler → must return valid JWKS JSON
func TestMutation_JWKSHandler(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	h := r.JWKSHandler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/jwks.json", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var jwks struct {
		Keys []json.RawMessage `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &jwks); err != nil {
		t.Fatalf("invalid JWKS JSON: %v", err)
	}
	if len(jwks.Keys) == 0 {
		t.Error("JWKS must contain at least one key")
	}
}

// Mutation: remove KID generation → different keys must have different KIDs
func TestMutation_UniqueKIDs(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k1 := r.Current()
	r.rotate()
	k2 := r.Current()
	if k1.ID == k2.ID {
		t.Error("rotated keys must have different KIDs")
	}
}

// Mutation: remove Stop → rotation goroutine must terminate
func TestMutation_StopTerminates(t *testing.T) {
	r, err := New(2048, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		r.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must terminate the rotation goroutine")
	}
}
