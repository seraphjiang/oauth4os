package audit

import (
	"bytes"
	"strings"
	"testing"
)

func TestAuditorLog(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.Log("my-agent", []string{"read:logs-*", "write:logs-*"}, "GET", "/logs-2026/_search")

	out := buf.String()
	if !strings.Contains(out, "client=my-agent") {
		t.Errorf("expected client=my-agent, got: %s", out)
	}
	if !strings.Contains(out, "scopes=[read:logs-*,write:logs-*]") {
		t.Errorf("expected scopes, got: %s", out)
	}
	if !strings.Contains(out, "GET /logs-2026/_search") {
		t.Errorf("expected method+path, got: %s", out)
	}
}

func TestAuditorEmptyScopes(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)

	a.Log("cli", nil, "POST", "/oauth/token")

	out := buf.String()
	if !strings.Contains(out, "scopes=[]") {
		t.Errorf("expected empty scopes, got: %s", out)
	}
}
