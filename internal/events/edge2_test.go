package events

import (
	"sync"
	"testing"
)

func TestEdge_ConcurrentEmit(t *testing.T) {
	n := New([]string{"http://localhost:1/webhook"})
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			n.Emit(Event{Type: TokenIssued, ClientID: "app"})
		}()
	}
	wg.Wait()
	n.Stop()
}

func TestEdge_EmitWithNoURLs(t *testing.T) {
	n := New(nil)
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
	n.Stop()
}

func TestEdge_StopIdempotent(t *testing.T) {
	n := New([]string{"http://localhost:1/webhook"})
	n.Stop()
	n.Stop() // must not panic
}

func TestEdge_EmitAllTypes(t *testing.T) {
	n := New(nil)
	defer n.Stop()
	n.Emit(Event{Type: TokenIssued, ClientID: "app"})
	n.Emit(Event{Type: TokenRevoked, ClientID: "app"})
	n.Emit(Event{Type: TokenRefresh, ClientID: "app"})
}
