package audit

import (
	"bytes"
	"encoding/json"
	"errors"
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
