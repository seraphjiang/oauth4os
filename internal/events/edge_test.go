package events

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestMultipleWebhookURLs(t *testing.T) {
	var count atomic.Int64
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(200)
	})
	srv1 := httptest.NewServer(handler)
	defer srv1.Close()
	srv2 := httptest.NewServer(handler)
	defer srv2.Close()

	n := New([]string{srv1.URL, srv2.URL})
	n.Emit(Event{Type: ClientReg, ClientID: "app"})
	time.Sleep(500 * time.Millisecond)

	if c := count.Load(); c != 2 {
		t.Fatalf("expected 2 deliveries (one per URL), got %d", c)
	}
}

func TestTimestampAutoSet(t *testing.T) {
	received := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		received <- e
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	n.Emit(Event{Type: TokenRevoked, ClientID: "c"})

	select {
	case e := <-received:
		if e.Timestamp.IsZero() {
			t.Fatal("timestamp should be auto-set")
		}
		if time.Since(e.Timestamp) > 5*time.Second {
			t.Fatal("timestamp should be recent")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestContentTypeJSON(t *testing.T) {
	received := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Header.Get("Content-Type")
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	n.Emit(Event{Type: TokenRefresh, ClientID: "c"})

	select {
	case ct := <-received:
		if ct != "application/json" {
			t.Fatalf("expected application/json, got %q", ct)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestFailedDeliveryDoesNotBlock(t *testing.T) {
	// Webhook URL that always fails
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
			n.Emit(Event{Type: TokenIssued, ClientID: "c"})
		}
		close(done)
	}()

	select {
	case <-done:
		// good — didn't block
	case <-time.After(3 * time.Second):
		t.Fatal("Emit blocked on failed delivery")
	}
}

func TestAllEventTypes(t *testing.T) {
	types := []EventType{TokenIssued, TokenRevoked, TokenRefresh, ClientReg, ClientDel}
	received := make(chan EventType, len(types))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		received <- e.Type
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	for _, et := range types {
		n.Emit(Event{Type: et, ClientID: "c"})
	}
	time.Sleep(500 * time.Millisecond)

	got := make(map[EventType]bool)
	for len(got) < len(types) {
		select {
		case et := <-received:
			got[et] = true
		default:
			break
		}
		if len(got) >= len(types) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	for _, et := range types {
		if !got[et] {
			t.Errorf("missing event type: %s", et)
		}
	}
}
