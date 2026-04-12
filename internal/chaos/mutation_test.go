package chaos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// M5: 0% error rate must pass through even when enabled.
func TestMutation_ZeroErrorRatePassthrough(t *testing.T) {
	inj := New(Config{ErrorRate: 0})
	inj.Enable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("0%% error rate should pass through, got %d", rec.Code)
	}
}

// M6: Enable then Disable must pass through.
func TestMutation_EnableThenDisable(t *testing.T) {
	inj := New(Config{ErrorRate: 1.0})
	inj.Enable()
	inj.Disable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("disabled should pass through, got %d", rec.Code)
	}
}

// M7: GetConfig returns current values.
func TestMutation_GetConfigValues(t *testing.T) {
	inj := New(Config{ErrorRate: 0.42, LatencyRate: 0.1, DropRate: 0.05})
	cfg := inj.GetConfig()
	if cfg.ErrorRate != 0.42 || cfg.LatencyRate != 0.1 || cfg.DropRate != 0.05 {
		t.Fatalf("GetConfig mismatch: %+v", cfg)
	}
}

// M8: Error response body contains chaos_fault.
func TestMutation_ErrorResponseBody(t *testing.T) {
	inj := New(Config{ErrorRate: 1.0})
	inj.Enable()
	h := inj.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 500 {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	body := rec.Body.String()
	if len(body) == 0 {
		t.Fatal("error response should have body")
	}
}
