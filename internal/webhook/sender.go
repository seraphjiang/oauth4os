package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// Sender sends signed webhook requests with HMAC-SHA256.
type Sender struct {
	Secret string
	Client *http.Client
}

// NewSender creates a webhook sender with the given HMAC secret.
func NewSender(secret string) *Sender {
	return &Sender{
		Secret: secret,
		Client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Sign computes HMAC-SHA256 of body with the secret.
func (s *Sender) Sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(s.Secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks an HMAC-SHA256 signature against body.
func (s *Sender) Verify(body []byte, signature string) bool {
	expected := s.Sign(body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// Send posts body to url with X-Webhook-Signature header.
func (s *Sender) Send(url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "sha256="+s.Sign(body))
	return s.Client.Do(req)
}
