package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zombocoder/goboot/generator/di"
)

// generatedFilePrefix marks files goboot produces (§40); clean removes files
// with this prefix and the generated marker.
const generatedFilePrefix = "zz_goboot_"

// generatedFileName is the wiring file goboot writes.
const generatedFileName = generatedFilePrefix + "wiring.gen.go"

func cmdGenerate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		dir     = fs.String("dir", ".", "working directory containing go.mod and goboot.yaml")
		output  = fs.String("output", "", "output directory (overrides goboot.yaml)")
		pkg     = fs.String("package", "", "generated package name (overrides goboot.yaml)")
		tags    = fs.String("tags", "", "comma-separated build tags")
		strict  = fs.Bool("strict", false, "treat warnings as errors")
		clean   = fs.Bool("clean", false, "remove existing generated files first")
		dialect = fs.String("dialect", "", "SQL dialect for repositories: postgres (default) or question")
		verbose = fs.Bool("verbose", false, "print progress")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	patterns := resolvePatterns(fs.Args(), cfg)
	outputDir := firstNonEmpty(*output, cfg.Generation.Output)
	pkgName := firstNonEmpty(*pkg, cfg.Generation.Package)
	strictMode := *strict || cfg.Generation.Strict
	cleanFirst := *clean || cfg.Generation.Clean

	res, host, errCount := analyzeCommon(*dir, patterns, *tags, strictMode, stderr)
	if res == nil {
		return 1
	}
	if errCount > 0 {
		fmt.Fprintf(stderr, "goboot: %d error(s); no files written\n", errCount)
		return 1
	}

	dialectName := firstNonEmpty(*dialect, cfg.Generation.Dialect)
	sqlDialect, ok := host.Dialect(dialectName)
	if !ok {
		fmt.Fprintf(stderr, "goboot: unknown SQL dialect %q\n", dialectName)
		return 2
	}

	src, err := di.Generate(res.App, res.Graph, di.Options{Package: pkgName, Dialect: sqlDialect})
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}

	// Plugin generators contribute additional artifacts.
	pluginFiles, gdiags := host.Generate(res.App)
	if n := printDiagnostics(stderr, gdiags, strictMode); n > 0 {
		fmt.Fprintf(stderr, "goboot: %d plugin generation error(s); no files written\n", n)
		return 1
	}

	absOut := filepath.Join(*dir, outputDir)
	if cleanFirst {
		if err := cleanDir(absOut); err != nil {
			fmt.Fprintf(stderr, "goboot: %v\n", err)
			return 1
		}
	}
	if err := os.MkdirAll(absOut, 0o755); err != nil {
		fmt.Fprintf(stderr, "goboot: creating output directory: %v\n", err)
		return 1
	}
	target := filepath.Join(absOut, generatedFileName)
	if err := writeFileAtomic(target, []byte(src)); err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	for _, f := range pluginFiles {
		path := filepath.Join(absOut, filepath.Base(f.Name))
		if err := writeFileAtomic(path, f.Content); err != nil {
			fmt.Fprintf(stderr, "goboot: %v\n", err)
			return 1
		}
		if *verbose {
			fmt.Fprintf(stdout, "goboot: wrote %s (plugin)\n", path)
		}
	}

	if *verbose {
		fmt.Fprintf(stdout, "goboot: analyzed %d component(s), %d route(s)\n",
			len(res.App.Components), len(res.App.Routes))
	}
	fmt.Fprintf(stdout, "goboot: wrote %s\n", target)
	return 0
}

// writeFileAtomic writes data to a temp file in the same directory and renames
// it into place, so a failed write never leaves partial output (§37.10).
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".goboot-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming into place: %w", err)
	}
	return nil
}

// resolvePatterns returns the package patterns to analyze: command-line
// arguments take precedence, then goboot.yaml, then "./...".
func resolvePatterns(args []string, cfg projectConfig) []string {
	if len(args) > 0 {
		return args
	}
	if len(cfg.Application.Packages) > 0 {
		return cfg.Application.Packages
	}
	return []string{"./..."}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
