package metrics

import (
	"bytes"
	"testing"
)

func TestEdge_NewCounterEmpty(t *testing.T) {
	c := NewCounter()
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "test", "test help")
	// Empty counter should still produce valid output or empty
	_ = buf.String()
}

func TestEdge_IncAndSnapshot(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/health", Status: 200})
	c.Inc(Labels{Method: "GET", Path: "/health", Status: 200})
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "requests", "Total requests")
	if buf.Len() == 0 {
		t.Error("counter with data should produce output")
	}
}

func TestEdge_DifferentLabelsTrackedSeparately(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/a", Status: 200})
	c.Inc(Labels{Method: "POST", Path: "/b", Status: 201})
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "req", "help")
	s := buf.String()
	if !bytes.Contains([]byte(s), []byte("GET")) || !bytes.Contains([]byte(s), []byte("POST")) {
		t.Error("different labels should be tracked separately")
	}
}
