// Package otel adapts OpenTelemetry tracing to goboot's runtime.Tracer seam, so
// @Traced service methods emit real spans. It lives in its own module to keep
// the OpenTelemetry dependency out of the goboot core. Wire it into the proxy
// dependencies:
//
//	import "go.opentelemetry.io/otel"
//	proxyDeps := runtime.DefaultProxyDependencies()
//	proxyDeps.Tracer = goboototel.NewTracer(otel.Tracer("goboot"))
//
// where a global TracerProvider has already been configured by the application.
package otel

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// Tracer implements runtime.Tracer over an OpenTelemetry tracer.
type Tracer struct{ tracer oteltrace.Tracer }

// NewTracer builds a goboot Tracer from an OpenTelemetry tracer (e.g.
// otel.Tracer("goboot") or provider.Tracer(name)).
func NewTracer(tracer oteltrace.Tracer) *Tracer { return &Tracer{tracer: tracer} }

// Begin starts a span for the intercepted method and returns the span-scoped
// context so nested work is attributed to it.
func (t *Tracer) Begin(ctx context.Context, name string) (context.Context, goruntime.Span) {
	ctx, s := t.tracer.Start(ctx, name)
	return ctx, &span{s: s}
}

// span adapts an OpenTelemetry span to runtime.Span, recording the method's
// error (if any) on End.
type span struct{ s oteltrace.Span }

// End records the error, sets the span status, and finishes the span.
func (w *span) End(err error) {
	if err != nil {
		w.s.RecordError(err)
		w.s.SetStatus(codes.Error, err.Error())
	}
	w.s.End()
}

// Compile-time assertions of the implemented contracts.
var (
	_ goruntime.Tracer = (*Tracer)(nil)
	_ goruntime.Span   = (*span)(nil)
)
