package store

import (
	"path/filepath"
	"testing"
)

// Edge: NewFile creates file
func TestEdge_NewFileCreates(t *testing.T) {
	dir := t.TempDir()
	f, err := NewFile(filepath.Join(dir, "test.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
}

// Edge: Set+Get round-trip
func TestEdge_SetGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	f, _ := NewFile(filepath.Join(dir, "test.json"))
	defer f.Close()
	f.Set("key1", []byte(`"value1"`))
	v, ok := f.Get("key1")
	if !ok {
		t.Fatal("Get should find key")
	}
	if string(v) != `"value1"` {
		t.Errorf("expected '\"value1\"', got %q", string(v))
	}
}

// Edge: Get missing key returns false
func TestEdge_GetMissing(t *testing.T) {
	dir := t.TempDir()
	f, _ := NewFile(filepath.Join(dir, "test.json"))
	defer f.Close()
	_, ok := f.Get("nonexistent")
	if ok {
		t.Error("missing key should return false")
	}
}

// Edge: Delete removes key
func TestEdge_DeleteRemoves(t *testing.T) {
	dir := t.TempDir()
	f, _ := NewFile(filepath.Join(dir, "test.json"))
	defer f.Close()
	f.Set("key1", []byte(`"value1"`))
	f.Delete("key1")
	_, ok := f.Get("key1")
	if ok {
		t.Error("deleted key should not be found")
	}
}

// Edge: Keys returns all keys
func TestEdge_KeysReturnsAll(t *testing.T) {
	dir := t.TempDir()
	f, _ := NewFile(filepath.Join(dir, "test.json"))
	defer f.Close()
	f.Set("a", []byte(`1`))
	f.Set("b", []byte(`2`))
	keys := f.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}
