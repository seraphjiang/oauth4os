package resilience_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seraphjiang/oauth4os/internal/cache"
	"github.com/seraphjiang/oauth4os/internal/circuit"
	"github.com/seraphjiang/oauth4os/internal/retry"
)

// fakeUpstream simulates an upstream that fails N times then recovers.
type fakeUpstream struct {
	calls   atomic.Int64
	failFor int64
}

func (f *fakeUpstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n := f.calls.Add(1)
	if n <= f.failFor {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"hits":{"total":42}}`))
}

func (f *fakeUpstream) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	f.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// TestRetry_RecoversFromTransientFailure verifies retry handles brief 5xx.
func TestRetry_RecoversFromTransientFailure(t *testing.T) {
	upstream := &fakeUpstream{failFor: 2} // fail first 2, succeed on 3rd
	rt := &retry.Transport{Base: upstream, MaxRetries: 3, BaseDelay: time.Millisecond}

	req := httptest.NewRequest("GET", "/test/_search", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if upstream.calls.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", upstream.calls.Load())
	}
}

// TestRetry_ExhaustsRetries verifies retry gives up after max attempts.
func TestRetry_ExhaustsRetries(t *testing.T) {
	upstream := &fakeUpstream{failFor: 100} // always fail
	rt := &retry.Transport{Base: upstream, MaxRetries: 3, BaseDelay: time.Millisecond}

	req := httptest.NewRequest("GET", "/test/_search", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 502 {
		t.Fatalf("expected 502 after exhausting retries, got %d", resp.StatusCode)
	}
	if upstream.calls.Load() != 4 { // 1 initial + 3 retries
		t.Fatalf("expected 4 attempts, got %d", upstream.calls.Load())
	}
}

// TestCircuitBreaker_OpensAfterThreshold verifies circuit opens on consecutive 5xx.
func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	breaker := circuit.New(3, 50*time.Millisecond)

	// 3 consecutive failures should open the circuit
	for i := 0; i < 3; i++ {
		if !breaker.Allow() {
			t.Fatalf("circuit should be closed on attempt %d", i)
		}
		breaker.Record(500)
	}

	// Circuit should now be open
	if breaker.Allow() {
		t.Fatal("circuit should be open after 3 failures")
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Should be half-open — one probe allowed
	if !breaker.Allow() {
		t.Fatal("circuit should allow probe after cooldown")
	}

	// Success closes the circuit
	breaker.Record(200)
	if !breaker.Allow() {
		t.Fatal("circuit should be closed after successful probe")
	}
}

// TestCircuitBreaker_ResetsOnSuccess verifies failures reset on success.
func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	breaker := circuit.New(3, time.Second)

	breaker.Record(500)
	breaker.Record(500)
	breaker.Record(200) // resets counter
	breaker.Record(500)
	breaker.Record(500)

	// Should still be closed — counter reset after the 200
	if !breaker.Allow() {
		t.Fatal("circuit should be closed — failure counter was reset by success")
	}
}

// TestCache_ServesStaleOnUpstreamFailure simulates the full chain:
// 1. First request succeeds and is cached
// 2. Upstream starts failing
// 3. Cache serves the stale response
func TestCache_ServesStaleOnUpstreamFailure(t *testing.T) {
	c := cache.New(5*time.Second, 100)

	// Simulate: first request succeeds, cache it
	body := []byte(`{"hits":{"total":42}}`)
	c.Set("client:/logs/_search", 200, map[string]string{"Content-Type": "application/json"}, body)

	// Simulate: upstream is now failing, but cache has data
	cached := c.Get("client:/logs/_search")
	if cached == nil {
		t.Fatal("expected cache hit")
	}
	if cached.StatusCode != 200 {
		t.Fatalf("expected cached 200, got %d", cached.StatusCode)
	}
	if string(cached.Body) != string(body) {
		t.Fatalf("expected cached body, got %s", cached.Body)
	}
}

// TestFullChain_RetryThenCircuitThenCache tests the complete resilience chain.
func TestFullChain_RetryThenCircuitThenCache(t *testing.T) {
	c := cache.New(5*time.Second, 100)
	breaker := circuit.New(3, 50*time.Millisecond)

	// Phase 1: Upstream healthy — cache a response
	body := []byte(`{"hits":{"total":42}}`)
	c.Set("client:/logs/_search", 200, map[string]string{"Content-Type": "application/json"}, body)
	breaker.Record(200)

	// Phase 2: Upstream starts failing — retry exhausts, circuit opens
	upstream := &fakeUpstream{failFor: 100}
	rt := &retry.Transport{Base: upstream, MaxRetries: 2, BaseDelay: time.Millisecond}

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/logs/_search", nil)
		resp, _ := rt.RoundTrip(req)
		breaker.Record(resp.StatusCode)
	}

	// Circuit should be open (3 consecutive 502s from exhausted retries)
	if breaker.Allow() {
		t.Fatal("circuit should be open")
	}

	// Phase 3: Circuit is open — serve from cache
	cached := c.Get("client:/logs/_search")
	if cached == nil {
		t.Fatal("cache should serve stale response while circuit is open")
	}
	if cached.StatusCode != 200 {
		t.Fatalf("expected cached 200, got %d", cached.StatusCode)
	}

	// Phase 4: Wait for cooldown, upstream recovers
	time.Sleep(60 * time.Millisecond)
	upstream.failFor = 0 // recover
	upstream.calls.Store(0)

	if !breaker.Allow() {
		t.Fatal("circuit should allow probe after cooldown")
	}

	req := httptest.NewRequest("GET", "/logs/_search", nil)
	resp, _ := rt.RoundTrip(req)
	breaker.Record(resp.StatusCode)

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 after recovery, got %d", resp.StatusCode)
	}

	// Circuit should be closed again
	if !breaker.Allow() {
		t.Fatal("circuit should be closed after successful probe")
	}
}
