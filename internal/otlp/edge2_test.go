package otlp

import (
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentRecord(t *testing.T) {
	e := New(100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Record("op", time.Now().Add(-time.Millisecond), time.Now(), nil, "")
		}()
	}
	wg.Wait()
}

func TestEdge_RingBufferOverflow(t *testing.T) {
	e := New(5)
	for i := 0; i < 20; i++ {
		e.Record("op", time.Now().Add(-time.Millisecond), time.Now(), nil, "")
	}
	// Should not panic — ring buffer wraps
}

func TestEdge_RecordWithError(t *testing.T) {
	e := New(10)
	e.Record("fail-op", time.Now().Add(-time.Second), time.Now(), nil, "connection refused")
	// Should not panic
}

func TestEdge_ZeroCapacityNoPanic(t *testing.T) {
	e := New(0)
	e.Record("op", time.Now(), time.Now(), nil, "")
	// must not panic
}

func TestEdge_ConcurrentHandler(t *testing.T) {
	e := New(100)
	h := e.Handler()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
		}()
	}
	wg.Wait()
}
