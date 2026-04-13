package histogram

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestEdge_ConcurrentWritePrometheus(t *testing.T) {
	h := New()
	h.Observe(50*time.Millisecond, "/api")
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			h.Observe(time.Duration(i)*time.Millisecond, "/api")
		}()
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			h.WritePrometheus(&buf, "dur")
		}()
	}
	wg.Wait()
}

func TestEdge_ManyPaths(t *testing.T) {
	h := New()
	for i := 0; i < 100; i++ {
		h.Observe(time.Millisecond, "/path/"+string(rune('a'+i%26)))
	}
	var buf bytes.Buffer
	h.WritePrometheus(&buf, "req")
	if buf.Len() == 0 {
		t.Error("should produce output for many paths")
	}
}
