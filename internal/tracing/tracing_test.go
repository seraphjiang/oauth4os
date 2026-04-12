package tracing

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestStartRequestAndStages(t *testing.T) {
	var exported *Span
	tr := &Tracer{enabled: true, exportFunc: func(s *Span) { exported = s }}

	req, _ := http.NewRequest("GET", "/logs-*/_search", nil)
	ctx, root := tr.StartRequest(context.Background(), req)

	jwt := tr.StartStage(ctx, "jwt")
	tr.End(jwt, nil)

	scope := tr.StartStage(ctx, "scope")
	tr.End(scope, nil)

	cedar := tr.StartStage(ctx, "cedar")
	tr.End(cedar, errors.New("denied"))

	tr.FinishRequest(root, 403)

	if exported == nil {
		t.Fatal("expected span to be exported")
	}
	if len(exported.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(exported.Children))
	}
	if exported.Children[2].Status != "error" {
		t.Error("expected cedar span to have error status")
	}
	if exported.Status != "error" {
		t.Error("expected root span to have error status for 403")
	}
}

func TestDisabledTracer(t *testing.T) {
	var called bool
	tr := &Tracer{enabled: false, exportFunc: func(s *Span) { called = true }}

	req, _ := http.NewRequest("GET", "/", nil)
	_, root := tr.StartRequest(context.Background(), req)
	tr.FinishRequest(root, 200)

	if called {
		t.Error("export should not be called when disabled")
	}
}
