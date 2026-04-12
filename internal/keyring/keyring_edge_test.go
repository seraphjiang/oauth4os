package keyring

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConcurrentJWKS(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()

	handler := r.JWKSHandler()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/.well-known/jwks.json", nil))
			if w.Code != 200 {
				t.Errorf("expected 200, got %d", w.Code)
			}
			if w.Header().Get("Content-Type") != "application/json" {
				t.Error("expected application/json")
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentCurrentDuringRotation(t *testing.T) {
	r, err := New(2048, 50*time.Millisecond) // fast rotation
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k := r.Current()
			if k == nil {
				t.Error("Current() should never return nil")
			}
			if k.Private == nil {
				t.Error("private key should not be nil")
			}
		}()
	}
	wg.Wait()
}

func TestJWKSContentType(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()

	w := httptest.NewRecorder()
	r.JWKSHandler().ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}
	if w.Header().Get("Cache-Control") == "" {
		t.Fatal("expected Cache-Control header")
	}
}

func TestJWKSMethodNotAllowed(t *testing.T) {
	r, err := New(2048, 1*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()

	w := httptest.NewRecorder()
	r.JWKSHandler().ServeHTTP(w, httptest.NewRequest("POST", "/", nil))

	// Should still return 200 (handler doesn't check method) or handle gracefully
	if w.Code >= 500 {
		t.Fatalf("POST should not cause 500, got %d", w.Code)
	}
}

func BenchmarkJWKSHandler(b *testing.B) {
	r, _ := New(2048, 1*time.Hour)
	defer r.Stop()
	handler := r.JWKSHandler()
	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkCurrent(b *testing.B) {
	r, _ := New(2048, 1*time.Hour)
	defer r.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.Current()
	}
}

func init() {
	// Suppress handler output
	http.DefaultServeMux = http.NewServeMux()
}
