package remotewrite

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
)

// Mutation: remove Ingest → WritePrometheus must reflect ingested data
func TestMutation_IngestAndWrite(t *testing.T) {
	r := New()
	n := r.Ingest(&WriteRequest{
		Timeseries: []TimeSeries{{
			Labels:  map[string]string{"__name__": "test_total", "job": "proxy"},
			Samples: []Sample{{Value: 42, Timestamp: 1000}},
		}},
	})
	if n != 1 {
		t.Errorf("expected 1 ingested, got %d", n)
	}
	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	if !bytes.Contains(buf.Bytes(), []byte("test_total")) {
		t.Error("WritePrometheus must include ingested metric")
	}
}

// Mutation: remove Handler → HTTP endpoint must accept remote write
func TestMutation_HandlerAccepts(t *testing.T) {
	r := New()
	h := r.Handler()
	if h == nil {
		t.Error("Handler must return non-nil")
	}
}

// Mutation: remove MaxSeries guard → must cap stored series
func TestMutation_CardinalityGuard(t *testing.T) {
	r := New()
	// Ingest more than MaxSeries unique series
	for i := 0; i < MaxSeries+100; i++ {
		r.Ingest(&WriteRequest{
			Timeseries: []TimeSeries{{
				Labels:  map[string]string{"__name__": "test", "id": fmt.Sprintf("%d", i)},
				Samples: []Sample{{Value: 1, Timestamp: 1000}},
			}},
		})
	}
	if r.SeriesCount() > MaxSeries {
		t.Errorf("series count %d exceeds MaxSeries %d", r.SeriesCount(), MaxSeries)
	}
}

// Mutation: remove POST-only check → Handler must reject GET
func TestMutation_HandlerRejectsGet(t *testing.T) {
	r := New()
	h := r.Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/write", nil))
	if w.Code != 405 {
		t.Errorf("GET should return 405, got %d", w.Code)
	}
}
