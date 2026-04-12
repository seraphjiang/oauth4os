package audit

import (
	"encoding/json"
	"io"
	"time"
)

type Auditor struct {
	w   io.Writer
	enc *json.Encoder
}

func NewAuditor(w io.Writer) *Auditor {
	return &Auditor{w: w, enc: json.NewEncoder(w)}
}

type LogEntry struct {
	Timestamp string   `json:"ts"`
	Level     string   `json:"level"`
	Event     string   `json:"event"`
	ClientID  string   `json:"client_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	Method    string   `json:"method,omitempty"`
	Path      string   `json:"path,omitempty"`
	RequestID string   `json:"request_id,omitempty"`
	Status    int      `json:"status,omitempty"`
	Latency   float64  `json:"latency_ms,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func (a *Auditor) Log(clientID string, scopes []string, method, path string) {
	a.Emit(LogEntry{
		Event:    "proxy_request",
		Level:    "info",
		ClientID: clientID,
		Scopes:   scopes,
		Method:   method,
		Path:     path,
	})
}

func (a *Auditor) LogAuth(clientID, event string, err error) {
	e := LogEntry{Event: event, Level: "info", ClientID: clientID}
	if err != nil {
		e.Level = "warn"
		e.Error = err.Error()
	}
	a.Emit(e)
}

func (a *Auditor) LogCedar(clientID, action, index, policy, reason string, allowed bool) {
	level := "info"
	if !allowed {
		level = "warn"
	}
	a.Emit(LogEntry{
		Event:    "cedar_eval",
		Level:    level,
		ClientID: clientID,
		Method:   action,
		Path:     index,
		Error:    reason,
	})
}

func (a *Auditor) Emit(e LogEntry) {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	a.enc.Encode(e)
}
