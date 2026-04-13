package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Edge: disabled injector passes through
func TestEdge_DisabledPassthrough(t *testing.T) {
	inj := New()
	inj.Disable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("disabled injector should pass through, got %d", w.Code)
		}
	}
}

// Edge: enabled with 100% error rate always fails
func TestEdge_FullErrorRate(t *testing.T) {
	inj := New()
	inj.Enable()
	inj.SetConfig(Config{ErrorRate: 1.0})
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code == 200 {
		t.Error("100% error rate should inject fault")
	}
}

// Edge: GetConfig returns current config
func TestEdge_GetConfigReturns(t *testing.T) {
	inj := New()
	inj.SetConfig(Config{ErrorRate: 0.5})
	cfg := inj.GetConfig()
	if cfg.ErrorRate != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.ErrorRate)
	}
}
