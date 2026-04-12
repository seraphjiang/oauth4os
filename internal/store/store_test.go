package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T, s Store) {
	t.Helper()

	// Set + Get
	if err := s.Set("k1", []byte(`"val1"`)); err != nil {
		t.Fatal(err)
	}
	v, err := s.Get("k1")
	if err != nil || string(v) != `"val1"` {
		t.Errorf("Get k1 = %q, %v", v, err)
	}

	// Get missing
	_, err = s.Get("missing")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}

	// List
	s.Set("k2", []byte(`"val2"`))
	keys, _ := s.List()
	if len(keys) != 2 {
		t.Errorf("List = %d, want 2", len(keys))
	}

	// Delete
	s.Delete("k1")
	_, err = s.Get("k1")
	if !errors.Is(err, ErrNotFound) {
		t.Error("k1 should be deleted")
	}

	// Close
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMemory(t *testing.T) {
	testStore(t, NewMemory())
}

func TestFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	s, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	testStore(t, s)
}

func TestFile_Persistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	s, _ := NewFile(path)
	s.Set("persist", []byte(`"hello"`))
	s.Close()

	// Reopen — data should survive
	s2, _ := NewFile(path)
	v, err := s2.Get("persist")
	if err != nil || string(v) != `"hello"` {
		t.Errorf("after reopen: %q, %v", v, err)
	}
}

func TestFile_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	s, err := NewFile(path)
	if err != nil {
		t.Fatal("should not error on missing file")
	}
	keys, _ := s.List()
	if len(keys) != 0 {
		t.Error("expected empty store")
	}
	_ = s
}

func TestMemory_CopyOnReadWrite(t *testing.T) {
	s := NewMemory()
	orig := []byte("original")
	s.Set("k", orig)
	orig[0] = 'X' // mutate original
	v, _ := s.Get("k")
	if string(v) != "original" {
		t.Error("Set should copy, not reference")
	}
	v[0] = 'Y' // mutate returned value
	v2, _ := s.Get("k")
	if string(v2) != "original" {
		t.Error("Get should copy, not reference")
	}
}
