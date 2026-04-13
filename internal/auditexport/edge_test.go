package auditexport

import (
	"encoding/json"
	"testing"
)

func TestEdge_AddAndFlush(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "test/", 0)
	defer e.Stop()
	e.Add(json.RawMessage(`{"action":"login"}`))
	e.Add(json.RawMessage(`{"action":"logout"}`))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	if len(u.data) == 0 {
		t.Error("Flush should upload data")
	}
}

func TestEdge_FlushEmptyNoError(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "test/", 0)
	defer e.Stop()
	if err := e.Flush(); err != nil {
		t.Errorf("flush empty should not error: %v", err)
	}
}

func TestEdge_StopIdempotent(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "test/", 0)
	e.Stop()
	e.Stop() // double stop must not panic
}

func TestEdge_AddAfterFlush(t *testing.T) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "test/", 0)
	defer e.Stop()
	e.Add(json.RawMessage(`{"a":1}`))
	e.Flush()
	e.Add(json.RawMessage(`{"b":2}`))
	e.Flush()
	if len(u.data) < 1 {
		t.Error("should have at least 1 upload")
	}
}
