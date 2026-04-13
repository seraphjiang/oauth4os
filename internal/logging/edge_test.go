package logging

import (
	"bytes"
	"testing"
)

// Edge: Info level logs info messages
func TestEdge_InfoLevelLogs(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.Info("test message")
	if buf.Len() == 0 {
		t.Error("info message should be logged at info level")
	}
}

// Edge: Info level suppresses debug
func TestEdge_InfoSuppressesDebug(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.Debug("debug message")
	if buf.Len() != 0 {
		t.Error("debug should be suppressed at info level")
	}
}

// Edge: Error level always logs
func TestEdge_ErrorAlwaysLogs(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "error")
	l.Error("error message")
	if buf.Len() == 0 {
		t.Error("error should always be logged")
	}
}

// Edge: output contains message text
func TestEdge_OutputContainsMessage(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "debug")
	l.Info("unique-marker-12345")
	if !bytes.Contains(buf.Bytes(), []byte("unique-marker-12345")) {
		t.Error("output should contain the message text")
	}
}
