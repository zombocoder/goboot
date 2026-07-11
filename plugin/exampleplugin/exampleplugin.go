// Package exampleplugin is a reference goboot plugin demonstrating every
// extension point (§46): it registers an annotation, contributes semantic
// analysis, generates an artifact, and provides a SQL dialect (as a database
// driver would). It doubles as documentation for plugin authors and as the
// end-to-end test subject for the plugin host.
package exampleplugin

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/generator/di"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/sqlgen"
)

// Plugin implements plugin.Plugin plus every optional capability interface.
type Plugin struct{}

// New returns the example plugin.
func New() *Plugin { return &Plugin{} }

func (*Plugin) Name() string    { return "example" }
func (*Plugin) Version() string { return "0.1.0" }

// Annotations registers the @Exposed marker so the compiler recognizes it.
func (*Plugin) Annotations() []*annotation.Definition {
	return []*annotation.Definition{
		{Name: "Exposed", Targets: []annotation.Target{annotation.TargetMethod}},
	}
}

// Analyze reports an informational summary of the application (§46.1). It never
// mutates the model.
func (*Plugin) Analyze(app *model.Application) []*annotation.Diagnostic {
	return []*annotation.Diagnostic{{
		Severity: annotation.SeverityInfo,
		Code:     "EXPL001",
		Message:  fmt.Sprintf("example plugin: application %q has %d component(s)", app.Name, len(app.Components)),
	}}
}

// Generate emits a deterministic component manifest artifact (§46.1). It carries
// the generated-file marker so `goboot clean` removes it.
func (*Plugin) Generate(app *model.Application) ([]plugin.File, error) {
	ids := make([]string, 0, len(app.Components))
	for _, c := range app.Components {
		ids = append(ids, string(c.ID))
	}
	sort.Strings(ids)

	var b strings.Builder
	b.WriteString(di.GeneratedMarker + "\n")
	b.WriteString("# goboot component manifest (example plugin)\n")
	fmt.Fprintf(&b, "application: %s\n", app.Name)
	for _, id := range ids {
		b.WriteString(id + "\n")
	}
	return []plugin.File{{
		Name:    "zz_goboot_manifest.txt",
		Content: []byte(b.String()),
	}}, nil
}

// Dialects registers a SQL Server dialect, illustrating how a database driver
// contributes its placeholder style (§27.4). SQL Server uses @p1, @p2, ...
func (*Plugin) Dialects() map[string]sqlgen.Dialect {
	return map[string]sqlgen.Dialect{"sqlserver": sqlServerDialect{}}
}

// sqlServerDialect renders @p1, @p2, ... placeholders.
type sqlServerDialect struct{}

func (sqlServerDialect) Name() string             { return "sqlserver" }
func (sqlServerDialect) Placeholder(i int) string { return "@p" + strconv.Itoa(i) }
