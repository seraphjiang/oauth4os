package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Mutation: remove no-op for empty URLs → Emit with no URLs must not send
func TestMutation_NoURLsNoop(t *testing.T) {
	n := New(nil)
	// Should not panic or block
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
}

// Mutation: remove timestamp assignment → events must have timestamp
func TestMutation_TimestampSet(t *testing.T) {
	var mu sync.Mutex
	var got Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	defer srv.Close()
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		mu.Lock()
		got = e
		mu.Unlock()
	}))
	defer srv.Close()
	n := New([]string{srv.URL})
	before := time.Now()
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	ts := got.Timestamp
	mu.Unlock()
	if ts.Before(before) {
		t.Error("event timestamp must be set on Emit")
	}
}

// Mutation: remove drop on full → must not block when buffer full
func TestMutation_NonBlocking(t *testing.T) {
	// Slow server that never responds quickly
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	defer srv.Close()
		time.Sleep(10 * time.Second)
	}))
	defer srv.Close()
	n := New([]string{srv.URL})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 200; i++ { // more than buffer size (100)
			n.Emit(Event{Type: TokenIssued, ClientID: "app"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Emit must not block when buffer is full")
	}
}

// Mutation: remove Content-Type header → webhook must receive JSON content type
func TestMutation_ContentType(t *testing.T) {
	var gotCT atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	defer srv.Close()
		gotCT.Store(r.Header.Get("Content-Type"))
	}))
	defer srv.Close()
	n := New([]string{srv.URL})
	n.Emit(Event{Type: TokenRevoked, ClientID: "app"})
	time.Sleep(100 * time.Millisecond)
	if ct, _ := gotCT.Load().(string); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}
