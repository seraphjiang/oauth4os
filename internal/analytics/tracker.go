// Package analytics tracks token usage patterns for operator dashboards.
package analytics

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Tracker collects per-client, per-scope, and per-index request metrics.
type Tracker struct {
	clients map[string]*clientStats
	scopes  map[string]*atomic.Int64
	indices map[string]*atomic.Int64
	mu      sync.RWMutex
}

type clientStats struct {
	Requests atomic.Int64
	LastSeen atomic.Value // time.Time
}

// New creates an analytics tracker.
func New() *Tracker {
	return &Tracker{
		clients: make(map[string]*clientStats),
		scopes:  make(map[string]*atomic.Int64),
		indices: make(map[string]*atomic.Int64),
	}
}

// Record tracks a single request.
func (t *Tracker) Record(clientID string, scopes []string, index string) {
	t.mu.Lock()
	cs, ok := t.clients[clientID]
	if !ok {
		cs = &clientStats{}
		t.clients[clientID] = cs
	}
	for _, s := range scopes {
		if _, ok := t.scopes[s]; !ok {
			t.scopes[s] = &atomic.Int64{}
		}
		t.scopes[s].Add(1)
	}
	if index != "" {
		if _, ok := t.indices[index]; !ok {
			t.indices[index] = &atomic.Int64{}
		}
		t.indices[index].Add(1)
	}
	t.mu.Unlock()

	cs.Requests.Add(1)
	cs.LastSeen.Store(time.Now())
}

// Snapshot returns current analytics data.
func (t *Tracker) Snapshot() Report {
	t.mu.RLock()
	defer t.mu.RUnlock()

	r := Report{
		Clients: make([]ClientEntry, 0, len(t.clients)),
		Scopes:  make([]CountEntry, 0, len(t.scopes)),
		Indices: make([]CountEntry, 0, len(t.indices)),
	}

	for id, cs := range t.clients {
		var lastSeen time.Time
		if v := cs.LastSeen.Load(); v != nil {
			lastSeen = v.(time.Time)
		}
		r.Clients = append(r.Clients, ClientEntry{
			ClientID: id,
			Requests: cs.Requests.Load(),
			LastSeen: lastSeen,
		})
	}
	for name, cnt := range t.scopes {
		r.Scopes = append(r.Scopes, CountEntry{Name: name, Count: cnt.Load()})
	}
	for name, cnt := range t.indices {
		r.Indices = append(r.Indices, CountEntry{Name: name, Count: cnt.Load()})
	}

	// Sort all by count descending
	sort.Slice(r.Clients, func(i, j int) bool { return r.Clients[i].Requests > r.Clients[j].Requests })
	sort.Slice(r.Scopes, func(i, j int) bool { return r.Scopes[i].Count > r.Scopes[j].Count })
	sort.Slice(r.Indices, func(i, j int) bool { return r.Indices[i].Count > r.Indices[j].Count })

	return r
}

// Report is the JSON response for GET /admin/analytics.
type Report struct {
	Clients []ClientEntry `json:"top_clients"`
	Scopes  []CountEntry  `json:"scope_distribution"`
	Indices []CountEntry  `json:"top_indices"`
}

type ClientEntry struct {
	ClientID string    `json:"client_id"`
	Requests int64     `json:"requests"`
	LastSeen time.Time `json:"last_seen"`
}

type CountEntry struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}
