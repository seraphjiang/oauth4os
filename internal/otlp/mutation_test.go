package otlp

import (
	"encoding/json"
	"net/http/httptest"
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
	var data map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &data)
	spans, _ := data["spans"].([]interface{})
	if len(spans) == 0 {
		t.Error("handler must return recorded spans")
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
	var data map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &data)
	spans, _ := data["spans"].([]interface{})
	if len(spans) > 2 {
		t.Errorf("ring buffer should cap at 2, got %d", len(spans))
	}
}
