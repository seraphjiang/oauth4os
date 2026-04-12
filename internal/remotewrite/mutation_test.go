package remotewrite

import (
	"bytes"
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
