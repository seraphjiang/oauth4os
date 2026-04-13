package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEdge_DisabledPassthrough(t *testing.T) {
	inj := New(Config{ErrorRate: 0})
	inj.Disable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 200 {
			t.Errorf("disabled should pass through, got %d", w.Code)
		}
	}
}

func TestEdge_FullErrorRate(t *testing.T) {
	inj := New(Config{ErrorRate: 1.0})
	inj.Enable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if w.Code == 200 {
		t.Error("100% error rate should inject fault")
	}
}

func TestEdge_GetConfigReturns(t *testing.T) {
	inj := New(Config{ErrorRate: 0.5})
	cfg := inj.GetConfig()
	if cfg.ErrorRate != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.ErrorRate)
	}
}
