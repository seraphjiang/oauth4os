package audit

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// Store persists and queries audit events.
type Store interface {
	Write(entry LogEntry) error
	Query(filter QueryFilter) ([]LogEntry, error)
}

// QueryFilter for searching audit logs.
type QueryFilter struct {
	ClientID string
	Event    string
	Since    time.Time
	Limit    int
}

// MemoryStore keeps audit events in memory with optional file persistence.
type MemoryStore struct {
	entries []LogEntry
	mu      sync.RWMutex
	maxSize int
	file    *os.File
	enc     *json.Encoder
}

// NewMemoryStore creates an in-memory store. If path is non-empty, also appends to file.
func NewMemoryStore(maxSize int, path string) (*MemoryStore, error) {
	s := &MemoryStore{maxSize: maxSize}
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		s.file = f
		s.enc = json.NewEncoder(f)
	}
	return s, nil
}

func (s *MemoryStore) Write(entry LogEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	s.mu.Lock()
	s.entries = append(s.entries, entry)
	// Ring buffer — drop oldest when full
	if s.maxSize > 0 && len(s.entries) > s.maxSize {
		s.entries = s.entries[len(s.entries)-s.maxSize:]
	}
	s.mu.Unlock()

	if s.enc != nil {
		return s.enc.Encode(entry)
	}
	return nil
}

func (s *MemoryStore) Query(f QueryFilter) ([]LogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	}

	var results []LogEntry
	// Iterate in reverse (newest first)
	for i := len(s.entries) - 1; i >= 0 && len(results) < limit; i-- {
		e := s.entries[i]
		if f.ClientID != "" && e.ClientID != f.ClientID {
			continue
		}
		if f.Event != "" && e.Event != f.Event {
			continue
		}
		if !f.Since.IsZero() {
			ts, err := time.Parse(time.RFC3339Nano, e.Timestamp)
			if err != nil || ts.Before(f.Since) {
				continue
			}
		}
		results = append(results, e)
	}
	return results, nil
}

func (s *MemoryStore) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}
