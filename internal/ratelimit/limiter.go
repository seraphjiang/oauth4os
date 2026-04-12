// Package ratelimit implements per-client token bucket rate limiting.
package ratelimit

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Bucket tracks tokens for a single client.
type Bucket struct {
	tokens    float64
	capacity  float64
	rate      float64 // tokens per second
	lastFill  time.Time
	mu        sync.Mutex
}

func (b *Bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastFill = now
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

func (b *Bucket) retryAfter() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.rate <= 0 {
		return 60
	}
	return int((1 - b.tokens) / b.rate) + 1
}

// status returns (limit, remaining, resetUnix) for rate limit headers.
func (b *Bucket) status() (int, int, int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	limit := int(b.capacity)
	remaining := int(b.tokens)
	if remaining < 0 {
		remaining = 0
	}
	// Reset = when bucket fully refills
	secsToFull := 0.0
	if b.rate > 0 {
		secsToFull = (b.capacity - b.tokens) / b.rate
	}
	reset := time.Now().Add(time.Duration(secsToFull) * time.Second).Unix()
	return limit, remaining, reset
}

// Limiter manages per-client rate limit buckets.
type Limiter struct {
	limits  map[string]int // scope → requests_per_minute
	buckets sync.Map       // clientID → *Bucket
	defaultRPM int
}

// New creates a Limiter from scope→RPM config. defaultRPM applies when no scope matches.
func New(limits map[string]int, defaultRPM int) *Limiter {
	if defaultRPM <= 0 {
		defaultRPM = 600
	}
	return &Limiter{limits: limits, defaultRPM: defaultRPM}
}

// Allow checks if a request from clientID with given scopes is allowed.
func (l *Limiter) Allow(clientID string, scopes []string) bool {
	rpm := l.resolveRPM(scopes)
	bucket := l.getBucket(clientID, rpm)
	return bucket.allow()
}

// RetryAfter returns seconds until the client can retry.
func (l *Limiter) RetryAfter(clientID string) int {
	if v, ok := l.buckets.Load(clientID); ok {
		return v.(*Bucket).retryAfter()
	}
	return 1
}

func (l *Limiter) resolveRPM(scopes []string) int {
	// Use the lowest (most restrictive) matching limit
	rpm := 0
	for _, s := range scopes {
		if r, ok := l.limits[s]; ok {
			if rpm == 0 || r < rpm {
				rpm = r
			}
		}
	}
	if rpm == 0 {
		return l.defaultRPM
	}
	return rpm
}

func (l *Limiter) getBucket(clientID string, rpm int) *Bucket {
	if v, ok := l.buckets.Load(clientID); ok {
		return v.(*Bucket)
	}
	rate := float64(rpm) / 60.0
	capacity := float64(rpm) // burst = 1 minute worth
	b := &Bucket{tokens: capacity, capacity: capacity, rate: rate, lastFill: time.Now()}
	actual, _ := l.buckets.LoadOrStore(clientID, b)
	return actual.(*Bucket)
}

// Middleware returns an http.Handler that enforces rate limits.
// extractClient returns (clientID, scopes) from the request; return ("", nil) to skip.
func (l *Limiter) Middleware(next http.Handler, extractClient func(r *http.Request) (string, []string)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID, scopes := extractClient(r)
		if clientID == "" {
			next.ServeHTTP(w, r)
			return
		}
		rpm := l.resolveRPM(scopes)
		bucket := l.getBucket(clientID, rpm)
		if !bucket.allow() {
			retry := bucket.retryAfter()
			limit, remaining, reset := bucket.status()
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
			w.Header().Set("Retry-After", strconv.Itoa(retry))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, `{"error":"rate_limit_exceeded","retry_after":%d}`, retry)
			return
		}
		limit, remaining, reset := bucket.status()
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
		next.ServeHTTP(w, r)
	})
}
