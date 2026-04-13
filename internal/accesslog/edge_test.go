package accesslog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_LogsRequestMethod(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) string { return "test-client" })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/data", nil))
	if !bytes.Contains(buf.Bytes(), []byte("GET")) {
		t.Error("log should contain request method")
	}
}

func TestEdge_LogsStatusCode(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}), func(r *http.Request) string { return "" })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/missing", nil))
	if !bytes.Contains(buf.Bytes(), []byte("404")) {
		t.Error("log should contain status code")
	}
}

func TestEdge_LogsPath(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf)
	h := l.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}), func(r *http.Request) string { return "" })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/specific/path", nil))
	if !bytes.Contains(buf.Bytes(), []byte("/specific/path")) {
		t.Error("log should contain request path")
	}
}
