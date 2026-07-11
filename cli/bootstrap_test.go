package cli

import (
	"go/parser"
	"go/token"
	"io"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/plugin/exampleplugin"
)

func TestToolMainRenders(t *testing.T) {
	refs := []pluginRef{
		{Module: "github.com/acme/plugin-pgx", Import: "github.com/acme/plugin-pgx", New: "New"},
		{Module: "github.com/acme/goboot-redis", Import: "github.com/acme/goboot-redis/gobootx", New: "NewRedis"},
	}
	src, err := toolMain(refs)
	if err != nil {
		t.Fatalf("toolMain: %v", err)
	}
	// It must be valid, gofmt-stable Go.
	if _, err := parser.ParseFile(token.NewFileSet(), "main.go", src, parser.AllErrors); err != nil {
		t.Fatalf("generated tool main does not parse: %v\n%s", err, src)
	}
	for _, want := range []string{
		"package main",
		`"github.com/zombocoder/goboot/cli"`,
		`p0 "github.com/acme/plugin-pgx"`,
		`p1 "github.com/acme/goboot-redis/gobootx"`,
		"os.Exit(cli.Main(",
		"p0.New(),",
		"p1.NewRedis(),",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("tool main missing %q\n%s", want, src)
		}
	}
}

func TestBootstrapKeyStableAndSensitive(t *testing.T) {
	a := []pluginRef{{Module: "m", Version: "v1", Import: "m", New: "New"}}
	b := []pluginRef{{Module: "m", Version: "v2", Import: "m", New: "New"}}

	if bootstrapKey(a) != bootstrapKey(a) {
		t.Error("bootstrap key should be stable for the same plugin set")
	}
	if bootstrapKey(a) == bootstrapKey(b) {
		t.Error("bootstrap key should change when a plugin version changes")
	}
	if len(bootstrapKey(a)) != 12 {
		t.Errorf("bootstrap key length = %d, want 12", len(bootstrapKey(a)))
	}
}

func TestMaybeBootstrapGating(t *testing.T) {
	withPlugins := func(p []pluginRef) projectConfig {
		var c projectConfig
		c.Plugins = p
		return c
	}
	one := []pluginRef{{Module: "m", Import: "m", New: "New"}}

	// No plugins configured → never bootstraps.
	if _, done := maybeBootstrap(t.TempDir(), projectConfig{}, io.Discard, io.Discard); done {
		t.Error("no plugins should not bootstrap")
	}

	// Already plugin-linked (hostPlugins set) → skip, even with plugins configured.
	prev := hostPlugins
	hostPlugins = []plugin.Plugin{exampleplugin.New()}
	if _, done := maybeBootstrap(t.TempDir(), withPlugins(one), io.Discard, io.Discard); done {
		t.Error("a plugin-linked binary should not re-bootstrap")
	}
	hostPlugins = prev

	// Explicitly disabled → skip.
	t.Setenv("GOBOOT_BOOTSTRAP", "off")
	if _, done := maybeBootstrap(t.TempDir(), withPlugins(one), io.Discard, io.Discard); done {
		t.Error("GOBOOT_BOOTSTRAP=off should skip bootstrap")
	}
}

func TestNormalizedPluginRefsSortedWithDefaults(t *testing.T) {
	var cfg projectConfig
	cfg.Plugins = []pluginRef{
		{Module: "github.com/z/last"},
		{Module: "github.com/a/first", Import: "github.com/a/first/pkg", New: "Make"},
	}
	refs := normalizedPluginRefs(cfg)
	if refs[0].Import != "github.com/a/first/pkg" {
		t.Errorf("refs not sorted by import: %+v", refs)
	}
	// Defaults applied to the entry that omitted them.
	if refs[1].Import != "github.com/z/last" || refs[1].New != "New" {
		t.Errorf("defaults not applied: %+v", refs[1])
	}
}
