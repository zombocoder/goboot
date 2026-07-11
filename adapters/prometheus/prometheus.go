// Package prometheus adapts Prometheus to goboot's runtime.MethodMetrics seam,
// so @Timed service methods increment a counter labeled by method and outcome.
// It lives in its own module to keep the Prometheus dependency out of the goboot
// core. Wire it into the proxy dependencies and expose the registry:
//
//	reg := prometheus.NewRegistry()
//	proxyDeps := runtime.DefaultProxyDependencies()
//	proxyDeps.Metrics = gobootprom.NewMetrics(reg)
//	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// Metrics implements runtime.MethodMetrics with a Prometheus counter vector.
type Metrics struct {
	calls *prometheus.CounterVec
}

// NewMetrics builds the metrics and registers the counter with reg (typically a
// *prometheus.Registry or prometheus.DefaultRegisterer). The exported series is
// goboot_method_calls_total{method,outcome}.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	calls := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "goboot",
			Name:      "method_calls_total",
			Help:      "Total intercepted service-method calls, by method and outcome.",
		},
		[]string{"method", "outcome"},
	)
	reg.MustRegister(calls)
	return &Metrics{calls: calls}
}

// RecordSuccess counts a successful call.
func (m *Metrics) RecordSuccess(method string) {
	m.calls.WithLabelValues(method, "success").Inc()
}

// RecordFailure counts a failed call.
func (m *Metrics) RecordFailure(method string) {
	m.calls.WithLabelValues(method, "failure").Inc()
}

// Compile-time assertion of the implemented contract.
var _ goruntime.MethodMetrics = (*Metrics)(nil)
