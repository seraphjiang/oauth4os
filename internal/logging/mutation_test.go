package logging

import (
	"bytes"
	"strings"
	"testing"
)

// Mutation: remove level filtering → Debug must not appear at INFO level
func TestMutation_LevelFilter(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.Debug("should not appear")
	if buf.Len() > 0 {
		t.Error("DEBUG message should not appear at INFO level")
	}
}

// Mutation: remove output → Info must produce output at INFO level
func TestMutation_InfoOutput(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.Info("hello")
	if buf.Len() == 0 {
		t.Error("INFO message should appear at INFO level")
	}
}

// Mutation: remove level name → output must contain level name
func TestMutation_LevelInOutput(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "debug")
	l.Warn("test warning")
	if !bytes.Contains(buf.Bytes(), []byte("WARN")) {
		t.Error("output must contain level name WARN")
	}
}

// Mutation: remove debug level → debug messages must be suppressed at info level
func TestMutation_DebugSuppressed(t *testing.T) {
	var buf bytes.Buffer
	l := New(&buf, "info")
	l.log(0, "test", "debug message") // Level 0 = debug
	if strings.Contains(buf.String(), "debug message") {
		t.Error("debug messages must be suppressed at info level")
	}
}
