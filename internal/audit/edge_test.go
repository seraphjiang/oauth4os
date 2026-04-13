package audit

import (
	"bytes"
	"testing"
)

// Edge: Log records action
func TestEdge_LogRecords(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.Log("admin", []string{"all"}, "POST", "/admin/clients")
	if buf.Len() == 0 {
		t.Error("Log should write audit entry")
	}
}

// Edge: LogAuth records auth event
func TestEdge_LogAuth(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.LogAuth("client-1", "login", nil)
	if buf.Len() == 0 {
		t.Error("LogAuth should write entry")
	}
}

// Edge: LogAuth with error records failure
func TestEdge_LogAuthError(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.LogAuth("client-1", "login", fmt.Errorf("bad credentials"))
	if !bytes.Contains(buf.Bytes(), []byte("bad credentials")) {
		t.Error("LogAuth error should include error message")
	}
}
