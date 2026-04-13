package logging

import (
	"bytes"
	"sync"
	"testing"
)

func TestEdge_ConcurrentLog(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "debug")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Info("concurrent message")
		}()
	}
	wg.Wait()
}

func TestEdge_AllLevels(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "debug")
	l.Debug("d")
	l.Info("i")
	l.Warn("w")
	l.Error("e")
	if buf.Len() == 0 {
		t.Error("all levels should produce output at debug level")
	}
}

func TestEdge_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "warn")
	l.Debug("d")
	l.Info("i")
	if buf.Len() != 0 {
		t.Error("debug and info should be suppressed at warn level")
	}
	l.Warn("w")
	if buf.Len() == 0 {
		t.Error("warn should be logged at warn level")
	}
}
