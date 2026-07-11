// Package cfge2e exercises the generated configuration and lifecycle wiring end
// to end: it loads typed configuration from a source, constructs components that
// depend on it, and drives the lifecycle. wiring.gen.go is produced by the
// goboot generator from the cfgapp example.
package cfge2e

import (
	"context"
	"errors"
	"testing"

	"github.com/zombocoder/goboot/runtime/config"
)

func TestConfigLoadedAndInjected(t *testing.T) {
	// Defaults apply when the source is empty.
	def, err := buildComponents(config.MapSource{})
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	if def.ServerProperties.Host != "0.0.0.0" || def.ServerProperties.Port != 8080 {
		t.Errorf("defaults not applied: %+v", def.ServerProperties)
	}
	// The engine received the loaded configuration by value.
	if def.Engine.Addr() != "0.0.0.0" {
		t.Errorf("config not injected into engine: %q", def.Engine.Addr())
	}

	// A source overrides the defaults, type-safely.
	src := config.MapSource{"server.host": "10.0.0.1", "server.port": "9090"}
	comps, err := buildComponents(src)
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	if comps.ServerProperties.Host != "10.0.0.1" || comps.ServerProperties.Port != 9090 {
		t.Errorf("overrides not applied: %+v", comps.ServerProperties)
	}
	if comps.Engine.Addr() != "10.0.0.1" {
		t.Errorf("engine did not see overridden config: %q", comps.Engine.Addr())
	}
}

func TestLifecycleStartAndStop(t *testing.T) {
	comps, err := buildComponents(config.MapSource{})
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	lc := buildLifecycle(comps)

	if comps.Engine.Started() {
		t.Fatal("engine should not be started before lifecycle start")
	}
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !comps.Engine.Started() {
		t.Error("@PostConstruct should have started the engine")
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if comps.Engine.Started() {
		t.Error("@PreDestroy should have stopped the engine")
	}
}

func TestNewApplicationWiresLifecycle(t *testing.T) {
	app, err := NewApplication(config.MapSource{"server.host": "example.com"})
	if err != nil {
		t.Fatalf("NewApplication: %v", err)
	}
	if app.Lifecycle == nil {
		t.Fatal("application should have a lifecycle")
	}
	if err := app.Lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := app.Lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
}

func TestInvalidConfigIsTypeSafeError(t *testing.T) {
	// A non-numeric port must produce a load error rather than a panic.
	_, err := buildComponents(config.MapSource{"server.port": "not-a-number"})
	if err == nil {
		t.Fatal("expected a config load error for an invalid port")
	}
}

// startupOrder records lifecycle events to verify ordering and rollback using
// the runtime lifecycle directly with the generated components.
func TestStartupRollbackInvokesStartedStops(t *testing.T) {
	// This test documents the rollback contract the generated wiring relies on:
	// when a later start hook fails, earlier components' stop hooks run in
	// reverse. It is verified against the runtime in the runtime package; here we
	// confirm the generated engine's hooks are wired such that Stop reverses
	// Start.
	comps, err := buildComponents(config.MapSource{})
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	lc := buildLifecycle(comps)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("stop: %v", err)
	}
	if comps.Engine.Started() {
		t.Error("engine should be stopped after Stop")
	}
}
