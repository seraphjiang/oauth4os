package tlsreload

import "testing"

func TestEdge_NewWithBadPathFails(t *testing.T) {
	_, err := New("/nonexistent/cert.pem", "/nonexistent/key.pem", 0)
	if err == nil {
		t.Error("bad cert path should fail")
	}
}
