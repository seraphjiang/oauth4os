package audit

import (
	"testing"
	"time"
)

func TestMemoryStoreWriteAndQuery(t *testing.T) {
	s, _ := NewMemoryStore(100, "")

	s.Write(LogEntry{Event: "auth_success", ClientID: "agent-1", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	s.Write(LogEntry{Event: "proxy_request", ClientID: "agent-1", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	s.Write(LogEntry{Event: "auth_failed", ClientID: "agent-2", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})

	// Query all
	results, _ := s.Query(QueryFilter{})
	if len(results) != 3 {
		t.Errorf("expected 3, got %d", len(results))
	}

	// Filter by client
	results, _ = s.Query(QueryFilter{ClientID: "agent-1"})
	if len(results) != 2 {
		t.Errorf("expected 2 for agent-1, got %d", len(results))
	}

	// Filter by event
	results, _ = s.Query(QueryFilter{Event: "auth_failed"})
	if len(results) != 1 {
		t.Errorf("expected 1 auth_failed, got %d", len(results))
	}
}

func TestMemoryStoreRingBuffer(t *testing.T) {
	s, _ := NewMemoryStore(3, "")

	for i := 0; i < 5; i++ {
		s.Write(LogEntry{Event: "test", ClientID: "x", Timestamp: time.Now().UTC().Format(time.RFC3339Nano)})
	}

	results, _ := s.Query(QueryFilter{})
	if len(results) != 3 {
		t.Errorf("ring buffer should cap at 3, got %d", len(results))
	}
}

func TestMemoryStoreNewestFirst(t *testing.T) {
	s, _ := NewMemoryStore(100, "")

	s.Write(LogEntry{Event: "first", Timestamp: "2026-01-01T00:00:00Z"})
	s.Write(LogEntry{Event: "second", Timestamp: "2026-01-02T00:00:00Z"})

	results, _ := s.Query(QueryFilter{})
	if results[0].Event != "second" {
		t.Error("expected newest first")
	}
}
