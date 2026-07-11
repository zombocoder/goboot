package otel

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// recorder builds a Tracer backed by an in-memory span exporter for assertions.
func recorder(t *testing.T) (*Tracer, *tracetest.InMemoryExporter) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	return NewTracer(tp.Tracer("test")), exp
}

func TestSpanRecordedOnSuccess(t *testing.T) {
	tr, exp := recorder(t)
	_, span := tr.Begin(context.Background(), "Svc.Do")
	span.End(nil)

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "Svc.Do" {
		t.Errorf("span name = %q, want Svc.Do", spans[0].Name)
	}
	if spans[0].Status.Code == codes.Error {
		t.Errorf("successful call should not have Error status")
	}
}

func TestSpanRecordsError(t *testing.T) {
	tr, exp := recorder(t)
	_, span := tr.Begin(context.Background(), "Svc.Fail")
	span.End(errors.New("boom"))

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Status.Code != codes.Error {
		t.Errorf("failed call span status = %v, want Error", spans[0].Status.Code)
	}
	// The error is recorded as an exception event.
	foundException := false
	for _, e := range spans[0].Events {
		if e.Name == "exception" {
			foundException = true
		}
	}
	if !foundException {
		t.Errorf("expected an exception event on the span, got events %+v", spans[0].Events)
	}
}

func TestBeginReturnsSpanContext(t *testing.T) {
	tr, _ := recorder(t)
	base := context.Background()
	ctx, span := tr.Begin(base, "Svc.Do")
	defer span.End(nil)
	// The returned context carries the active span, so nested work nests under it.
	if ctx == base {
		t.Error("Begin should return a span-scoped context, not the original")
	}
}
