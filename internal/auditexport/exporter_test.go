package auditexport

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
)

type memUploader struct {
	mu   sync.Mutex
	data map[string][]byte
}

func (m *memUploader) Upload(key string, data []byte) error {
	m.mu.Lock()
	m.data[key] = data
	m.mu.Unlock()
	return nil
}

func TestFlush(t *testing.T) {
	u := &memUploader{data: make(map[string][]byte)}
	e := New(u, "audit", 0)
	e.Add(json.RawMessage(`{"action":"login"}`))
	e.Add(json.RawMessage(`{"action":"logout"}`))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	if len(u.data) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(u.data))
	}
	for _, v := range u.data {
		lines := strings.Split(strings.TrimSpace(string(v)), "\n")
		if len(lines) != 2 {
			t.Errorf("expected 2 lines, got %d", len(lines))
		}
	}
}

func TestFlush_Empty(t *testing.T) {
	u := &memUploader{data: make(map[string][]byte)}
	e := New(u, "audit", 0)
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	if len(u.data) != 0 {
		t.Error("empty flush should not upload")
	}
}

func TestOnFlushCallback(t *testing.T) {
	u := &memUploader{data: make(map[string][]byte)}
	e := New(u, "audit", 0)
	var flushedCount int
	e.OnFlush = func(count int, key string) { flushedCount = count }
	e.Add(json.RawMessage(`{"x":1}`))
	e.Flush()
	if flushedCount != 1 {
		t.Errorf("OnFlush count = %d, want 1", flushedCount)
	}
}

func TestStop_FinalFlush(t *testing.T) {
	u := &memUploader{data: make(map[string][]byte)}
	e := New(u, "audit", 0)
	e.Add(json.RawMessage(`{"final":true}`))
	e.Stop()
	if len(u.data) != 1 {
		t.Error("Stop should do final flush")
	}
}
