// Package events provides webhook notifications for token lifecycle events.
package events

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Type is a token event type.
type Type string

const (
	TokenIssued  Type = "token.issued"
	TokenRevoked Type = "token.revoked"
	TokenExpired Type = "token.expired"
	LoginSuccess Type = "login.success"
	LoginFailed  Type = "login.failed"
)

// Event is a token lifecycle event.
type Event struct {
	Type      Type   `json:"type"`
	ClientID  string `json:"client_id"`
	TokenID   string `json:"token_id,omitempty"`
	Scopes    []string `json:"scopes,omitempty"`
	IP        string `json:"ip,omitempty"`
	Timestamp string `json:"timestamp"`
}

// Notifier sends events to configured webhook URLs.
type Notifier struct {
	urls   []string
	client *http.Client
	ch     chan Event
}

// NewNotifier creates an event notifier. URLs are called async.
func NewNotifier(urls []string) *Notifier {
	n := &Notifier{
		urls:   urls,
		client: &http.Client{Timeout: 5 * time.Second},
		ch:     make(chan Event, 100),
	}
	go n.drain()
	return n
}

// Emit sends an event to all configured webhooks.
func (n *Notifier) Emit(e Event) {
	if len(n.urls) == 0 {
		return
	}
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	select {
	case n.ch <- e:
	default:
		// drop if buffer full
	}
}

func (n *Notifier) drain() {
	for e := range n.ch {
		body, _ := json.Marshal(e)
		for _, u := range n.urls {
			resp, err := n.client.Post(u, "application/json", bytes.NewReader(body))
			if err != nil {
				log.Printf("[events] webhook %s failed: %v", u, err)
				continue
			}
			resp.Body.Close()
		}
	}
}
