package otlp

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mutation: remove Record → Handler must return recorded spans
func TestMutation_RecordAndExport(t *testing.T) {
	e := New(100)
	e.Record("test-op", time.Now().Add(-time.Second), time.Now(), map[string]string{"k": "v"}, "")
	w := httptest.NewRecorder()
	e.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "test-op") {
		t.Error("handler must return recorded span with operation name")
	}
}

// Mutation: remove ring buffer cap → must not exceed maxSpans
func TestMutation_RingBuffer(t *testing.T) {
	e := New(2)
	for i := 0; i < 5; i++ {
		e.Record("op", time.Now(), time.Now(), nil, "")
	}
	w := httptest.NewRecorder()
	e.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
	body := w.Body.String()
	if strings.Count(body, "traceId") > 2 {
		t.Error("ring buffer should cap at 2 spans")
	}
}

// Mutation: remove error status → Record with error must include error message
func TestMutation_RecordError(t *testing.T) {
	e := New(100)
	e.Record("fail-op", time.Now().Add(-time.Second), time.Now(), nil, "something went wrong")
	w := httptest.NewRecorder()
	e.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
	if !strings.Contains(w.Body.String(), "something went wrong") {
		t.Error("recorded error must appear in trace output")
	}
}
