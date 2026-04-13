// Package otlp provides OTLP-compatible JSON export for traces.
// Spans are buffered in a ring buffer and served at GET /v1/traces.
// Compatible with OpenTelemetry Collector's OTLP/HTTP receiver.
package otlp

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// Span is an OTLP-compatible span.
type Span struct {
	TraceID    string            `json:"traceId"`
	SpanID     string            `json:"spanId"`
	ParentID   string            `json:"parentSpanId,omitempty"`
	Name       string            `json:"name"`
	Kind       int               `json:"kind"` // 1=internal, 2=server, 3=client
	StartNano  int64             `json:"startTimeUnixNano"`
	EndNano    int64             `json:"endTimeUnixNano"`
	Attributes []Attribute       `json:"attributes,omitempty"`
	Status     *Status           `json:"status,omitempty"`
}

type Attribute struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

type AttributeValue struct {
	StringValue string `json:"stringValue,omitempty"`
	IntValue    int64  `json:"intValue,omitempty"`
}

type Status struct {
	Code    int    `json:"code"` // 0=unset, 1=ok, 2=error
	Message string `json:"message,omitempty"`
}

// Exporter collects spans and serves them as OTLP JSON.
type Exporter struct {
	mu    sync.Mutex
	spans []Span
	max   int
}

// New creates an exporter with a ring buffer of given size.
func New(maxSpans int) *Exporter {
	return &Exporter{spans: make([]Span, 0, maxSpans), max: maxSpans}
}

// Record adds a span.
func (e *Exporter) Record(name string, start, end time.Time, attrs map[string]string, errMsg string) {
	s := Span{
		TraceID:   randomHex(16),
		SpanID:    randomHex(8),
		Name:      name,
		Kind:      2, // server
		StartNano: start.UnixNano(),
		EndNano:   end.UnixNano(),
	}
	for k, v := range attrs {
		s.Attributes = append(s.Attributes, Attribute{Key: k, Value: AttributeValue{StringValue: v}})
	}
	if errMsg != "" {
		s.Status = &Status{Code: 2, Message: errMsg}
	} else {
		s.Status = &Status{Code: 1}
	}
	e.mu.Lock()
	if e.max > 0 && len(e.spans) >= e.max {
		e.spans = e.spans[1:]
	}
	e.spans = append(e.spans, s)
	e.mu.Unlock()
}

// Handler serves GET /v1/traces in OTLP JSON format.
func (e *Exporter) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		spans := make([]Span, len(e.spans))
		copy(spans, e.spans)
		e.mu.Unlock()

		resp := map[string]interface{}{
			"resourceSpans": []map[string]interface{}{{
				"resource": map[string]interface{}{
					"attributes": []Attribute{
						{Key: "service.name", Value: AttributeValue{StringValue: "oauth4os"}},
					},
				},
				"scopeSpans": []map[string]interface{}{{
					"scope": map[string]interface{}{"name": "oauth4os.proxy"},
					"spans": spans,
				}},
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
