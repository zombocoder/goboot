// Package oracle is a goboot plugin that contributes an Oracle SQL dialect
// (:1, :2 positional placeholders). It is the reference for shipping a plugin as
// a separate module under plugins/ (§46.2): list it in goboot.yaml and the CLI
// self-bootstraps a plugin-aware build.
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/oracle@latest
//
// then generate a repository with `-dialect oracle`.
package oracle

import (
	"strconv"

	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/sqlgen"
)

// New constructs the Oracle plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Plugin registers the Oracle SQL dialect via the DialectProvider capability.
type Plugin struct{}

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "oracle" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.0" }

// Dialects contributes the "oracle" dialect (§27.4).
func (*Plugin) Dialects() map[string]sqlgen.Dialect {
	return map[string]sqlgen.Dialect{"oracle": Dialect{}}
}

// Dialect renders Oracle positional bind placeholders: :1, :2, ...
type Dialect struct{}

// Name identifies the dialect.
func (Dialect) Name() string { return "oracle" }

// Placeholder renders the 1-based bind placeholder for position i.
func (Dialect) Placeholder(i int) string { return ":" + strconv.Itoa(i) }

// Compile-time assertions that the plugin satisfies the expected capabilities.
var (
	_ plugin.Plugin          = (*Plugin)(nil)
	_ plugin.DialectProvider = (*Plugin)(nil)
	_ sqlgen.Dialect         = Dialect{}
)
