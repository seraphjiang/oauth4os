// Package accesslog provides JSON-structured HTTP access logging middleware.
// This file adds a rotating file writer.
package accesslog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RotatingWriter writes to a file, rotating when maxBytes is exceeded.
type RotatingWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	maxFiles int
	file     *os.File
	size     int64
}

// NewRotatingWriter creates a writer that rotates at maxBytes, keeping maxFiles backups.
func NewRotatingWriter(path string, maxBytes int64, maxFiles int) (*RotatingWriter, error) {
	w := &RotatingWriter{path: path, maxBytes: maxBytes, maxFiles: maxFiles}
	if err := w.open(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *RotatingWriter) open() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	info, _ := f.Stat()
	w.file = f
	w.size = info.Size()
	return nil
}

func (w *RotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.size+int64(len(p)) > w.maxBytes {
		w.rotate()
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *RotatingWriter) rotate() {
	w.file.Close()
	// Shift existing backups
	for i := w.maxFiles - 1; i > 0; i-- {
		os.Rename(fmt.Sprintf("%s.%d", w.path, i), fmt.Sprintf("%s.%d", w.path, i+1))
	}
	os.Rename(w.path, w.path+".1")
	// Remove oldest if over limit
	os.Remove(fmt.Sprintf("%s.%d", w.path, w.maxFiles+1))
	w.open()
}

// Close closes the underlying file.
func (w *RotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
