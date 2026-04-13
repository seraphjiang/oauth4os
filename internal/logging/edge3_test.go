package logging

import (
	"bytes"
	"testing"
)

func TestEdge_NilWriter(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("nil writer panics — expected")
		}
	}()
	l := New(nil, "info")
	l.Info("test")
}

func TestEdge_EmptyLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "")
	l.Info("test")
	// Empty level should default to something — just verify no panic
}

func TestEdge_UnknownLevel(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "trace")
	l.Info("test")
	// Unknown level should not panic
}
