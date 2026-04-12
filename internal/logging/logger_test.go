package logging

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestLoggerJSON(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")

	l.Info("test message", "key", "value", "num", 42)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry["level"] != "INFO" {
		t.Errorf("expected INFO, got %v", entry["level"])
	}
	if entry["msg"] != "test message" {
		t.Errorf("expected 'test message', got %v", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Errorf("expected 'value', got %v", entry["key"])
	}
}

func TestLoggerLevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "warn")

	l.Debug("skip")
	l.Info("skip")
	if buf.Len() != 0 {
		t.Error("expected debug/info to be filtered")
	}

	l.Warn("visible")
	if buf.Len() == 0 {
		t.Error("expected warn to be logged")
	}
}
