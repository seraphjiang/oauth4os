// Package events provides webhook notifications for token lifecycle events.
package events

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// EventType identifies the token event.
type EventType string

const (
	TokenIssued  EventType = "token.issued"
	TokenRevoked EventType = "token.revoked"
	TokenRefresh EventType = "token.refreshed"
	ClientReg    EventType = "client.registered"
	ClientDel    EventType = "client.deleted"
)

// Event is the webhook payload.
type Event struct {
	Type      EventType `json:"type"`
	ClientID  string    `json:"client_id"`
	Scopes    []string  `json:"scopes,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id,omitempty"`
}

// Notifier sends events to configured webhook URLs.
type Notifier struct {
	urls      []string
	client    *http.Client
	ch        chan Event
	done      chan struct{}
	signKey   []byte // HMAC-SHA256 signing key (optional)
}

// New creates a notifier that posts events to the given URLs.
// Events are sent asynchronously via a buffered channel.
func New(urls []string) *Notifier {
	n := &Notifier{
		urls:   urls,
		client: &http.Client{Timeout: 5 * time.Second},
		ch:     make(chan Event, 100),
		done:   make(chan struct{}),
	}
	go n.drain()
	return n
}

// Stop closes the event channel and waits for drain to finish.
func (n *Notifier) Stop() {
	close(n.ch)
	<-n.done
}

// SetSigningKey enables HMAC-SHA256 signature on outgoing webhooks.
// The signature is sent in the X-Webhook-Signature header.
func (n *Notifier) SetSigningKey(key []byte) {
	n.signKey = key
}

// Emit queues an event for delivery. Non-blocking; drops if buffer full.
func (n *Notifier) Emit(e Event) {
	if len(n.urls) == 0 {
		return
	}
	e.Timestamp = time.Now()
	select {
	case n.ch <- e:
	default: // drop if full
	}
}

func (n *Notifier) drain() {
	defer close(n.done)
	for e := range n.ch {
		body, _ := json.Marshal(e)
		for _, u := range n.urls {
			req, err := http.NewRequest("POST", u, bytes.NewReader(body))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")
			if len(n.signKey) > 0 {
				mac := hmac.New(sha256.New, n.signKey)
				mac.Write(body)
				req.Header.Set("X-Webhook-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
			}
			resp, err := n.client.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	}
}
