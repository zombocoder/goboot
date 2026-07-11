package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigPlugins(t *testing.T) {
	dir := t.TempDir()
	yaml := `
application:
  name: demo
plugins:
  - github.com/acme/plugin-pgx@v0.2.0
  - module: github.com/acme/goboot-redis
    version: v1.3.1
    import: github.com/acme/goboot-redis/gobootx
    new: NewRedis
`
	if err := os.WriteFile(filepath.Join(dir, "goboot.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Plugins) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(cfg.Plugins))
	}

	// Shorthand form splits module@version and applies conventional defaults.
	short := cfg.Plugins[0].normalize()
	if short.Module != "github.com/acme/plugin-pgx" || short.Version != "v0.2.0" {
		t.Errorf("shorthand parse = %+v", short)
	}
	if short.Import != short.Module || short.New != "New" {
		t.Errorf("shorthand defaults = %+v, want import=module new=New", short)
	}

	// Explicit form keeps its import path and constructor.
	full := cfg.Plugins[1].normalize()
	if full.Import != "github.com/acme/goboot-redis/gobootx" || full.New != "NewRedis" {
		t.Errorf("explicit parse = %+v", full)
	}
}

func TestLoadConfigNoPlugins(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "goboot.yaml"), []byte("application:\n  name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if len(cfg.Plugins) != 0 {
		t.Errorf("expected no plugins, got %d", len(cfg.Plugins))
	}
}
