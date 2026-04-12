// Package auditexport batches audit log entries and uploads to S3.
package auditexport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Uploader writes bytes to a destination (S3, file, etc).
type Uploader interface {
	Upload(key string, data []byte) error
}

// Exporter batches audit entries and flushes to an Uploader.
type Exporter struct {
	mu       sync.Mutex
	buf      []json.RawMessage
	uploader Uploader
	prefix   string
	interval time.Duration
	stopCh   chan struct{}
	OnFlush  func(count int, key string)
}

// New creates an exporter that flushes every interval.
func New(uploader Uploader, prefix string, interval time.Duration) *Exporter {
	e := &Exporter{
		uploader: uploader,
		prefix:   prefix,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
	if interval > 0 {
		go e.loop()
	}
	return e
}

// Add queues an audit entry for export.
func (e *Exporter) Add(entry json.RawMessage) {
	e.mu.Lock()
	e.buf = append(e.buf, entry)
	e.mu.Unlock()
}

// Flush uploads buffered entries now.
func (e *Exporter) Flush() error {
	e.mu.Lock()
	if len(e.buf) == 0 {
		e.mu.Unlock()
		return nil
	}
	batch := e.buf
	e.buf = nil
	e.mu.Unlock()

	var out bytes.Buffer
	for _, entry := range batch {
		out.Write(entry)
		out.WriteByte('\n')
	}

	key := fmt.Sprintf("%s/%s.ndjson", e.prefix, time.Now().UTC().Format("2006/01/02/15-04-05"))
	err := e.uploader.Upload(key, out.Bytes())
	if err == nil && e.OnFlush != nil {
		e.OnFlush(len(batch), key)
	}
	return err
}

// Stop halts the flush loop and does a final flush.
func (e *Exporter) Stop() error {
	close(e.stopCh)
	return e.Flush()
}

func (e *Exporter) loop() {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.Flush()
		case <-e.stopCh:
			return
		}
	}
}
