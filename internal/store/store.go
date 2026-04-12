// Package store provides a pluggable key-value store for token persistence.
// Backends: memory (default), file (JSON + fsync). DDB/Redis planned for v1.2.0.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Store is the interface for token persistence backends.
type Store interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
	Delete(key string) error
	List() ([]string, error)
	Close() error
}

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = fmt.Errorf("key not found")

// --- Memory backend ---

// Memory is an in-memory store (default, non-persistent).
type Memory struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{data: make(map[string][]byte)}
}

func (m *Memory) Get(key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	cp := make([]byte, len(v))
	copy(cp, v)
	return cp, nil
}

func (m *Memory) Set(key string, value []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]byte, len(value))
	copy(cp, value)
	m.data[key] = cp
	return nil
}

func (m *Memory) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *Memory) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *Memory) Close() error { return nil }

// --- File backend ---

// File is a JSON file-backed store with fsync durability.
type File struct {
	mu   sync.RWMutex
	path string
	data map[string]json.RawMessage
}

func NewFile(path string) (*File, error) {
	f := &File{path: path, data: make(map[string]json.RawMessage)}
	raw, err := os.ReadFile(path)
	if err == nil {
		json.Unmarshal(raw, &f.data)
	}
	// Missing file is fine — start empty
	return f, nil
}

func (f *File) Get(key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	v, ok := f.data[key]
	if !ok {
		return nil, ErrNotFound
	}
	return []byte(v), nil
}

func (f *File) Set(key string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = json.RawMessage(value)
	return f.flush()
}

func (f *File) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.data, key)
	return f.flush()
}

func (f *File) List() ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	keys := make([]string, 0, len(f.data))
	for k := range f.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.flush()
}

// flush writes data to disk with atomic rename + fsync.
func (f *File) flush() error {
	tmp := f.path + ".tmp"
	raw, err := json.MarshalIndent(f.data, "", "  ")
	if err != nil {
		return err
	}
	fp, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := fp.Write(raw); err != nil {
		fp.Close()
		return err
	}
	if err := fp.Sync(); err != nil {
		fp.Close()
		return err
	}
	fp.Close()
	return os.Rename(tmp, f.path)
}
