package accesslog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Mutation: remove status capture → logged status must match actual
func TestMutation_StatusCapture(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	handler := l.Middleware(inner, nil)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/missing", nil))
	var e Entry
	json.Unmarshal(buf.Bytes(), &e)
	if e.Status != 404 {
		t.Errorf("logged status should be 404, got %d", e.Status)
	}
}

// Mutation: remove size tracking → logged size must match bytes written
func TestMutation_SizeTracking(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("hello")) })
	handler := l.Middleware(inner, nil)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	var e Entry
	json.Unmarshal(buf.Bytes(), &e)
	if e.Size != 5 {
		t.Errorf("logged size should be 5, got %d", e.Size)
	}
}

// Mutation: remove method/path → must log request method and path
func TestMutation_MethodPath(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, nil)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/search", nil))
	var e Entry
	json.Unmarshal(buf.Bytes(), &e)
	if e.Method != "POST" {
		t.Errorf("method should be POST, got %s", e.Method)
	}
	if e.Path != "/api/search" {
		t.Errorf("path should be /api/search, got %s", e.Path)
	}
}

// Mutation: remove client ID extraction → must log client ID when provided
func TestMutation_ClientID(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := l.Middleware(inner, func(r *http.Request) string { return "my-app" })
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	var e Entry
	json.Unmarshal(buf.Bytes(), &e)
	if e.ClientID != "my-app" {
		t.Errorf("client_id should be my-app, got %s", e.ClientID)
	}
}

// Mutation: remove duration tracking → log must include duration_ms
func TestMutation_DurationTracked(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond)
		w.WriteHeader(200)
	})
	handler := l.Middleware(inner, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/test", nil))
	if !strings.Contains(buf.String(), "duration") {
		t.Error("access log must include duration")
	}
}
