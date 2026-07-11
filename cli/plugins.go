package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/zombocoder/goboot/plugin"
)

// hostPlugins holds the plugins injected into this CLI process. The default
// binary at cmd/goboot injects none; a plugin-aware build passes them through
// Main, and the self-bootstrap flow generates such a build from goboot.yaml
// (§46.2). It is a package variable so tests can inject plugins directly.
var hostPlugins []plugin.Plugin

// pluginHost builds the plugin registry for a CLI invocation.
func pluginHost() *plugin.Registry {
	return plugin.New(hostPlugins...)
}

// cmdPlugins dispatches the plugin subcommands: `list` (the default) reports
// configured vs. linked plugins; `sync` writes a committed tool main and pins
// the plugin modules for reproducible / CI builds (§46.2).
func cmdPlugins(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "list":
			return pluginsList(args[1:], stdout, stderr)
		case "sync":
			return pluginsSync(args[1:], stdout, stderr)
		}
	}
	return pluginsList(args, stdout, stderr)
}

// pluginsList reports the plugins configured in goboot.yaml alongside those
// actually linked into this binary, so a developer can see whether a
// plugin-aware build is required (§46.2).
func pluginsList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plugins list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "working directory containing goboot.yaml")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}

	// Linked plugins, indexed for a quick "configured but not linked" check.
	linked := map[string]string{}
	for _, p := range pluginHost().Plugins() {
		linked[p.Name()] = p.Version()
	}

	fmt.Fprintf(stdout, "plugin API version: %s\n\n", plugin.APIVersion)

	fmt.Fprintln(stdout, "configured (goboot.yaml):")
	if len(cfg.Plugins) == 0 {
		fmt.Fprintln(stdout, "\t(none)")
	}
	for _, ref := range cfg.Plugins {
		ref = ref.normalize()
		version := ref.Version
		if version == "" {
			version = "latest"
		}
		fmt.Fprintf(stdout, "\t%s@%s (import %s, %s())\n", ref.Module, version, ref.Import, ref.New)
	}

	fmt.Fprintln(stdout, "\nlinked into this binary:")
	if len(linked) == 0 {
		fmt.Fprintln(stdout, "\t(none)")
	}
	for name, version := range linked {
		fmt.Fprintf(stdout, "\t%s %s\n", name, version)
	}

	if len(cfg.Plugins) > 0 && len(linked) == 0 {
		fmt.Fprintln(stdout, "\nnote: plugins are configured but none are linked; run through a")
		fmt.Fprintln(stdout, "plugin-aware build (goboot generate self-bootstraps one from goboot.yaml).")
	}
	return 0
}

// pluginsSync pins the configured plugin modules and writes a committed tool
// main at tools/goboot/main.go, giving a reproducible, network-free build for CI
// (drive it with `go run ./tools/goboot generate ./...`). The self-bootstrap
// covers the interactive path; sync is the explicit alternative (§46.2).
func pluginsSync(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plugins sync", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "working directory containing go.mod and goboot.yaml")
	out := fs.String("out", filepath.Join("tools", "goboot"), "directory for the generated tool main")
	skipGet := fs.Bool("no-get", false, "do not run `go get` to pin plugin modules")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	refs := normalizedPluginRefs(cfg)
	if len(refs) == 0 {
		fmt.Fprintln(stderr, "goboot: no plugins configured in goboot.yaml")
		return 1
	}

	if !*skipGet {
		goGetPlugins(*dir, refs, stderr)
	}

	src, err := toolMain(refs)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	toolDir := filepath.Join(*dir, *out)
	if err := os.MkdirAll(toolDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	mainPath := filepath.Join(toolDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(src), 0o644); err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "goboot: wrote %s (%d plugin(s))\n", filepath.Join(*out, "main.go"), len(refs))
	fmt.Fprintf(stdout, "run generation through it: go run ./%s generate ./...\n", filepath.ToSlash(*out))
	return 0
}
