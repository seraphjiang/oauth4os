// Package logging provides structured JSON logging for oauth4os.
//
// Usage:
//
//	logger := logging.New(os.Stdout, "info")
//	logger.Info("request proxied", "method", "GET", "path", "/logs-*/_search", "status", 200)
//
// Output:
//
//	{"time":"2025-04-12T03:00:00Z","level":"INFO","msg":"request proxied","method":"GET","path":"/logs-*/_search","status":200}
package logging

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger struct {
	out   io.Writer
	level Level
	mu    sync.Mutex
}

func New(out io.Writer, level string) *Logger {
	if out == nil {
		out = os.Stdout
	}
	return &Logger{out: out, level: parseLevel(level)}
}

func parseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return DEBUG
	case "warn", "warning":
		return WARN
	case "error":
		return ERROR
	case "fatal":
		return FATAL
	default:
		return INFO
	}
}

func (l *Logger) log(level Level, name string, msg string, kvs ...any) {
	if level < l.level {
		return
	}
	entry := map[string]any{
		"time":  time.Now().UTC().Format(time.RFC3339),
		"level": name,
		"msg":   msg,
	}
	for i := 0; i+1 < len(kvs); i += 2 {
		if k, ok := kvs[i].(string); ok {
			entry[k] = kvs[i+1]
		}
	}
	data, _ := json.Marshal(entry)
	data = append(data, '\n')
	l.mu.Lock()
	l.out.Write(data)
	l.mu.Unlock()
}

func (l *Logger) Debug(msg string, kvs ...any) { l.log(DEBUG, "DEBUG", msg, kvs...) }
func (l *Logger) Info(msg string, kvs ...any)  { l.log(INFO, "INFO", msg, kvs...) }
func (l *Logger) Warn(msg string, kvs ...any)  { l.log(WARN, "WARN", msg, kvs...) }
func (l *Logger) Error(msg string, kvs ...any) { l.log(ERROR, "ERROR", msg, kvs...) }
func (l *Logger) Fatal(msg string, kvs ...any) {
	l.log(FATAL, "FATAL", msg, kvs...)
	os.Exit(1)
}
