package analytics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordAndStats(t *testing.T) {
	tr := NewTracker(100)
	tr.Record("agent-1", []string{"read:logs-*"}, "GET", "/logs/_search")
	tr.Record("agent-1", []string{"read:logs-*"}, "GET", "/logs/_search")
	tr.Record("agent-2", []string{"admin"}, "PUT", "/settings")
	tr.RecordDenied()

	s := tr.GetStats(10)
	if s.TotalRequests != 3 {
		t.Fatalf("expected 3 total, got %d", s.TotalRequests)
	}
	if s.TotalDenied != 1 {
		t.Fatalf("expected 1 denied, got %d", s.TotalDenied)
	}
	if s.TopClients[0].ClientID != "agent-1" || s.TopClients[0].Requests != 2 {
		t.Fatalf("expected agent-1 with 2 requests at top, got %+v", s.TopClients[0])
	}
	if s.ScopeDistro["read:logs-*"] != 2 {
		t.Fatalf("expected read:logs-* count 2, got %d", s.ScopeDistro["read:logs-*"])
	}
}

func TestHandler(t *testing.T) {
	tr := NewTracker(100)
	tr.Record("c1", []string{"read:logs-*"}, "GET", "/logs")

	w := httptest.NewRecorder()
	tr.Handler(w, httptest.NewRequest(http.MethodGet, "/oauth/analytics", nil))

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var s Stats
	json.NewDecoder(w.Body).Decode(&s)
	if s.TotalRequests != 1 {
		t.Fatalf("expected 1, got %d", s.TotalRequests)
	}
}

func TestMaxEventsCaped(t *testing.T) {
	tr := NewTracker(5)
	for i := 0; i < 20; i++ {
		tr.Record("c1", nil, "GET", "/")
	}
	s := tr.GetStats(10)
	if s.RecentEvents != 5 {
		t.Fatalf("expected 5 stored events, got %d", s.RecentEvents)
	}
	if s.TotalRequests != 20 {
		t.Fatalf("expected 20 total, got %d", s.TotalRequests)
	}
}
