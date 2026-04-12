package metrics

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// M1: Counter Inc increments by 1.
func TestMutation_CounterInc(t *testing.T) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/", Status: 200}
	c.Inc(l)
	c.Inc(l)
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "test", "help")
	if !strings.Contains(buf.String(), "2") {
		t.Fatalf("expected count 2, got %s", buf.String())
	}
}

// M2: Counter cardinality guard at 1000.
func TestMutation_CounterCardinalityGuard(t *testing.T) {
	c := NewCounter()
	for i := 0; i < MaxCardinality+100; i++ {
		c.Inc(Labels{Method: "GET", Path: "/", Status: i})
	}
	if c.Cardinality() > MaxCardinality {
		t.Fatalf("cardinality %d exceeds max %d", c.Cardinality(), MaxCardinality)
	}
}

// M3: Summary Observe tracks count.
func TestMutation_SummaryCount(t *testing.T) {
	s := NewSummary()
	l := Labels{Method: "POST", Path: "/api", Status: 201}
	s.Observe(l, 10*time.Millisecond)
	s.Observe(l, 20*time.Millisecond)
	if s.Cardinality() != 1 {
		t.Fatalf("expected 1 series, got %d", s.Cardinality())
	}
	var buf bytes.Buffer
	s.WritePrometheus(&buf, "latency", "help")
	if !strings.Contains(buf.String(), "latency_count") {
		t.Fatalf("missing count line: %s", buf.String())
	}
}

// M4: Summary cardinality guard.
func TestMutation_SummaryCardinalityGuard(t *testing.T) {
	s := NewSummary()
	for i := 0; i < MaxCardinality+100; i++ {
		s.Observe(Labels{Method: "GET", Path: "/", Status: i}, time.Millisecond)
	}
	if s.Cardinality() > MaxCardinality {
		t.Fatalf("cardinality %d exceeds max %d", s.Cardinality(), MaxCardinality)
	}
}

// M5: Labels.String() format.
func TestMutation_LabelsString(t *testing.T) {
	l := Labels{Method: "GET", Path: "/health", Status: 200}
	s := l.String()
	if !strings.Contains(s, "GET") || !strings.Contains(s, "/health") || !strings.Contains(s, "200") {
		t.Fatalf("unexpected label string: %s", s)
	}
}

// M6: Counter Add with custom value.
func TestMutation_CounterAdd(t *testing.T) {
	c := NewCounter()
	l := Labels{Method: "GET", Path: "/", Status: 200}
	c.Add(l, 5)
	c.Add(l, 3)
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "test", "help")
	if !strings.Contains(buf.String(), "8") {
		t.Fatalf("expected 8, got %s", buf.String())
	}
}

// M7: Empty counter writes header only.
func TestMutation_EmptyCounterOutput(t *testing.T) {
	c := NewCounter()
	var buf bytes.Buffer
	c.WritePrometheus(&buf, "test", "help")
	if !strings.Contains(buf.String(), "# TYPE test counter") {
		t.Fatalf("missing type header: %s", buf.String())
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 { // HELP + TYPE
		t.Fatalf("expected 2 header lines, got %d", len(lines))
	}
}
