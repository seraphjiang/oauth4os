package remotewrite

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestIngestAndExpose(t *testing.T) {
	r := New()
	wr := &WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"__name__": "cpu_usage", "host": "web-1"}, Samples: []Sample{{Value: 0.85, Timestamp: 1000}}},
		{Labels: map[string]string{"__name__": "mem_usage", "host": "web-1"}, Samples: []Sample{{Value: 0.60, Timestamp: 1000}}},
	}}
	n := r.Ingest(wr)
	if n != 2 {
		t.Fatalf("expected 2 ingested, got %d", n)
	}
	if r.SeriesCount() != 2 {
		t.Fatalf("expected 2 series, got %d", r.SeriesCount())
	}
	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	out := buf.String()
	if !strings.Contains(out, "cpu_usage") || !strings.Contains(out, "mem_usage") {
		t.Fatalf("expected both metrics:\n%s", out)
	}
}

func TestHandler(t *testing.T) {
	r := New()
	h := r.Handler()

	wr := WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"__name__": "test_metric"}, Samples: []Sample{{Value: 42}}},
	}}
	body, _ := json.Marshal(wr)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/write", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if r.SeriesCount() != 1 {
		t.Fatalf("expected 1 series, got %d", r.SeriesCount())
	}
}

func TestHandlerRejectsGet(t *testing.T) {
	r := New()
	h := r.Handler()
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodGet, "/api/v1/write", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHandlerRejectsInvalidJSON(t *testing.T) {
	r := New()
	h := r.Handler()
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest(http.MethodPost, "/api/v1/write", strings.NewReader("not json")))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCardinalityCap(t *testing.T) {
	r := New()
	for i := 0; i < MaxSeries+100; i++ {
		r.Ingest(&WriteRequest{Timeseries: []TimeSeries{
			{Labels: map[string]string{"__name__": "m", "id": labelKey(map[string]string{"i": string(rune(i))})}, Samples: []Sample{{Value: 1}}},
		}})
	}
	if r.SeriesCount() > MaxSeries {
		t.Fatalf("expected capped at %d, got %d", MaxSeries, r.SeriesCount())
	}
}

func TestMissingName(t *testing.T) {
	r := New()
	n := r.Ingest(&WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"host": "web-1"}, Samples: []Sample{{Value: 1}}},
	}})
	if n != 0 {
		t.Fatalf("expected 0 ingested for missing __name__, got %d", n)
	}
}

func TestOverwrite(t *testing.T) {
	r := New()
	r.Ingest(&WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"__name__": "x"}, Samples: []Sample{{Value: 1}}},
	}})
	r.Ingest(&WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"__name__": "x"}, Samples: []Sample{{Value: 99}}},
	}})
	if r.SeriesCount() != 1 {
		t.Fatalf("expected 1 series after overwrite, got %d", r.SeriesCount())
	}
	var buf bytes.Buffer
	r.WritePrometheus(&buf)
	if !strings.Contains(buf.String(), "99") {
		t.Fatal("expected latest value 99")
	}
}

func TestConcurrent(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				r.Ingest(&WriteRequest{Timeseries: []TimeSeries{
					{Labels: map[string]string{"__name__": "c", "g": string(rune(n))}, Samples: []Sample{{Value: float64(j)}}},
				}})
			}
		}(i)
	}
	wg.Wait()
}

func BenchmarkIngest(b *testing.B) {
	r := New()
	wr := &WriteRequest{Timeseries: []TimeSeries{
		{Labels: map[string]string{"__name__": "bench", "host": "h1"}, Samples: []Sample{{Value: 1}}},
	}}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			r.Ingest(wr)
		}
	})
}
