package otlp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRecordAndHandler(t *testing.T) {
	e := New(10)
	now := time.Now()
	e.Record("test-span", now, now.Add(50*time.Millisecond), map[string]string{"http.method": "GET"}, "")

	w := httptest.NewRecorder()
	e.Handler()(w, httptest.NewRequest("GET", "/v1/traces", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	rs, ok := resp["resourceSpans"].([]interface{})
	if !ok || len(rs) == 0 {
		t.Fatal("expected resourceSpans")
	}
}

func TestRingBuffer(t *testing.T) {
	e := New(3)
	now := time.Now()
	for i := 0; i < 5; i++ {
		e.Record("span", now, now, nil, "")
	}
	e.mu.Lock()
	count := len(e.spans)
	e.mu.Unlock()
	if count != 3 {
		t.Errorf("expected 3 spans (ring buffer), got %d", count)
	}
}

func TestErrorSpan(t *testing.T) {
	e := New(10)
	now := time.Now()
	e.Record("fail", now, now, nil, "timeout")
	e.mu.Lock()
	s := e.spans[0]
	e.mu.Unlock()
	if s.Status == nil || s.Status.Code != 2 {
		t.Error("expected error status code 2")
	}
}
