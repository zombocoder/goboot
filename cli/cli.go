// Package cli implements the annotation-driven compiler CLI (§43): it loads Go
// packages, parses annotations, validates the application, and generates
// type-safe wiring through the standard go/generate workflow (§44, §59).
//
// The package is importable so a project can build a plugin-injected CLI: a
// small main calls cli.Main(pluginA.New(), pluginB.New(), ...). The default
// binary at cmd/goboot injects no plugins; the self-bootstrap flow (§46.2)
// generates such a main automatically from goboot.yaml.
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/plugin"
)

// Version identifiers surfaced by `goboot version` (§47).
const (
	// CLIVersion is the compiler/CLI version.
	CLIVersion = "0.1.2"
	// RequiredRuntimeVersion is the runtime compatibility version generated code
	// depends on.
	RequiredRuntimeVersion = "0.1"
)

// Main is the process entry point for a goboot CLI. A plugin-injected build
// passes its plugins here — cli.Main(pluginA.New(), pluginB.New()) — and they
// become active for annotation registration, analysis, generation, and dialect
// resolution. It reads os.Args and returns a process exit code.
func Main(plugins ...plugin.Plugin) int {
	hostPlugins = plugins
	return Run(os.Args[1:], os.Stdout, os.Stderr)
}

// command is a single CLI subcommand.
type command struct {
	name    string
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

// commands lists the subcommands in help order.
func commands() []command {
	return []command{
		{"generate", "generate wiring for the annotated packages", cmdGenerate},
		{"validate", "validate the application without writing files", cmdValidate},
		{"graph", "print the dependency graph", cmdGraph},
		{"clean", "remove goboot-generated files", cmdClean},
		{"doctor", "check the project environment", cmdDoctor},
		{"init", "scaffold a goboot.yaml", cmdInit},
		{"plugins", "list configured and linked plugins", cmdPlugins},
		{"version", "print version information", cmdVersion},
	}
}

// Run dispatches to a subcommand with explicit arguments and output streams. It
// is the testable entry point; Main wraps it with os.Args and os.Stdout/Stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	name := args[0]
	if name == "-h" || name == "--help" || name == "help" {
		usage(stdout)
		return 0
	}
	for _, c := range commands() {
		if c.name == name {
			return c.run(args[1:], stdout, stderr)
		}
	}
	fmt.Fprintf(stderr, "goboot: unknown command %q\n\n", name)
	usage(stderr)
	return 2
}

// usage prints the top-level help.
func usage(w io.Writer) {
	fmt.Fprintln(w, "goboot is the annotation-driven compiler for Go.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "\tgoboot <command> [arguments]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range commands() {
		fmt.Fprintf(w, "\t%-10s %s\n", c.name, c.summary)
	}
}

func cmdVersion(_ []string, stdout, _ io.Writer) int {
	fmt.Fprintf(stdout, "goboot %s\n", CLIVersion)
	fmt.Fprintf(stdout, "runtime compatibility %s\n", RequiredRuntimeVersion)
	plugins := pluginHost().Plugins()
	if len(plugins) == 0 {
		fmt.Fprintln(stdout, "plugins: none")
		return 0
	}
	fmt.Fprintln(stdout, "plugins:")
	for _, p := range plugins {
		fmt.Fprintf(stdout, "\t%s %s\n", p.Name(), p.Version())
	}
	return 0
}

// analyzeCommon loads and analyzes the given patterns, running plugin
// annotations and analyzers through the host, and prints diagnostics to stderr.
// It returns the analysis result, the plugin host, and the number of blocking
// errors (warnings counted as errors when strict).
func analyzeCommon(dir string, patterns []string, tags string, strict bool, opts compiler.Options, ignorePkg string, stderr io.Writer) (*compiler.AnalysisResult, *plugin.Registry, int) {
	host := pluginHost()
	registry, regDiags := host.AnnotationRegistry()

	loader := &compiler.Loader{Dir: dir, Registry: registry, IgnorePkgPath: ignorePkg}
	if tags != "" {
		loader.BuildFlags = []string{"-tags=" + tags}
	}
	scan, err := loader.Load(patterns...)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return nil, host, 1
	}
	res := compiler.AnalyzeWith(scan, opts)

	// Combine core, plugin-registration, and plugin-analyzer diagnostics.
	diags := append([]*annotation.Diagnostic(nil), regDiags...)
	diags = append(diags, res.Diagnostics...)
	diags = append(diags, host.Analyze(res.App)...)

	errCount := printDiagnostics(stderr, diags, strict)
	return res, host, errCount
}

// generatedPackagePath returns the import path of the output directory, so the
// loader can ignore the transient errors caused by that package not existing
// yet (a composition root importing the not-yet-generated wiring). It returns ""
// — meaning "no suppression" — when the path cannot be determined or the output
// lies outside the module.
func generatedPackagePath(dir, outputDir string) string {
	root, modPath := findModule(dir)
	if root == "" || modPath == "" {
		return ""
	}
	absOut, err := filepath.Abs(filepath.Join(dir, outputDir))
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(root, absOut)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	if rel == "." {
		return modPath
	}
	return modPath + "/" + filepath.ToSlash(rel)
}

// findModule walks up from dir to the nearest go.mod, returning its directory
// and declared module path (both empty if none is found).
func findModule(dir string) (root, modPath string) {
	d, err := filepath.Abs(dir)
	if err != nil {
		return "", ""
	}
	for {
		if data, err := os.ReadFile(filepath.Join(d, "go.mod")); err == nil {
			return d, modulePathFromGoMod(data)
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", ""
		}
		d = parent
	}
}

// modulePathFromGoMod extracts the module path from go.mod contents.
func modulePathFromGoMod(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
		}
	}
	return ""
}

// printDiagnostics writes diagnostics in deterministic position order and
// returns the number that are blocking. Under strict mode, warnings are
// promoted to errors (§39.5).
func printDiagnostics(w io.Writer, diags []*annotation.Diagnostic, strict bool) int {
	sorted := append([]*annotation.Diagnostic(nil), diags...)
	sort.SliceStable(sorted, func(i, j int) bool { return lessPosition(sorted[i], sorted[j]) })

	blocking := 0
	for _, d := range sorted {
		severity := d.Severity
		if strict && severity == annotation.SeverityWarning {
			severity = annotation.SeverityError
		}
		if severity == annotation.SeverityError {
			blocking++
		}
		fmt.Fprintf(w, "%s: %s\n", severity, d.Error())
	}
	return blocking
}

// conditionOptions builds analysis options from the comma-separated -profile
// and -property flag values. -property items are key=value pairs.
func conditionOptions(profile, property string) compiler.Options {
	var opts compiler.Options
	opts.Profiles = append(opts.Profiles, splitCSV(profile)...)
	for _, kv := range splitCSV(property) {
		if eq := indexByte(kv, '='); eq >= 0 {
			if opts.Properties == nil {
				opts.Properties = map[string]string{}
			}
			opts.Properties[kv[:eq]] = kv[eq+1:]
		}
	}
	return opts
}

// splitCSV splits a comma-separated list, trimming spaces and dropping empties.
func splitCSV(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// indexByte returns the index of b in s, or -1.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// lessPosition orders diagnostics by file, line, then column.
func lessPosition(a, b *annotation.Diagnostic) bool {
	if a.Position.Filename != b.Position.Filename {
		return a.Position.Filename < b.Position.Filename
	}
	if a.Position.Line != b.Position.Line {
		return a.Position.Line < b.Position.Line
	}
	return a.Position.Column < b.Position.Column
}
