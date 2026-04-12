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
		go func(n int) {
			defer wg.Done()
			tr.Record("svc-1", []string{"read:logs-*"}, "logs-app")
		}(i)
	}
	wg.Wait()

	snap := tr.Snapshot()
	if snap.TotalRequests != 100 {
		t.Fatalf("expected 100 requests, got %d", snap.TotalRequests)
	}
}

func TestSnapshotIsolation(t *testing.T) {
	tr := New()
	tr.Record("svc-1", []string{"admin"}, "logs-app")
	snap1 := tr.Snapshot()

	tr.Record("svc-2", []string{"read:logs-*"}, "logs-infra")
	snap2 := tr.Snapshot()

	if snap1.TotalRequests == snap2.TotalRequests {
		t.Fatal("snapshots should reflect different states")
	}
}

func TestMultipleIndices(t *testing.T) {
	tr := New()
	tr.Record("svc-1", nil, "logs-app")
	tr.Record("svc-1", nil, "logs-infra")
	tr.Record("svc-1", nil, "logs-app")

	snap := tr.Snapshot()
	if snap.TotalRequests != 3 {
		t.Fatalf("expected 3 requests, got %d", snap.TotalRequests)
	}
}
