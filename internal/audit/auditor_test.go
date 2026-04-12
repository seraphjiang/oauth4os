package audit

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAuditorLogJSON(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.Log("my-agent", []string{"read:logs-*"}, "GET", "/logs/_search")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("not valid JSON: %v\nraw: %s", err, buf.String())
	}
	if entry.Event != "proxy_request" {
		t.Errorf("event = %s", entry.Event)
	}
	if entry.ClientID != "my-agent" {
		t.Errorf("client_id = %s", entry.ClientID)
	}
	if entry.Method != "GET" {
		t.Errorf("method = %s", entry.Method)
	}
	if entry.Timestamp == "" {
		t.Error("missing timestamp")
	}
}

func TestAuditorLogAuth(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.LogAuth("bad-client", "auth_failed", errors.New("token expired"))

	var entry LogEntry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.Level != "warn" {
		t.Errorf("level = %s, want warn", entry.Level)
	}
	if entry.Error != "token expired" {
		t.Errorf("error = %s", entry.Error)
	}
}

func TestAuditorLogAuthSuccess(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.LogAuth("good-client", "auth_success", nil)

	var entry LogEntry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.Level != "info" {
		t.Errorf("level = %s, want info", entry.Level)
	}
	if entry.Error != "" {
		t.Errorf("error should be empty, got %s", entry.Error)
	}
}

func TestAuditorEmptyScopes(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.Log("cli", nil, "POST", "/oauth/token")

	var entry LogEntry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.Scopes != nil {
		t.Errorf("scopes should be nil, got %v", entry.Scopes)
	}
}

func TestWithStore(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	s, _ := NewMemoryStore(100, "")
	a2 := a.WithStore(s)
	if a2 == nil {
		t.Fatal("WithStore returned nil")
	}
	a2.Log("client1", []string{"read:*"}, "GET", "/test")
	if s.Len() != 1 {
		t.Errorf("store should have 1 entry, got %d", s.Len())
	}
}

func TestLogCedar(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	s, _ := NewMemoryStore(100, "")
	a.WithStore(s)
	a.LogCedar("client1", "GET", "logs-demo", "policy-1", "permitted", true)
	if buf.Len() == 0 {
		t.Error("expected JSON output")
	}
	if !strings.Contains(buf.String(), "cedar") {
		t.Error("expected cedar in output")
	}
}

func TestAuditorQuery(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	s, _ := NewMemoryStore(100, "")
	a = a.WithStore(s)
	a.Log("c1", nil, "GET", "/a")
	a.Log("c2", nil, "POST", "/b")
	entries, err := a.Query(QueryFilter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestStoreCloseAndLen(t *testing.T) {
	s, _ := NewMemoryStore(100, "")
	s.Write(LogEntry{ClientID: "c1"})
	s.Write(LogEntry{ClientID: "c2"})
	if s.Len() != 2 {
		t.Errorf("Len = %d, want 2", s.Len())
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close error: %v", err)
	}
}
