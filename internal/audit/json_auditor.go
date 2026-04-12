package audit

import (
	"encoding/json"
	"io"
	"time"
)

// Entry is a structured audit log entry.
type Entry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	ClientID  string `json:"client_id"`
	Scopes    []string `json:"scopes"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	RequestID string `json:"request_id,omitempty"`
	Status    int    `json:"status,omitempty"`
	DurationMs int64 `json:"duration_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// JSONAuditor writes structured JSON log lines.
type JSONAuditor struct {
	w   io.Writer
	enc *json.Encoder
}

// NewJSONAuditor creates a JSON auditor.
func NewJSONAuditor(w io.Writer) *JSONAuditor {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &JSONAuditor{w: w, enc: enc}
}

// Log writes a structured audit entry.
func (a *JSONAuditor) Log(clientID string, scopes []string, method, path string) {
	a.enc.Encode(Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "info",
		ClientID:  clientID,
		Scopes:    scopes,
		Method:    method,
		Path:      path,
	})
}

// LogWithDetails writes an audit entry with request ID, status, and duration.
func (a *JSONAuditor) LogWithDetails(clientID string, scopes []string, method, path, requestID string, status int, durationMs int64) {
	a.enc.Encode(Entry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Level:      "info",
		ClientID:   clientID,
		Scopes:     scopes,
		Method:     method,
		Path:       path,
		RequestID:  requestID,
		Status:     status,
		DurationMs: durationMs,
	})
}

// LogError writes an error audit entry.
func (a *JSONAuditor) LogError(clientID, method, path, requestID, errMsg string) {
	a.enc.Encode(Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     "error",
		ClientID:  clientID,
		Method:    method,
		Path:      path,
		RequestID: requestID,
		Error:     errMsg,
	})
}
