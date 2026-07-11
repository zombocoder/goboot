package main

import (
	"fmt"
	"os"
	"path/filepath"

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
