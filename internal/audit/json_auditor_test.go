package audit

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestJSONAuditor_Log(t *testing.T) {
	var buf bytes.Buffer
	a := NewJSONAuditor(&buf)
	a.Log("client1", []string{"read:logs-*"}, "GET", "/logs-*/_search")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry.ClientID != "client1" {
		t.Fatalf("client_id = %s", entry.ClientID)
	}
	if entry.Method != "GET" {
		t.Fatalf("method = %s", entry.Method)
	}
	if entry.Level != "info" {
		t.Fatalf("level = %s", entry.Level)
	}
	if entry.Timestamp == "" {
		t.Fatal("missing timestamp")
	}
}

func TestJSONAuditor_LogWithDetails(t *testing.T) {
	var buf bytes.Buffer
	a := NewJSONAuditor(&buf)
	a.LogWithDetails("client1", []string{"admin"}, "POST", "/_bulk", "req-123", 200, 42)

	var entry Entry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.RequestID != "req-123" {
		t.Fatalf("request_id = %s", entry.RequestID)
	}
	if entry.Status != 200 {
		t.Fatalf("status = %d", entry.Status)
	}
	if entry.DurationMs != 42 {
		t.Fatalf("duration_ms = %d", entry.DurationMs)
	}
}

func TestJSONAuditor_LogError(t *testing.T) {
	var buf bytes.Buffer
	a := NewJSONAuditor(&buf)
	a.LogError("client1", "GET", "/bad", "req-456", "invalid token")

	var entry Entry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.Level != "error" {
		t.Fatalf("level = %s", entry.Level)
	}
	if entry.Error != "invalid token" {
		t.Fatalf("error = %s", entry.Error)
	}
}
