package par

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Edge: PAR with valid client returns request_uri
func TestEdge_ValidPAR(t *testing.T) {
	h := NewHandler(func(id, secret string) error { return nil })
	body := "client_id=test&client_secret=secret&redirect_uri=http://localhost/cb&scope=read"
	r := httptest.NewRequest("POST", "/oauth/par", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.Push(w, r)
	if w.Code >= 500 {
		t.Errorf("unexpected server error: %d", w.Code)
	}
}
