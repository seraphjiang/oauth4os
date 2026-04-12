package introspect

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestActiveToken(t *testing.T) {
	now := time.Now()
	adapter := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			if id == "tok_valid" {
				return "client1", []string{"read:logs-*"}, now.Add(-time.Hour), now.Add(time.Hour), false, true
			}
			return "", nil, time.Time{}, time.Time{}, false, false
		},
	}
	h := NewHandler(adapter)

	req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader("token=tok_valid"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Active {
		t.Fatal("expected active=true")
	}
	if resp.ClientID != "client1" {
		t.Fatalf("expected client1, got %s", resp.ClientID)
	}
}

func TestRevokedToken(t *testing.T) {
	adapter := &ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "c", nil, time.Now(), time.Now().Add(time.Hour), true, true
		},
	}
	h := NewHandler(adapter)

	req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader("token=tok_revoked"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Active {
		t.Fatal("expected active=false for revoked token")
	}
}

func TestEmptyToken(t *testing.T) {
	h := NewHandler(&ManagerAdapter{
		GetToken: func(id string) (string, []string, time.Time, time.Time, bool, bool) {
			return "", nil, time.Time{}, time.Time{}, false, false
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/oauth/introspect", strings.NewReader("token="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Active {
		t.Fatal("expected active=false for empty token")
	}
}
