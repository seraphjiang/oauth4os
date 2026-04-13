package remotewrite

import (
	"fmt"
	"sync"
	"testing"
)

// Property: concurrent Ingest must not corrupt state or panic
func TestProperty_ConcurrentIngest(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				r.Ingest(&WriteRequest{
					Timeseries: []TimeSeries{{
						Labels:  map[string]string{"__name__": "concurrent", "worker": fmt.Sprintf("%d", n)},
						Samples: []Sample{{Value: float64(j), Timestamp: int64(j * 1000)}},
					}},
				})
			}
		}(i)
	}
	wg.Wait()
	if r.SeriesCount() == 0 {
		t.Error("must have ingested series")
	}
	if r.SeriesCount() > 50 {
		t.Errorf("expected ≤50 series (one per worker), got %d", r.SeriesCount())
	}
}
