// Package metrics is a goboot plugin that turns @Counter / @Gauge annotations
// into registered Prometheus collectors (github.com/zombocoder/goboot issue #35).
// It showcases a plugin driving code generation from its own annotations: it
// registers the annotation schemas (AnnotationProvider), validates them
// (Analyzer), and emits a metrics file (Generator) with the collectors, a
// RegisterMetrics function, and a typed accessor per metric:
//
//	// @Counter(name="orders_processed_total", help="Orders processed", labels=["status"])
//	func (s *OrderService) Process(...) { ... s.metrics... }
//
// generates a registered *prometheus.CounterVec and an accessor
// OrdersProcessedTotal() the application increments. (It complements @Timed,
// which the core proxy instruments automatically; @Counter/@Gauge are for custom
// business metrics.)
//
// Register it in goboot.yaml:
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/metrics
//
// and register the collectors in the composition root:
//
//	generated.RegisterMetrics(reg)
//
// The generated file imports github.com/prometheus/client_golang/prometheus, so
// the target module must require it.
package metrics

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Annotation names this plugin owns.
const (
	annCounter = "Counter"
	annGauge   = "Gauge"
)

// outputFile is the generated file's name; the zz_goboot_ prefix lets
// `goboot clean` remove it (§40).
const outputFile = "zz_goboot_metrics.gen.go"

// Plugin implements AnnotationProvider, Analyzer, and Generator.
type Plugin struct{}

// New constructs the metrics plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "metrics" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.0" }

// Annotations registers @Counter and @Gauge so the compiler recognizes them.
func (*Plugin) Annotations() []*annotation.Definition {
	targets := []annotation.Target{annotation.TargetMethod, annotation.TargetType}
	args := map[string]annotation.ArgumentDefinition{
		"name":      {Type: annotation.ArgString, Required: true},
		"help":      {Type: annotation.ArgString},
		"namespace": {Type: annotation.ArgString},
		"labels":    {Type: annotation.ArgStringArray},
	}
	return []*annotation.Definition{
		{Name: annCounter, Targets: targets, Arguments: args},
		{Name: annGauge, Targets: targets, Arguments: args},
	}
}

// hasOurAnnotation reports whether a declaration carries @Counter or @Gauge.
func hasOurAnnotation(d model.AnnotatedDecl) bool {
	return d.Has(annCounter) || d.Has(annGauge)
}
