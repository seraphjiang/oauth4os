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

func TestConcurrentEmit(t *testing.T) {
	var count atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.Emit(Event{Type: "token_issued", ClientID: "svc-1", Timestamp: time.Now()})
		}()
	}
	wg.Wait()
	time.Sleep(200 * time.Millisecond) // let drain goroutine deliver

	if count.Load() == 0 {
		t.Fatal("expected at least some events delivered")
	}
}

func TestEventBody(t *testing.T) {
	var mu sync.Mutex
	var received Event
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e Event
		json.NewDecoder(r.Body).Decode(&e)
		mu.Lock()
		received = e
		mu.Unlock()
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	n.Emit(Event{
		Type:     "token_revoked",
		ClientID: "svc-1",
		Scopes:   []string{"admin"},
	})
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if received.Type != "token_revoked" || received.ClientID != "svc-1" {
		t.Fatalf("event body mismatch: %+v", received)
	}
}

func TestMultipleURLs(t *testing.T) {
	var count1, count2 atomic.Int64
	srv1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count1.Add(1)
		w.WriteHeader(200)
	}))
	defer srv1.Close()
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count2.Add(1)
		w.WriteHeader(200)
	}))
	defer srv2.Close()

	n := New([]string{srv1.URL, srv2.URL})
	n.Emit(Event{Type: "token_issued", ClientID: "svc-1"})
	time.Sleep(200 * time.Millisecond)

	if count1.Load() == 0 || count2.Load() == 0 {
		t.Fatalf("expected both URLs to receive event: srv1=%d srv2=%d", count1.Load(), count2.Load())
	}
}

func TestSlowWebhookDoesNotBlock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	n := New([]string{srv.URL})
	start := time.Now()
	n.Emit(Event{Type: "token_issued", ClientID: "svc-1"})
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Fatalf("Emit should be async, took %v", elapsed)
	}
}
