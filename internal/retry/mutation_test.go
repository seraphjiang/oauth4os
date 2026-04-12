package retry

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type statusRT struct {
	calls    atomic.Int64
	statuses []int
}

func (s *statusRT) RoundTrip(req *http.Request) (*http.Response, error) {
	n := int(s.calls.Add(1)) - 1
	code := s.statuses[n%len(s.statuses)]
	return &http.Response{StatusCode: code, Body: http.NoBody}, nil
}

// Mutation: remove < 500 check → 4xx must not be retried
func TestMutation_No4xxRetry(t *testing.T) {
	rt := &statusRT{statuses: []int{429}}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: time.Millisecond}
	resp, _ := tr.RoundTrip(&http.Request{})
	if resp.StatusCode != 429 {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}
	if rt.calls.Load() != 1 {
		t.Errorf("4xx must not retry, got %d calls", rt.calls.Load())
	}
}

// Mutation: remove backoff cap → delay must not exceed 5s
func TestMutation_BackoffCap(t *testing.T) {
	rt := &statusRT{statuses: []int{500, 500, 500, 200}}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: 100 * time.Millisecond}
	start := time.Now()
	tr.RoundTrip(&http.Request{})
	elapsed := time.Since(start)
	// Delays: 100ms, 200ms, 400ms = 700ms total. All under 5s cap.
	// If cap were broken with huge base, it would take much longer.
	if elapsed > 2*time.Second {
		t.Errorf("backoff took too long: %v", elapsed)
	}
}

// Mutation: remove attempt > 0 guard → first attempt must not sleep
func TestMutation_NoSleepOnFirst(t *testing.T) {
	rt := &statusRT{statuses: []int{200}}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: 1 * time.Second}
	start := time.Now()
	tr.RoundTrip(&http.Request{})
	if time.Since(start) > 100*time.Millisecond {
		t.Error("first attempt must not sleep")
	}
}

// Mutation: change maxRetries default → 0 retries must default to 3
func TestMutation_DefaultRetries(t *testing.T) {
	rt := &statusRT{statuses: []int{500, 500, 500, 200}}
	tr := &Transport{Base: rt, MaxRetries: 0, BaseDelay: time.Millisecond}
	resp, _ := tr.RoundTrip(&http.Request{})
	if resp.StatusCode != 200 {
		t.Errorf("default retries should recover, got %d", resp.StatusCode)
	}
	if rt.calls.Load() != 4 {
		t.Errorf("expected 4 calls (1+3 retries), got %d", rt.calls.Load())
	}
}
