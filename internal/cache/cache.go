// Package cache provides a simple TTL-based response cache for GET requests.
package cache

import (
	"sync"
	"time"
)

// Entry holds a cached response.
type Entry struct {
	Body       []byte
	StatusCode int
	Header     map[string]string
	ExpiresAt  time.Time
}

// Cache is a thread-safe in-memory response cache.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]*Entry
	ttl     time.Duration
	maxSize int
	stopCh  chan struct{}
}

// New creates a cache with the given TTL and max entries.
func New(ttl time.Duration, maxSize int) *Cache {
	c := &Cache{
		entries: make(map[string]*Entry),
		ttl:     ttl,
		maxSize: maxSize,
		stopCh:  make(chan struct{}),
	}
	go c.reap()
	return c
}

// Stop halts the background reaper goroutine.
func (c *Cache) Stop() { close(c.stopCh) }

// Get returns a cached entry if valid, or nil.
func (c *Cache) Get(key string) *Entry {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.ExpiresAt) {
		return nil
	}
	return e
}

// Set stores a response in the cache.
func (c *Cache) Set(key string, statusCode int, header map[string]string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.entries) >= c.maxSize {
		// Evict oldest
		var oldest string
		var oldestTime time.Time
		for k, v := range c.entries {
			if oldest == "" || v.ExpiresAt.Before(oldestTime) {
				oldest = k
				oldestTime = v.ExpiresAt
			}
		}
		delete(c.entries, oldest)
	}
	c.entries[key] = &Entry{
		Body:       body,
		StatusCode: statusCode,
		Header:     header,
		ExpiresAt:  time.Now().Add(c.ttl),
	}
}

// reap removes expired entries every TTL interval.
func (c *Cache) reap() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.entries {
				if now.After(v.ExpiresAt) {
					delete(c.entries, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}
