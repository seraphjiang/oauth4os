package tracing

import (
	"context"
	"testing"
)

func TestNoopTracer(t *testing.T) {
	tr := NoopTracer{}
	ctx, span := tr.StartSpan(context.Background(), "test", nil)
	if span == nil {
		t.Fatal("span should not be nil")
	}
	tr.EndSpan(span, "ok")
	if FromContext(ctx) == nil {
		t.Error("span should be in context")
	}
}
