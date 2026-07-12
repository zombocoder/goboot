package compiler

import (
	"fmt"
	"go/token"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/zombocoder/goboot/annotation"
)

// Compiler diagnostic codes. Loading failures use the GOBLOAD family; they sit
// alongside the annotation package's GOBANN family (§39.4).
const (
	// CodeLoadError is a package that failed to load or type-check.
	CodeLoadError = "GOBLOAD001"
	// CodeNoPackages is a pattern set that matched no packages.
	CodeNoPackages = "GOBLOAD002"
)

// loadMode is the set of package fields the compiler requires. It mirrors the
// mode described in specification §37.2: names, files, imports, dependencies,
// syntax trees, and full type information.
const loadMode = packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedModule

// Loader loads and scans Go packages. The zero value is usable; Registry
// defaults to annotation.DefaultRegistry() when nil.
type Loader struct {
	// Registry validates discovered annotations. Nil means the default v0.1
	// core catalogue.
	Registry *annotation.Registry
	// Dir is the directory in which to resolve patterns and modules. Empty
	// means the current working directory.
	Dir string
	// BuildFlags are passed to the underlying go tool, e.g. -tags. They allow
	// build tags and platform constraints to be respected (§37.2).
	BuildFlags []string
	// Env overrides the process environment for the go tool when non-nil.
	Env []string
	// Tests includes test files and synthesized test variants in the load.
	Tests bool
	// IgnorePkgPath, when set, suppresses load errors that arise solely because
	// this package is absent or empty — namely its own "invalid package name"/
	// "no Go files" error and any in-scope package's failure to import it. The
	// generate command sets it to the output package's import path so that a
	// composition root importing the not-yet-generated wiring does not block the
	// very generation that would create it (§40).
	IgnorePkgPath string
}

// registry returns the effective registry.
func (l *Loader) registry() *annotation.Registry {
	if l.Registry != nil {
		return l.Registry
	}
	return annotation.DefaultRegistry()
}

// Load loads the packages matching patterns, scans them for annotations, and
// returns the result. A returned error indicates a failure to invoke the go
// tool at all; per-package problems (type errors, unmatched patterns) are
// reported as diagnostics in the result so that partial analysis can proceed.
func (l *Loader) Load(patterns ...string) (*ScanResult, error) {
	cfg := &packages.Config{
		Mode:       loadMode,
		Dir:        l.Dir,
		BuildFlags: l.BuildFlags,
		Env:        l.Env,
		Tests:      l.Tests,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("compiler: loading packages: %w", err)
	}

	result := &ScanResult{}
	if len(pkgs) == 0 {
		result.Diagnostics = append(result.Diagnostics, &annotation.Diagnostic{
			Severity: annotation.SeverityError,
			Code:     CodeNoPackages,
			Message:  fmt.Sprintf("no packages matched %v", patterns),
		})
		return result, nil
	}

	// Load packages in a deterministic order so output does not depend on the
	// go tool's iteration order (§6.7).
	sort.Slice(pkgs, func(i, j int) bool { return pkgs[i].PkgPath < pkgs[j].PkgPath })

	scanner := newScanner(l.registry())
	for _, pkg := range pkgs {
		result.Diagnostics = append(result.Diagnostics, loadDiagnostics(pkg, l.IgnorePkgPath)...)
		// Skip scanning packages that did not type-check, since declaration
		// association relies on complete type information.
		if pkg.Types == nil || pkg.TypesInfo == nil || len(pkg.Syntax) == 0 {
			continue
		}
		scanned := scanner.scanPackage(pkg)
		result.Packages = append(result.Packages, scanned)
		result.Declarations = append(result.Declarations, scanned.Declarations...)
		result.Diagnostics = append(result.Diagnostics, scanner.takeDiagnostics()...)
	}
	return result, nil
}

// loadDiagnostics converts go/packages load and type-check errors into
// compiler diagnostics, dropping those caused only by an absent/empty ignore
// package (the not-yet-generated output; see Loader.IgnorePkgPath).
func loadDiagnostics(pkg *packages.Package, ignore string) []*annotation.Diagnostic {
	var diags []*annotation.Diagnostic
	for _, e := range pkg.Errors {
		if isNotYetGeneratedError(pkg.PkgPath, e.Msg, ignore) {
			continue
		}
		diags = append(diags, &annotation.Diagnostic{
			Severity: annotation.SeverityError,
			Code:     CodeLoadError,
			Message:  fmt.Sprintf("%s: %s", pkg.PkgPath, e.Msg),
			Position: parsePackagesPos(e.Pos),
		})
	}
	return diags
}

// isNotYetGeneratedError reports whether a load error is only an artifact of the
// output package not existing yet: either the output package's own empty-dir
// error, or an in-scope package failing to import it. Matching the import path
// plus the characteristic phrasing keeps genuine errors that merely mention the
// path from being swallowed.
func isNotYetGeneratedError(pkgPath, msg, ignore string) bool {
	if ignore == "" {
		return false
	}
	if pkgPath == ignore {
		return true // the output package itself: empty dir → invalid package name
	}
	return strings.Contains(msg, ignore) &&
		(strings.Contains(msg, "could not import") ||
			strings.Contains(msg, "invalid package name") ||
			strings.Contains(msg, "no Go files"))
}

// parsePackagesPos parses the "file:line:col" position string that
// packages.Error carries into a token.Position. An unparseable or empty string
// yields the zero position.
func parsePackagesPos(s string) token.Position {
	if s == "" {
		return token.Position{}
	}
	var pos token.Position
	// packages.Error.Pos is "file:line:col"; parse from the right so that
	// Windows drive-letter colons in the filename are preserved.
	if line, col, file, ok := splitPos(s); ok {
		pos.Filename = file
		pos.Line = line
		pos.Column = col
	} else {
		pos.Filename = s
	}
	return pos
}
