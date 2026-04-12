package exchange

import (
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
)

type safeIssuer struct {
	calls atomic.Int64
}

func (i *safeIssuer) IssueExchangeToken(subject, issuer string, scopes []string) (string, int) {
	i.calls.Add(1)
	return "tok_exchanged", 3600
}

type safeValidator struct{}

func (v *safeValidator) ValidateSubject(token string) (*SubjectClaims, error) {
	return &SubjectClaims{Subject: "user1", Issuer: "https://idp.example.com", Scopes: []string{"read:logs-*"}}, nil
}

func TestConcurrentExchange(t *testing.T) {
	h := NewHandler(&safeValidator{}, &safeIssuer{}, "https://proxy.example.com")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := postForm(h, url.Values{
				"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
				"subject_token":      {"valid-jwt"},
				"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
			})
			if w.Code != 200 {
				t.Errorf("expected 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()
}

func TestExchangeMissingSubjectToken(t *testing.T) {
	h := newTestHandler(&SubjectClaims{Subject: "user1"}, nil)
	w := postForm(h, url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
	})
	if w.Code == 200 {
		t.Fatal("missing subject_token should fail")
	}
}

func TestExchangeEmptyBody(t *testing.T) {
	h := newTestHandler(&SubjectClaims{Subject: "user1"}, nil)
	w := postForm(h, url.Values{})
	if w.Code == 200 {
		t.Fatal("empty body should fail")
	}
}
