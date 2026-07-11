package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func TestDiscoverConfigProperties(t *testing.T) {
	res := analyzeApp(t, "./testdata/cfgapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	props := componentByName(res.App, "ServerProperties")
	if props == nil {
		t.Fatal("ServerProperties config component not discovered")
	}
	if props.Kind != model.ComponentConfigProperties {
		t.Errorf("kind = %v, want config-properties", props.Kind)
	}
	if props.ConfigPrefix != "server" {
		t.Errorf("prefix = %q, want server", props.ConfigPrefix)
	}
	if props.Constructor == nil || !props.Constructor.ConfigLoader {
		t.Errorf("config component should have a config-loader constructor")
	}
	if props.Constructor.FuncName != "LoadServerProperties" {
		t.Errorf("loader name = %q", props.Constructor.FuncName)
	}
}

func TestConfigPropertiesInjected(t *testing.T) {
	res := analyzeApp(t, "./testdata/cfgapp")
	engine := componentByName(res.App, "engine")
	if engine == nil {
		t.Fatal("engine not found")
	}
	props := componentByName(res.App, "ServerProperties")
	if len(engine.Dependencies) != 1 || engine.Dependencies[0].ResolvedTo != props.ID {
		t.Errorf("engine should depend on the config properties; deps=%v", engine.DependsOn())
	}
}

func TestDiscoverLifecycleHooks(t *testing.T) {
	res := analyzeApp(t, "./testdata/cfgapp")
	engine := componentByName(res.App, "engine")
	if engine == nil {
		t.Fatal("engine not found")
	}
	if engine.PostConstruct == nil {
		t.Fatal("engine should have a @PostConstruct hook")
	}
	if engine.PostConstruct.MethodName != "Start" || !engine.PostConstruct.TakesContext || !engine.PostConstruct.ReturnsError {
		t.Errorf("PostConstruct = %+v", engine.PostConstruct)
	}
	if engine.PreDestroy == nil {
		t.Fatal("engine should have a @PreDestroy hook")
	}
	if engine.PreDestroy.MethodName != "Stop" || engine.PreDestroy.TakesContext || !engine.PreDestroy.ReturnsError {
		t.Errorf("PreDestroy = %+v", engine.PreDestroy)
	}
}

func TestInvalidLifecycleSignature(t *testing.T) {
	res := analyzeApp(t, "./testdata/badlifecycle")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidLifecycle {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid-lifecycle diagnostic, got %v", res.Diagnostics)
	}
}
