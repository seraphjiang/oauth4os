package metrics

import (
	"bytes"
	"sync"
	"testing"
)

func TestEdge_ConcurrentInc(t *testing.T) {
	c := NewCounter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Inc(Labels{Method: "GET", Path: "/", Status: 200 + n%5})
		}(i)
	}
	wg.Wait()
}

func TestEdge_HighCardinalityLabels(t *testing.T) {
	c := NewCounter()
	for i := 0; i < MaxCardinality+10; i++ {
		c.Inc(Labels{Method: "GET", Path: "/" + string(rune('a'+i%26)), Status: 200})
	}
	// Should not panic — cardinality guard caps it
}

func TestEdge_ConcurrentWritePrometheus(t *testing.T) {
	c := NewCounter()
	c.Inc(Labels{Method: "GET", Path: "/", Status: 200})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			c.WritePrometheus(&buf, "req", "help")
		}()
	}
	wg.Wait()
}
