package audit

import (
	"io"
	"testing"
)

// FuzzLog ensures audit logging never panics on arbitrary input.
func FuzzLog(f *testing.F) {
	f.Add("client", "read write", "GET", "/logs/_search")
	f.Add("", "", "", "")
	f.Add("x", "a b c d e", "POST", "/"+string(make([]byte, 1000)))
	f.Fuzz(func(t *testing.T, clientID, scopeStr, method, path string) {
		a := NewAuditor(io.Discard)
		a.Log(clientID, []string{scopeStr}, method, path) // must not panic
	})
}
