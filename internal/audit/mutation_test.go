package audit

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Mutation: remove JSON encoding → Log must produce valid JSON
func TestMutation_JSONOutput(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.Log("app", []string{"read"}, "GET", "/logs/_search")
	var e LogEntry
	if err := json.Unmarshal(buf.Bytes(), &e); err != nil {
		t.Fatalf("audit log must be valid JSON: %v", err)
	}
	if e.ClientID != "app" {
		t.Errorf("expected client_id=app, got %s", e.ClientID)
	}
}

// Mutation: remove method/path → must log request details
func TestMutation_RequestDetails(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.Log("app", nil, "POST", "/oauth/token")
	var e LogEntry
	json.Unmarshal(buf.Bytes(), &e)
	if e.Method != "POST" || e.Path != "/oauth/token" {
		t.Errorf("expected POST /oauth/token, got %s %s", e.Method, e.Path)
	}
}

// Mutation: remove auth event logging → LogAuth must record auth events
func TestMutation_AuthEvent(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.LogAuth("app", "login", nil)
	if buf.Len() == 0 {
		t.Error("LogAuth must produce output")
	}
}

// Mutation: remove cedar logging → LogCedar must record policy decisions
func TestMutation_CedarEvent(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.LogCedar("app", "search", "logs-*", "policy1", "matched", true)
	if buf.Len() == 0 {
		t.Error("LogCedar must produce output")
	}
}

// Mutation: remove Write → store must persist entries
func TestMutation_StoreWriteQuery(t *testing.T) {
	s, err := NewMemoryStore(100, "")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.Write(LogEntry{ClientID: "app", Method: "GET", Path: "/test"})
	entries, err := s.Query(QueryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("store must return written entries")
	}
}
