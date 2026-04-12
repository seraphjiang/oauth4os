package device

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// FuzzDeviceCode ensures the code endpoint never panics on arbitrary form data.
func FuzzDeviceCode(f *testing.F) {
	f.Add("client_id=test&scope=admin")
	f.Add("")
	f.Add("client_id=")
	f.Add("scope=read:logs-*")
	f.Add("client_id=x&scope=a b c d e")
	f.Add("client_id=" + strings.Repeat("A", 10000))
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(func(clientID string, scopes []string) (string, string) {
			return "tok", "rtk"
		})
		mux := http.NewServeMux()
		h.Register(mux)
		r := httptest.NewRequest("POST", "/oauth/device/code", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
	})
}

// FuzzDeviceToken ensures the token poll endpoint never panics.
func FuzzDeviceToken(f *testing.F) {
	f.Add("grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=abc")
	f.Add("")
	f.Add("device_code=nonexistent")
	f.Add("grant_type=invalid")
	f.Fuzz(func(t *testing.T, body string) {
		h := NewHandler(func(clientID string, scopes []string) (string, string) {
			return "tok", "rtk"
		})
		mux := http.NewServeMux()
		h.Register(mux)
		r := httptest.NewRequest("POST", "/oauth/device/token", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
	})
}
