package store

import (
	"testing"
)

func TestEdge_MemorySetGet(t *testing.T) {
	m := NewMemory()
	m.Set("k", []byte(`"v"`))
	v, err := m.Get("k")
	if err != nil || string(v) != `"v"` {
		t.Errorf("memory set/get failed: %v %q", err, v)
	}
}

func TestEdge_MemoryDelete(t *testing.T) {
	m := NewMemory()
	m.Set("k", []byte(`"v"`))
	m.Delete("k")
	_, err := m.Get("k")
	if err == nil {
		t.Error("deleted key should error")
	}
}

func TestEdge_MemoryGetMissing(t *testing.T) {
	m := NewMemory()
	_, err := m.Get("nope")
	if err == nil {
		t.Error("missing key should error")
	}
}
