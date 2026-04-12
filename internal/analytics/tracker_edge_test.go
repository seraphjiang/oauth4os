package analytics

import (
	"sync"
	"testing"
)

func TestConcurrentRecord(t *testing.T) {
	tr := New()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr.Record("svc-1", []string{"read:logs-*"}, "logs-app")
		}()
	}
	wg.Wait()

	snap := tr.Snapshot()
	total := int64(0)
	for _, c := range snap.Clients {
		total += c.Requests
	}
	if total != 100 {
		t.Fatalf("expected 100 requests, got %d", total)
	}
}

func TestSnapshotIsolation(t *testing.T) {
	tr := New()
	tr.Record("svc-1", []string{"admin"}, "logs-app")
	snap1 := tr.Snapshot()

	tr.Record("svc-2", []string{"read:logs-*"}, "logs-infra")
	snap2 := tr.Snapshot()

	if len(snap1.Clients) == len(snap2.Clients) && len(snap2.Clients) > 1 {
		t.Fatal("snapshots should reflect different client counts")
	}
}

func TestMultipleIndices(t *testing.T) {
	tr := New()
	tr.Record("svc-1", nil, "logs-app")
	tr.Record("svc-1", nil, "logs-infra")
	tr.Record("svc-1", nil, "logs-app")

	snap := tr.Snapshot()
	if len(snap.Indices) < 2 {
		t.Fatalf("expected at least 2 indices, got %d", len(snap.Indices))
	}
}
