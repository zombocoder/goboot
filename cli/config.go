package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// projectConfig is the subset of goboot.yaml the CLI consumes (§45). Absent
// fields fall back to defaults.
type projectConfig struct {
	Application struct {
		Name     string   `yaml:"name"`
		Packages []string `yaml:"packages"`
	} `yaml:"application"`
	Generation struct {
		Output  string `yaml:"output"`
		Package string `yaml:"package"`
		Clean   bool   `yaml:"clean"`
		Strict  bool   `yaml:"strict"`
		Dialect string `yaml:"dialect"`
	} `yaml:"generation"`
	// Plugins lists the compile-time plugins to link into the CLI (§46.2). The
	// self-bootstrap flow builds a plugin-aware binary from this list.
	Plugins []pluginRef `yaml:"plugins"`
}

// pluginRef references a plugin Go module. It accepts either a shorthand scalar
// ("module@version", using conventions) or an explicit mapping.
type pluginRef struct {
	// Module is the plugin's Go module path (e.g. github.com/acme/goboot-redis).
	Module string `yaml:"module"`
	// Version is the module version to pin (e.g. v1.2.0); empty means latest.
	Version string `yaml:"version"`
	// Import is the package path holding the constructor; defaults to Module.
	Import string `yaml:"import"`
	// New is the constructor function name; defaults to "New".
	New string `yaml:"new"`
}

// UnmarshalYAML accepts a scalar "module@version" shorthand as well as the full
// mapping form, so goboot.yaml can list plugins concisely.
func (p *pluginRef) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		module, version, _ := strings.Cut(node.Value, "@")
		p.Module = strings.TrimSpace(module)
		p.Version = strings.TrimSpace(version)
		return nil
	}
	type raw pluginRef // avoid recursing into this method
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*p = pluginRef(r)
	return nil
}

// normalize fills in the conventional defaults: Import defaults to the module
// root, and New defaults to "New".
func (p pluginRef) normalize() pluginRef {
	if p.Import == "" {
		p.Import = p.Module
	}
	if p.New == "" {
		p.New = "New"
	}
	return p
}

// defaultConfig returns the configuration used when goboot.yaml is absent or a
// field is unset.
func defaultConfig() projectConfig {
	var c projectConfig
	c.Generation.Output = "internal/generated"
	c.Generation.Package = "generated"
	return c
}

// configFileName is the project configuration file (§45).
const configFileName = "goboot.yaml"

// loadConfig reads goboot.yaml from dir, applying defaults for unset fields. A
// missing file is not an error; it yields the defaults.
func loadConfig(dir string) (projectConfig, error) {
	cfg := defaultConfig()
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading %s: %w", configFileName, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", configFileName, err)
	}
	// Re-apply defaults for anything the file left empty.
	if cfg.Generation.Output == "" {
		cfg.Generation.Output = "internal/generated"
	}
	if cfg.Generation.Package == "" {
		cfg.Generation.Package = "generated"
	}
	return cfg, nil
}
