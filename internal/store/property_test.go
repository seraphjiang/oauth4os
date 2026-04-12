package store

import (
	"path/filepath"
	"testing"
)

// Property: File store Set→Get round-trip preserves data across reopen
func TestProperty_FilePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "store.json")

	f, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Set("key1", []byte("value1"))
	f.Set("key2", []byte("value2"))
	f.Close()

	// Reopen and verify
	f2, err := NewFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	got, err := f2.Get("key1")
	if err != nil || string(got) != "value1" {
		t.Errorf("expected 'value1', got %q err %v", got, err)
	}
	got2, err := f2.Get("key2")
	if err != nil || string(got2) != "value2" {
		t.Errorf("expected 'value2', got %q err %v", got2, err)
	}
	keys, _ := f2.List()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys after reopen, got %d", len(keys))
	}
}
