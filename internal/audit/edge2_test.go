package audit

import (
	"bytes"
	"sync"
	"testing"
)

func TestEdge_ConcurrentLog(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.Log("admin", []string{"all"}, "POST", "/admin/clients")
		}()
	}
	wg.Wait()
}

func TestEdge_LogCedar(t *testing.T) {
	var buf bytes.Buffer
	a := NewAuditor(&buf)
	a.LogCedar("client-1", "read", "logs-*", "policy-1", "matched", true)
	if buf.Len() == 0 {
		t.Error("LogCedar should write entry")
	}
}
