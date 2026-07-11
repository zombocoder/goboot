package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "working directory")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	type check struct {
		name string
		ok   bool
		note string
	}
	var checks []check

	// Go toolchain.
	checks = append(checks, check{"go version", true, runtime.Version()})

	// Module presence.
	goMod := filepath.Join(*dir, "go.mod")
	_, modErr := os.Stat(goMod)
	checks = append(checks, check{"go.mod", modErr == nil, describeErr(modErr, goMod)})

	// Project config (optional).
	if _, err := os.Stat(filepath.Join(*dir, configFileName)); err == nil {
		checks = append(checks, check{"goboot.yaml", true, "found"})
	} else {
		checks = append(checks, check{"goboot.yaml", true, "not found (using defaults)"})
	}

	// Output directory writability.
	cfg, cfgErr := loadConfig(*dir)
	if cfgErr != nil {
		checks = append(checks, check{"config", false, cfgErr.Error()})
	} else {
		outDir := filepath.Join(*dir, cfg.Generation.Output)
		writable := isWritable(outDir)
		checks = append(checks, check{"output writable", writable, outDir})
	}

	failed := 0
	for _, c := range checks {
		status := "ok"
		if !c.ok {
			status = "FAIL"
			failed++
		}
		fmt.Fprintf(stdout, "[%-4s] %-16s %s\n", status, c.name, c.note)
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "goboot: %d check(s) failed\n", failed)
		return 1
	}
	return 0
}

// isWritable reports whether dir (creating it if needed) can be written to.
func isWritable(dir string) bool {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}
	f, err := os.CreateTemp(dir, ".goboot-doctor-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

func describeErr(err error, path string) string {
	if err != nil {
		return "missing: " + path
	}
	return path
}
