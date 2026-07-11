// Package lint is a goboot plugin that enforces HTTP/REST conventions over the
// analyzed model (§46.1, Analyzer capability). It reports source-positioned
// warnings — advisory unless the build runs with -strict — so it never blocks
// generation on its own.
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/lint@latest
//
// Rules:
//
//	LINT001  duplicate operationId (two routes share a handler name)
//	LINT002  non-lowercase static path segment
//	LINT003  trailing slash in a route pattern
package lint

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/plugin"
)

// New constructs the lint plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Plugin implements the Analyzer capability.
type Plugin struct{}

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "lint" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.0" }

// Analyze reports convention violations across the application's routes. The
// routes are examined in a stable (pattern, method) order so the diagnostics are
// deterministic (§46.4).
func (*Plugin) Analyze(app *model.Application) []*annotation.Diagnostic {
	routes := append([]*model.Route(nil), app.Routes...)
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Pattern != routes[j].Pattern {
			return routes[i].Pattern < routes[j].Pattern
		}
		return routes[i].Method < routes[j].Method
	})

	var diags []*annotation.Diagnostic
	seen := map[string]token.Position{}
	for _, r := range routes {
		if prev, ok := seen[r.HandlerName]; ok {
			diags = append(diags, warn("LINT001", r.Position,
				"duplicate operationId %q (also declared at %s); generated clients require unique operation names",
				r.HandlerName, prev))
		} else {
			seen[r.HandlerName] = r.Position
		}
		if seg, ok := nonLowerSegment(r.Pattern); ok {
			diags = append(diags, warn("LINT002", r.Position,
				"route %s %s has a non-lowercase path segment %q; prefer lowercase, hyphenated paths",
				r.Method, r.Pattern, seg))
		}
		if r.Pattern != "/" && strings.HasSuffix(r.Pattern, "/") {
			diags = append(diags, warn("LINT003", r.Position,
				"route %s %s has a trailing slash; drop it for consistency", r.Method, r.Pattern))
		}
	}
	return diags
}

// nonLowerSegment returns the first static path segment containing an uppercase
// ASCII letter, ignoring empty and {param} segments.
func nonLowerSegment(pattern string) (string, bool) {
	for _, seg := range strings.Split(pattern, "/") {
		if seg == "" || (strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")) {
			continue
		}
		for _, r := range seg {
			if r >= 'A' && r <= 'Z' {
				return seg, true
			}
		}
	}
	return "", false
}

// warn builds a warning-severity diagnostic anchored at pos.
func warn(code string, pos token.Position, format string, args ...any) *annotation.Diagnostic {
	return &annotation.Diagnostic{
		Severity: annotation.SeverityWarning,
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
	}
}

// Compile-time assertions of the implemented capabilities.
var (
	_ plugin.Plugin   = (*Plugin)(nil)
	_ plugin.Analyzer = (*Plugin)(nil)
)
