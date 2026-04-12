package audit

import (
	"sync"
	"testing"
	"time"
)

// Edge cases and chaos tests for audit store.

func TestQueryFilter_Since(t *testing.T) {
	s, _ := NewMemoryStore(100, "")
	old := LogEntry{Event: "old", ClientID: "c", Timestamp: "2020-01-01T00:00:00Z"}
	recent := LogEntry{Event: "recent", ClientID: "c", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
	s.Write(old)
	s.Write(recent)

	results, _ := s.Query(QueryFilter{Since: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)})
	if len(results) != 1 || results[0].Event != "recent" {
		t.Errorf("expected only recent event, got %d", len(results))
	}
}

func TestQueryFilter_ClientID(t *testing.T) {
	s, _ := NewMemoryStore(100, "")
	s.Write(LogEntry{Event: "a", ClientID: "alice"})
	s.Write(LogEntry{Event: "b", ClientID: "bob"})

	results, _ := s.Query(QueryFilter{ClientID: "alice"})
	if len(results) != 1 || results[0].ClientID != "alice" {
		t.Errorf("expected alice only, got %d", len(results))
	}
}

func TestQueryFilter_Limit(t *testing.T) {
	s, _ := NewMemoryStore(100, "")
	for i := 0; i < 50; i++ {
		s.Write(LogEntry{Event: "e", ClientID: "c"})
	}
	results, _ := s.Query(QueryFilter{Limit: 5})
	if len(results) != 5 {
		t.Errorf("expected 5, got %d", len(results))
	}
}

func TestQueryFilter_DefaultLimit(t *testing.T) {
	s, _ := NewMemoryStore(200, "")
	for i := 0; i < 150; i++ {
		s.Write(LogEntry{Event: "e", ClientID: "c"})
	}
	results, _ := s.Query(QueryFilter{})
	if len(results) != 100 {
		t.Errorf("default limit should be 100, got %d", len(results))
	}
}

func TestConcurrentWriteAndQuery(t *testing.T) {
	s, _ := NewMemoryStore(1000, "")
	var wg sync.WaitGroup

	// 50 concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				s.Write(LogEntry{Event: "write", ClientID: "c"})
			}
		}(i)
	}

	// 20 concurrent readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				s.Query(QueryFilter{Limit: 10})
			}
		}()
	}

	wg.Wait()

	results, _ := s.Query(QueryFilter{Limit: 1000})
	if len(results) != 1000 {
		t.Errorf("expected 1000 entries, got %d", len(results))
	}
}

func TestRingBuffer_Eviction(t *testing.T) {
	s, _ := NewMemoryStore(5, "")
	for i := 0; i < 10; i++ {
		s.Write(LogEntry{Event: "e", ClientID: "c"})
	}
	results, _ := s.Query(QueryFilter{Limit: 100})
	if len(results) != 5 {
		t.Errorf("ring buffer should cap at 5, got %d", len(results))
	}
}

func TestMalformedTimestamp_SkippedBySince(t *testing.T) {
	s, _ := NewMemoryStore(100, "")
	s.Write(LogEntry{Event: "bad", ClientID: "c", Timestamp: "not-a-timestamp"})
	s.Write(LogEntry{Event: "good", ClientID: "c", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})

	results, _ := s.Query(QueryFilter{Since: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)})
	if len(results) != 1 || results[0].Event != "good" {
		t.Errorf("malformed timestamp should be skipped by Since filter, got %d results", len(results))
	}
}
