package retry

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

type countTransport struct {
	calls    atomic.Int32
	statuses []int
}

func (t *countTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := int(t.calls.Add(1)) - 1
	status := 500
	if idx < len(t.statuses) {
		status = t.statuses[idx]
	}
	return &http.Response{StatusCode: status, Body: http.NoBody}, nil
}

func TestNoRetryOn200(t *testing.T) {
	ct := &countTransport{statuses: []int{200}}
	tr := &Transport{Base: ct, MaxRetries: 3, BaseDelay: 1 * time.Millisecond}
	resp, err := tr.RoundTrip(&http.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if ct.calls.Load() != 1 {
		t.Errorf("expected 1 call, got %d", ct.calls.Load())
	}
}

func TestRetryOn500(t *testing.T) {
	ct := &countTransport{statuses: []int{500, 500, 200}}
	tr := &Transport{Base: ct, MaxRetries: 3, BaseDelay: 1 * time.Millisecond}
	resp, _ := tr.RoundTrip(&http.Request{})
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if ct.calls.Load() != 3 {
		t.Errorf("expected 3 calls, got %d", ct.calls.Load())
	}
}

func TestMaxRetriesExhausted(t *testing.T) {
	ct := &countTransport{statuses: []int{502, 502, 502, 502}}
	tr := &Transport{Base: ct, MaxRetries: 2, BaseDelay: 1 * time.Millisecond}
	resp, _ := tr.RoundTrip(&http.Request{})
	if resp.StatusCode != 502 {
		t.Errorf("expected 502 after exhausting retries, got %d", resp.StatusCode)
	}
	if ct.calls.Load() != 3 { // 1 initial + 2 retries
		t.Errorf("expected 3 calls (1+2 retries), got %d", ct.calls.Load())
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	ct := &countTransport{statuses: []int{403}}
	tr := &Transport{Base: ct, MaxRetries: 3, BaseDelay: 1 * time.Millisecond}
	resp, _ := tr.RoundTrip(&http.Request{})
	if resp.StatusCode != 403 {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
	if ct.calls.Load() != 1 {
		t.Errorf("expected 1 call for 4xx, got %d", ct.calls.Load())
	}
}
