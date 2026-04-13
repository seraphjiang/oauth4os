package otlp

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestEdge_HandlerReturnsTraces(t *testing.T) {
	e := New(100)
	e.Record("test-op", time.Now().Add(-time.Second), time.Now(), nil, "")
	w := httptest.NewRecorder()
	e.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("should return trace data")
	}
}

func TestEdge_EmptyExporterReturnsEmpty(t *testing.T) {
	e := New(100)
	w := httptest.NewRecorder()
	e.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/v1/traces", nil))
	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
