package auditexport

import (
	"encoding/json"
	"testing"
)

func BenchmarkAdd(b *testing.B) {
	u := &memUploader{data: map[string][]byte{}}
	e := New(u, "audit/", 0)
	defer e.Stop()
	entry := json.RawMessage(`{"action":"login","client":"app","ts":"2026-04-13T00:00:00Z"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Add(entry)
	}
}
