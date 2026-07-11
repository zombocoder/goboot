// Command goboot is the annotation-driven compiler CLI (§43). It loads Go
// packages, parses annotations, validates the application, and generates
// type-safe wiring — all through the standard go/generate workflow (§44, §59).
package main

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/compiler"
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
	return 0
}

// analyzeCommon loads and analyzes the given patterns, printing diagnostics to
// stderr. It returns the analysis result and the number of blocking errors
// (warnings counted as errors when strict).
func analyzeCommon(dir string, patterns []string, tags string, strict bool, stderr io.Writer) (*compiler.AnalysisResult, int) {
	loader := &compiler.Loader{Dir: dir}
	if tags != "" {
		loader.BuildFlags = []string{"-tags=" + tags}
	}
	scan, err := loader.Load(patterns...)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return nil, 1
	}
	res := compiler.Analyze(scan)
	errCount := printDiagnostics(stderr, res.Diagnostics, strict)
	return res, errCount
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
