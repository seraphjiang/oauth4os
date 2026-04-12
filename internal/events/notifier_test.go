package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmitAndReceive(t *testing.T) {
	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		received <- e
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := NewNotifier([]string{srv.URL})
	n.Emit(Event{Type: TokenIssued, ClientID: "svc-a", TokenID: "tok-1"})

	select {
	case e := <-received:
		if e.Type != TokenIssued || e.ClientID != "svc-a" {
			t.Fatalf("unexpected event: %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for webhook")
	}
}

func TestNoURLsNoPanic(t *testing.T) {
	n := NewNotifier(nil)
	n.Emit(Event{Type: TokenRevoked, ClientID: "x"})
	// should not panic
}
