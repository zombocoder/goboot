package cli

import (
	"flag"
	"fmt"
	"io"

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

// cmdPlugins reports the plugins configured in goboot.yaml alongside those
// actually linked into this binary, so a developer can see whether a
// plugin-aware build is required (§46.2).
func cmdPlugins(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("plugins", flag.ContinueOnError)
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
