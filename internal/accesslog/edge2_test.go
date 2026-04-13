package accesslog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_LogsClientID(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) string { return "my-client-id" })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if !bytes.Contains(buf.Bytes(), []byte("my-client-id")) {
		t.Error("log should contain client ID")
	}
}

func TestEdge_EmptyClientID(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) string { return "" })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	// Should still log — just without client ID
	if buf.Len() == 0 {
		t.Error("should still produce log entry")
	}
}
