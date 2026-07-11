// Package cfgapp exercises configuration properties and lifecycle hooks.
package cfgapp

import "context"

// Application is the root.
//
// @Application(name="cfg-app")
type Application struct{}

// ServerProperties is bound from configuration under the "server" prefix.
//
// @ConfigurationProperties(prefix="server")
type ServerProperties struct {
	Host string `config:"host" default:"0.0.0.0"`
	Port int    `config:"port" default:"8080"`
}

// Engine depends on the loaded configuration and has lifecycle hooks.
//
// @Service(name="engine")
type Engine struct {
	props   ServerProperties
	started bool
}

// NewEngine injects the configuration properties.
func NewEngine(props ServerProperties) *Engine {
	return &Engine{props: props}
}

// Started reports whether the engine has started (for tests).
func (e *Engine) Started() bool { return e.started }

// Addr returns the configured address.
func (e *Engine) Addr() string { return e.props.Host }

// Start is invoked after construction.
//
// @PostConstruct
func (e *Engine) Start(ctx context.Context) error {
	e.started = true
	return nil
}

// Stop is invoked before destruction.
//
// @PreDestroy
func (e *Engine) Stop() error {
	e.started = false
	return nil
}
