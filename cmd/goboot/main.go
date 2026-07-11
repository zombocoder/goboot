// Command goboot is the annotation-driven compiler CLI (§43). It loads Go
// packages, parses annotations, validates the application, and generates
// type-safe wiring — all through the standard go/generate workflow (§44, §59).
package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/plugin"
)

// Version identifiers surfaced by `goboot version` (§47).
const (
	// CLIVersion is the compiler/CLI version.
	CLIVersion = "0.1.0"
	// RequiredRuntimeVersion is the runtime compatibility version generated code
	// depends on.
	RequiredRuntimeVersion = "0.1"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
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
		{"version", "print version information", cmdVersion},
	}
}

// run dispatches to a subcommand and returns a process exit code. It is the
// testable entry point.
func run(args []string, stdout, stderr io.Writer) int {
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
func analyzeCommon(dir string, patterns []string, tags string, strict bool, opts compiler.Options, stderr io.Writer) (*compiler.AnalysisResult, *plugin.Registry, int) {
	host := pluginHost()
	registry, regDiags := host.AnnotationRegistry()

	loader := &compiler.Loader{Dir: dir, Registry: registry}
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
	for _, p := range splitCSV(profile) {
		opts.Profiles = append(opts.Profiles, p)
	}
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
