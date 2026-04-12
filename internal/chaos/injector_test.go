package chaos

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestMutation_DisabledPassthrough(t *testing.T) {
	inj := New(Config{ErrorRate: 1.0})
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 200 {
		t.Errorf("disabled injector must pass through, got %d", w.Code)
	}
}

func TestMutation_ErrorInjection(t *testing.T) {
	inj := New(Config{ErrorRate: 1.0})
	inj.Enable()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code != 500 {
		t.Errorf("100%% error rate must return 500, got %d", w.Code)
	}
}

func TestMutation_LatencyInjection(t *testing.T) {
	inj := New(Config{LatencyRate: 1.0, LatencyMin: 50 * time.Millisecond, LatencyMax: 60 * time.Millisecond})
	inj.Enable()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	start := time.Now()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if time.Since(start) < 40*time.Millisecond {
		t.Error("latency injection must delay request")
	}
}

func TestMutation_SetConfig(t *testing.T) {
	inj := New(Config{ErrorRate: 0})
	inj.SetConfig(Config{ErrorRate: 0.5})
	cfg := inj.GetConfig()
	if cfg.ErrorRate != 0.5 {
		t.Errorf("expected ErrorRate 0.5, got %f", cfg.ErrorRate)
	}
}

func TestProperty_ConcurrentRequests(t *testing.T) {
	inj := New(Config{ErrorRate: 0.3, LatencyRate: 0.3, LatencyMin: time.Millisecond, LatencyMax: 5 * time.Millisecond})
	inj.Enable()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	handler := inj.Middleware(inner)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		}()
	}
	wg.Wait()
}
