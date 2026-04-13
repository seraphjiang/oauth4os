package remotewrite

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEdge_HandlerAcceptsPOST(t *testing.T) {
	r := New()
	h := r.Handler()
	body := `name,value
http_requests_total,42`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/write", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	h.ServeHTTP(w, req)
	if w.Code == 404 {
		t.Error("POST should be accepted")
	}
}

func TestEdge_EmptyBodyHandled(t *testing.T) {
	r := New()
	h := r.Handler()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/write", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")
	h.ServeHTTP(w, req)
	// Should not panic
	if w.Code == 0 {
		t.Error("should return valid status")
	}
}
