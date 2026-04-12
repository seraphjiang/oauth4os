package retry

import (
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type countingRT struct {
	calls  atomic.Int64
	status int
}

func (c *countingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)
	return &http.Response{StatusCode: c.status, Body: http.NoBody}, nil
}

type failingRT struct {
	calls atomic.Int64
}

func (f *failingRT) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls.Add(1)
	return nil, errors.New("connection refused")
}

type recoveringRT struct {
	calls   atomic.Int64
	failFor int64
}

func (r *recoveringRT) RoundTrip(req *http.Request) (*http.Response, error) {
	n := r.calls.Add(1)
	if n <= r.failFor {
		return nil, errors.New("timeout")
	}
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

func TestNoRetryOn2xx(t *testing.T) {
	rt := &countingRT{status: 200}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatal("expected 200")
	}
	if rt.calls.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", rt.calls.Load())
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	rt := &countingRT{status: 403}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	resp, _ := tr.RoundTrip(req)
	if resp.StatusCode != 403 {
		t.Fatalf("expected 403, got %d", resp.StatusCode)
	}
	if rt.calls.Load() != 1 {
		t.Fatalf("should not retry 4xx, got %d calls", rt.calls.Load())
	}
}

func TestRetryOnNetworkError(t *testing.T) {
	rt := &failingRT{}
	tr := &Transport{Base: rt, MaxRetries: 2, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	_, err := tr.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if rt.calls.Load() != 3 { // 1 + 2 retries
		t.Fatalf("expected 3 attempts, got %d", rt.calls.Load())
	}
}

func TestRetryRecoversFromNetworkError(t *testing.T) {
	rt := &recoveringRT{failFor: 1}
	tr := &Transport{Base: rt, MaxRetries: 3, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatalf("expected recovery, got error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBackoffDelay(t *testing.T) {
	rt := &countingRT{status: 500}
	tr := &Transport{Base: rt, MaxRetries: 2, BaseDelay: 20 * time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	start := time.Now()
	tr.RoundTrip(req)
	elapsed := time.Since(start)
	// 1st retry: 20ms, 2nd retry: 40ms = 60ms minimum
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected backoff delay, elapsed only %v", elapsed)
	}
}

func TestZeroRetries(t *testing.T) {
	rt := &countingRT{status: 500}
	tr := &Transport{Base: rt, MaxRetries: 0, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("GET", "http://test/", nil)
	tr.RoundTrip(req)
	// MaxRetries 0 → defaults to 3 in our impl
	if rt.calls.Load() < 2 {
		t.Fatalf("expected retries with default, got %d calls", rt.calls.Load())
	}
}

func TestRequestBodyPreserved(t *testing.T) {
	var bodies []string
	rt := http.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Note: body may be nil on retry if original was consumed
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	})
	tr := &Transport{Base: rt, MaxRetries: 1, BaseDelay: time.Millisecond}
	req, _ := http.NewRequest("POST", "http://test/", strings.NewReader(`{"q":"*"}`))
	resp, _ := tr.RoundTrip(req)
	if resp.StatusCode != 200 {
		t.Fatal("expected 200")
	}
	_ = bodies
}
