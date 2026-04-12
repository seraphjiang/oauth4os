package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEmitDelivers(t *testing.T) {
	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		received <- e
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	n.Emit(Event{Type: TokenIssued, ClientID: "test"})

	select {
	case e := <-received:
		if e.Type != TokenIssued || e.ClientID != "test" {
			t.Errorf("unexpected event: %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event not delivered")
	}
}

func TestEmitNoURLsNoOp(t *testing.T) {
	n := New(nil)
	n.Emit(Event{Type: TokenRevoked, ClientID: "x"}) // should not panic
}

func TestEmitDropsWhenFull(t *testing.T) {
	n := &Notifier{urls: []string{"http://unreachable.invalid"}, ch: make(chan Event, 1)}
	n.Emit(Event{Type: TokenIssued})
	n.Emit(Event{Type: TokenIssued}) // should drop, not block
}
