package accesslog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareLogsJSON(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("ok"))
	})
	handler := l.Middleware(inner, func(r *http.Request) string { return "test-client" })
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/oauth/token", nil))

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if entry.Method != "POST" {
		t.Errorf("expected POST, got %s", entry.Method)
	}
	if entry.Path != "/oauth/token" {
		t.Errorf("expected /oauth/token, got %s", entry.Path)
	}
	if entry.Status != 201 {
		t.Errorf("expected 201, got %d", entry.Status)
	}
	if entry.ClientID != "test-client" {
		t.Errorf("expected test-client, got %s", entry.ClientID)
	}
	if entry.Size != 2 {
		t.Errorf("expected size 2, got %d", entry.Size)
	}
}

func TestMiddlewareNilClientID(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	var entry Entry
	json.Unmarshal(buf.Bytes(), &entry)
	if entry.ClientID != "" {
		t.Errorf("expected empty client_id, got %s", entry.ClientID)
	}
}
