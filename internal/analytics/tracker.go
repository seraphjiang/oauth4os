// Package analytics tracks token usage patterns per client.
// Provides GET /oauth/analytics with usage stats, top clients, scope distribution.
package analytics

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Event records a single token use.
type Event struct {
	ClientID string
	Scope    string
	Action   string
	Resource string
	At       time.Time
}

// Tracker collects token usage analytics.
type Tracker struct {
	mu       sync.Mutex
	events   []Event
	clients  map[string]*clientStats
	scopes   map[string]*atomic.Int64
	total    atomic.Int64
	denied   atomic.Int64
	maxEvents int
}

type clientStats struct {
	Requests atomic.Int64
	LastSeen atomic.Value // time.Time
}

// NewTracker creates an analytics tracker.
func NewTracker(maxEvents int) *Tracker {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	return &Tracker{
		clients:   make(map[string]*clientStats),
		scopes:    make(map[string]*atomic.Int64),
		maxEvents: maxEvents,
	}
}

// Record logs a token usage event.
func (t *Tracker) Record(clientID string, scopes []string, action, resource string) {
	t.total.Add(1)
	now := time.Now()

	t.mu.Lock()
	cs, ok := t.clients[clientID]
	if !ok {
		cs = &clientStats{}
		t.clients[clientID] = cs
	}
	for _, s := range scopes {
		cnt, ok := t.scopes[s]
		if !ok {
			cnt = &atomic.Int64{}
			t.scopes[s] = cnt
		}
		cnt.Add(1)
	}
	if len(t.events) < t.maxEvents {
		t.events = append(t.events, Event{ClientID: clientID, Action: action, Resource: resource, At: now})
	}
	t.mu.Unlock()

	cs.Requests.Add(1)
	cs.LastSeen.Store(now)
}

// RecordDenied logs a denied request.
func (t *Tracker) RecordDenied() {
	t.denied.Add(1)
}

// Stats is the analytics response.
type Stats struct {
	TotalRequests  int64          `json:"total_requests"`
	TotalDenied    int64          `json:"total_denied"`
	UniqueClients  int            `json:"unique_clients"`
	TopClients     []ClientUsage  `json:"top_clients"`
	ScopeDistro    map[string]int64 `json:"scope_distribution"`
	RecentEvents   int            `json:"recent_events_stored"`
}

// ClientUsage is per-client usage info.
type ClientUsage struct {
	ClientID string `json:"client_id"`
	Requests int64  `json:"requests"`
	LastSeen string `json:"last_seen"`
}

// GetStats returns current analytics.
func (t *Tracker) GetStats(topN int) Stats {
	if topN <= 0 {
		topN = 10
	}
	t.mu.Lock()
	var clients []ClientUsage
	for id, cs := range t.clients {
		ls, _ := cs.LastSeen.Load().(time.Time)
		clients = append(clients, ClientUsage{
			ClientID: id,
			Requests: cs.Requests.Load(),
			LastSeen: ls.Format(time.RFC3339),
		})
	}
	scopeDist := make(map[string]int64)
	for s, cnt := range t.scopes {
		scopeDist[s] = cnt.Load()
	}
	eventCount := len(t.events)
	t.mu.Unlock()

	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Requests > clients[j].Requests
	})
	if len(clients) > topN {
		clients = clients[:topN]
	}

	return Stats{
		TotalRequests: t.total.Load(),
		TotalDenied:   t.denied.Load(),
		UniqueClients: len(clients),
		TopClients:    clients,
		ScopeDistro:   scopeDist,
		RecentEvents:  eventCount,
	}
}

// Handler serves GET /oauth/analytics.
func (t *Tracker) Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t.GetStats(10))
}
