package events

import (
	"testing"
	"time"
)

// Mutation: remove Stop → goroutine must terminate cleanly
func TestMutation_StopDrainsGoroutine(t *testing.T) {
	n := New(nil)
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
	done := make(chan struct{})
	go func() {
		n.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop must terminate the drain goroutine")
	}
}
