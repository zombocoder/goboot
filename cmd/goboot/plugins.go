package main

import "github.com/zombocoder/goboot/plugin"

// builtinPlugins returns the plugins compiled into the default goboot CLI. It is
// empty by default: a project that needs plugins builds its own small main that
// imports them and calls the same run() entry point (§46.2). It is a variable so
// tests can inject plugins to exercise the host wiring.
var builtinPlugins = func() []plugin.Plugin { return nil }

// pluginHost builds the plugin registry for a CLI invocation.
func pluginHost() *plugin.Registry {
	return plugin.New(builtinPlugins()...)
}
