package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// OTLPTracer exports spans to an OTLP HTTP endpoint.
type OTLPTracer struct {
	endpoint string
	client   *http.Client
	batch    []*Span
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
}

// NewOTLPTracer creates a tracer that batches spans and exports to the OTLP endpoint.
func NewOTLPTracer(endpoint string) *OTLPTracer {
	t := &OTLPTracer{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 5 * time.Second},
		stop:     make(chan struct{}),
	}
	go t.flush()
	return t
}

func (t *OTLPTracer) StartSpan(ctx context.Context, name string, attrs map[string]string) (context.Context, *Span) {
	span := &Span{
		TraceID:   traceIDFromContext(ctx),
		SpanID:    genID(8),
		Name:      name,
		StartTime: time.Now(),
		Attrs:     attrs,
	}
	if parent := FromContext(ctx); parent != nil {
		span.ParentID = parent.SpanID
		span.TraceID = parent.TraceID
	}
	return context.WithValue(ctx, spanKey{}, span), span
}

func (t *OTLPTracer) EndSpan(span *Span, status string) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	span.Status = status
	t.mu.Lock()
	t.batch = append(t.batch, span)
	t.mu.Unlock()
}

// Stop flushes remaining spans and stops the exporter.
func (t *OTLPTracer) Stop() {
	t.stopOnce.Do(func() { close(t.stop) })
	t.export()
}

func (t *OTLPTracer) flush() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.export()
		case <-t.stop:
			return
		}
	}
}

func (t *OTLPTracer) export() {
	t.mu.Lock()
	if len(t.batch) == 0 {
		t.mu.Unlock()
		return
	}
	spans := t.batch
	t.batch = nil
	t.mu.Unlock()

	body, _ := json.Marshal(spans)
	req, err := http.NewRequest("POST", t.endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}
