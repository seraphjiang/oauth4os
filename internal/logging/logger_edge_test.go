package logging

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "debug")

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			l.Info("request", "id", n)
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Fatalf("expected 100 log lines, got %d", len(lines))
	}
}

func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "warn")

	l.Debug("should not appear")
	l.Info("should not appear")
	l.Warn("should appear")
	l.Error("should appear")

	output := buf.String()
	if strings.Contains(output, "DEBUG") || strings.Contains(output, "INFO") {
		t.Fatal("debug/info should be filtered at warn level")
	}
	if !strings.Contains(output, "WARN") || !strings.Contains(output, "ERROR") {
		t.Fatal("warn/error should appear")
	}
}
