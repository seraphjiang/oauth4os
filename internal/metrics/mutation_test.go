package metrics

import (
	"bytes"
	"testing"
)

// Mutation: remove Inc → WritePrometheus must show incremented counter
func TestMutation_CounterInc(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/health", Status: 200})
	c.Inc(Labels{Method: "GET", Path: "/health", Status: 200})
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "http_requests", "Total requests")
	if !bytes.Contains(buf.Bytes(), []byte("2")) {
		t.Error("counter must reflect 2 increments")
	}
}

// Mutation: remove label isolation → different labels must appear separately
func TestMutation_LabelIsolation(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/a", Status: 200})
	c.Inc(Labels{Method: "POST", Path: "/b", Status: 201})
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "http_requests", "Total requests")
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("GET")) || !bytes.Contains([]byte(out), []byte("POST")) {
		t.Error("different labels must appear separately in output")
	}
}
