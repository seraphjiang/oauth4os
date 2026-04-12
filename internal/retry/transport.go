// Package retry provides an HTTP round-tripper with exponential backoff for 5xx responses.
package retry

import (
	"math"
	"net/http"
	"time"
)

// Transport wraps an http.RoundTripper with retry logic for 5xx responses.
type Transport struct {
	Base       http.RoundTripper
	MaxRetries int           // default 3
	BaseDelay  time.Duration // default 100ms
}

// RoundTrip executes the request with retries on 5xx.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	maxRetries := t.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	baseDelay := t.BaseDelay
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond
	}

	var resp *http.Response
	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}
			time.Sleep(delay)
		}
		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			continue // network error — retry
		}
		if resp.StatusCode < 500 {
			return resp, nil // success or client error — don't retry
		}
		// 5xx — retry (drain body to reuse connection)
		if attempt < maxRetries {
			resp.Body.Close()
		}
	}
	return resp, err
}
