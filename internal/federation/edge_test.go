package federation

import (
	"net/http/httptest"
	"testing"
)

// Edge: Route with nil clusters returns nil handler
func TestEdge_NilClustersRoute(t *testing.T) {
	f := New(nil, nil)
	r := httptest.NewRequest("GET", "/", nil)
	h := f.Route(r)
	if h != nil {
		t.Error("nil clusters should return nil route")
	}
}
