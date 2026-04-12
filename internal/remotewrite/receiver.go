// Package remotewrite provides a Prometheus-compatible remote write receiver.
// Accepts JSON-encoded metric samples via POST /api/v1/write.
// External apps push metrics here; oauth4os stores and exposes them on /metrics.
package remotewrite

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"
)

// MaxSeries caps stored series to prevent unbounded growth.
const MaxSeries = 10000

// Sample is a single metric data point.
type Sample struct {
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"` // unix millis, 0 = server time
}

// TimeSeries is a labeled metric with samples.
type TimeSeries struct {
	Labels  map[string]string `json:"labels"`
	Samples []Sample          `json:"samples"`
}

// WriteRequest is the JSON body for POST /api/v1/write.
type WriteRequest struct {
	Timeseries []TimeSeries `json:"timeseries"`
}

type storedSeries struct {
	labels map[string]string
	last   Sample
}

// Receiver stores the latest sample per unique label set.
type Receiver struct {
	mu     sync.RWMutex
	series map[string]*storedSeries // key = sorted label string
	count  int64
	errors int64
}

// New creates a remote write receiver.
func New() *Receiver {
	return &Receiver{series: make(map[string]*storedSeries)}
}

func labelKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var s string
	for _, k := range keys {
		s += k + "=" + labels[k] + ","
	}
	return s
}

// Ingest processes a write request.
func (r *Receiver) Ingest(req *WriteRequest) int {
	now := time.Now().UnixMilli()
	ingested := 0
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ts := range req.Timeseries {
		if ts.Labels == nil || ts.Labels["__name__"] == "" {
			continue
		}
		key := labelKey(ts.Labels)
		for _, s := range ts.Samples {
			if s.Timestamp == 0 {
				s.Timestamp = now
			}
			existing, ok := r.series[key]
			if !ok {
				if len(r.series) >= MaxSeries {
					continue
				}
				r.series[key] = &storedSeries{labels: ts.Labels, last: s}
			} else {
				existing.last = s
			}
			ingested++
		}
	}
	r.count += int64(ingested)
	return ingested
}

// Handler returns an http.HandlerFunc for POST /api/v1/write.
func (r *Receiver) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20)) // 1MB max
		if err != nil {
			r.mu.Lock()
			r.errors++
			r.mu.Unlock()
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}
		var wr WriteRequest
		if err := json.Unmarshal(body, &wr); err != nil {
			r.mu.Lock()
			r.errors++
			r.mu.Unlock()
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		n := r.Ingest(&wr)
		w.WriteHeader(http.StatusNoContent)
		_ = n
	}
}

// WritePrometheus outputs all stored series in Prometheus exposition format.
func (r *Receiver) WritePrometheus(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fmt.Fprintf(w, "# remote_write ingested series: %d, total samples: %d, errors: %d\n",
		len(r.series), r.count, r.errors)
	for _, ss := range r.series {
		name := ss.labels["__name__"]
		var labelStr string
		for k, v := range ss.labels {
			if k == "__name__" {
				continue
			}
			if labelStr != "" {
				labelStr += ","
			}
			labelStr += fmt.Sprintf("%s=%q", k, v)
		}
		if labelStr != "" {
			fmt.Fprintf(w, "%s{%s} %g %d\n", name, labelStr, ss.last.Value, ss.last.Timestamp)
		} else {
			fmt.Fprintf(w, "%s %g %d\n", name, ss.last.Value, ss.last.Timestamp)
		}
	}
}

// SeriesCount returns the number of stored series.
func (r *Receiver) SeriesCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.series)
}
