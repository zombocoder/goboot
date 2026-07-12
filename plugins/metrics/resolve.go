package metrics

import (
	"go/token"
	"regexp"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Diagnostic codes this plugin emits (§39.4, plugin-owned family).
const (
	codeInvalidName  = "GOBMET001" // invalid Prometheus metric name
	codeDuplicate    = "GOBMET002" // two metrics share a name
	codeInvalidLabel = "GOBMET003" // invalid Prometheus label name
)

// metricKind is a counter or a gauge.
type metricKind int

const (
	counterKind metricKind = iota
	gaugeKind
)

// metric is a resolved @Counter/@Gauge declaration.
type metric struct {
	kind      metricKind
	name      string // metric name (without namespace)
	fullName  string // namespace_name, the exported series name
	help      string
	namespace string
	labels    []string // non-empty → a *Vec collector
	accessor  string   // exported accessor func name
	varName   string   // unexported package var holding the collector
	pos       token.Position
}

var (
	promName  = regexp.MustCompile(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`)
	promLabel = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// resolve reads every @Counter/@Gauge into the metric model, sorted by name, and
// returns validation diagnostics (bad names, duplicates, bad labels).
func resolve(app *model.Application) ([]metric, []*annotation.Diagnostic) {
	var metrics []metric
	var diags []*annotation.Diagnostic
	seen := map[string]bool{}

	for _, d := range app.Declarations {
		if !hasOurAnnotation(d) {
			continue
		}
		for _, a := range d.Annotations {
			var k metricKind
			switch a.Name {
			case annCounter:
				k = counterKind
			case annGauge:
				k = gaugeKind
			default:
				continue
			}
			m, ds := metricFrom(k, a, seen)
			diags = append(diags, ds...)
			if m != nil {
				metrics = append(metrics, *m)
			}
		}
	}
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].fullName < metrics[j].fullName })
	return metrics, diags
}

// metricFrom validates one annotation and builds its metric, or returns
// diagnostics and nil.
func metricFrom(k metricKind, a annotation.Annotation, seen map[string]bool) (*metric, []*annotation.Diagnostic) {
	name := stringArg(a, "name")
	if name == "" {
		return nil, nil // required by the schema; a missing name is reported there
	}
	ns := stringArg(a, "namespace")
	full := name
	if ns != "" {
		full = ns + "_" + name
	}
	var diags []*annotation.Diagnostic
	if !promName.MatchString(full) {
		return nil, append(diags, diag(codeInvalidName, a.Position, "invalid Prometheus metric name %q", full))
	}
	if seen[full] {
		return nil, append(diags, diag(codeDuplicate, a.Position, "duplicate metric name %q", full))
	}
	seen[full] = true

	labels := stringArrayArg(a, "labels")
	for _, l := range labels {
		if !promLabel.MatchString(l) {
			diags = append(diags, diag(codeInvalidLabel, a.Position, "invalid Prometheus label name %q", l))
		}
	}
	return &metric{
		kind:      k,
		name:      name,
		fullName:  full,
		help:      stringArg(a, "help"),
		namespace: ns,
		labels:    labels,
		accessor:  exportedName(full),
		varName:   "metric" + exportedName(full),
		pos:       a.Position,
	}, diags
}

// exportedName turns a metric name into an exported Go identifier, e.g.
// "orders_processed_total" → "OrdersProcessedTotal".
func exportedName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	})
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(p[1:])
	}
	if b.Len() == 0 {
		return "Metric"
	}
	return b.String()
}
