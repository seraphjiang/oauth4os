package histogram

import (
	"bytes"
	"testing"
	"time"
)

func TestEdge_ObserveAndWrite(t *testing.T) {
	h := New()
	h.Observe(50*time.Millisecond, "/api/data")
	h.Observe(150*time.Millisecond, "/api/data")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "request_duration")
	if buf.Len() == 0 {
		t.Error("WritePrometheus should produce output")
	}
}

func TestEdge_ObserveZero(t *testing.T) {
	h := New()
	h.Observe(0, "/")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "latency")
	if buf.Len() == 0 {
		t.Error("zero observation should still produce output")
	}
}

func TestEdge_MultiplePaths(t *testing.T) {
	h := New()
	h.Observe(100*time.Millisecond, "/a")
	h.Observe(200*time.Millisecond, "/b")
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "dur")
	s := buf.String()
	if !bytes.Contains([]byte(s), []byte("/a")) || !bytes.Contains([]byte(s), []byte("/b")) {
		t.Error("output should contain both paths")
	}
}
