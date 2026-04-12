package accesslog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingWriter_Rotates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	w, err := NewRotatingWriter(path, 50, 3)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// Write enough to trigger rotation
	data := []byte("0123456789012345678901234567890123456789012345678\n") // 50 bytes
	w.Write(data)
	w.Write(data) // triggers rotation

	// Original file should exist (new after rotation)
	if _, err := os.Stat(path); err != nil {
		t.Error("rotated file should exist")
	}
	// Backup should exist
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Error("backup .1 should exist")
	}
}

func TestRotatingWriter_MaxFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	w, err := NewRotatingWriter(path, 20, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	data := []byte("12345678901234567890\n") // 21 bytes
	for i := 0; i < 5; i++ {
		w.Write(data)
	}

	// Should have at most 2 backups + current
	if _, err := os.Stat(path + ".3"); err == nil {
		t.Error("backup .3 should not exist (maxFiles=2)")
	}
}

func TestRotatingWriter_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	w, err := NewRotatingWriter(path, 1000, 3)
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("test\n"))
	if err := w.Close(); err != nil {
		t.Errorf("close failed: %v", err)
	}
}
