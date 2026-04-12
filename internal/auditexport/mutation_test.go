package auditexport

import (
	"encoding/json"
	"strings"
	"testing"
)

type mockUploader struct {
	keys []string
	data [][]byte
	err  error
}

func (m *mockUploader) Upload(key string, data []byte) error {
	m.keys = append(m.keys, key)
	m.data = append(m.data, data)
	return m.err
}

// M1: Flush with no entries is a no-op.
func TestMutation_FlushEmpty(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "audit", 0)
	if err := e.Flush(); err != nil {
		t.Fatalf("empty flush should succeed, got %v", err)
	}
	if len(u.keys) != 0 {
		t.Fatal("no upload should happen on empty flush")
	}
}

// M2: Add + Flush uploads NDJSON.
func TestMutation_AddAndFlush(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "audit", 0)
	e.Add(json.RawMessage(`{"action":"login"}`))
	e.Add(json.RawMessage(`{"action":"logout"}`))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	if len(u.data) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(u.data))
	}
	lines := strings.Split(strings.TrimSpace(string(u.data[0])), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}
}

// M3: Flush clears buffer — second flush is no-op.
func TestMutation_FlushClearsBuffer(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "audit", 0)
	e.Add(json.RawMessage(`{"x":1}`))
	e.Flush()
	e.Flush()
	if len(u.data) != 1 {
		t.Fatalf("second flush should be no-op, got %d uploads", len(u.data))
	}
}

// M4: Key contains prefix.
func TestMutation_KeyContainsPrefix(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "my-prefix", 0)
	e.Add(json.RawMessage(`{}`))
	e.Flush()
	if !strings.HasPrefix(u.keys[0], "my-prefix/") {
		t.Fatalf("key should start with prefix, got %s", u.keys[0])
	}
}

// M5: Key ends with .ndjson.
func TestMutation_KeyFormat(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "p", 0)
	e.Add(json.RawMessage(`{}`))
	e.Flush()
	if !strings.HasSuffix(u.keys[0], ".ndjson") {
		t.Fatalf("key should end with .ndjson, got %s", u.keys[0])
	}
}

// M6: OnFlush callback fires with correct count.
func TestMutation_OnFlushCallback(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "p", 0)
	var gotCount int
	e.OnFlush = func(count int, key string) { gotCount = count }
	e.Add(json.RawMessage(`{}`))
	e.Add(json.RawMessage(`{}`))
	e.Add(json.RawMessage(`{}`))
	e.Flush()
	if gotCount != 3 {
		t.Fatalf("OnFlush count: expected 3, got %d", gotCount)
	}
}

// M7: Stop does final flush.
func TestMutation_StopFlushes(t *testing.T) {
	u := &mockUploader{}
	e := New(u, "p", 0)
	e.Add(json.RawMessage(`{}`))
	e.Stop()
	if len(u.data) != 1 {
		t.Fatal("Stop should trigger final flush")
	}
}
