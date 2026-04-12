package auditexport

import (
	"encoding/json"
	"testing"
	"time"
)

// Mutation: remove Add → Flush must export added entries
func TestMutation_AddAndFlush(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "audit/", 0)
	defer e.Stop()
	e.Add(json.RawMessage(`{"action":"login"}`))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.data) == 0 {
		t.Error("Flush must upload added entries")
	}
}

// Mutation: remove Stop → loop goroutine must terminate
func TestMutation_StopTerminates(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "audit/", 50*time.Millisecond)
	done := make(chan struct{})
	go func() {
		e.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must terminate")
	}
}
