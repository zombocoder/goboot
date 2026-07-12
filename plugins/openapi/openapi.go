// Package openapi is a goboot plugin that emits an OpenAPI 3 description of the
// application's HTTP surface (§46.1, Generator capability). It reads the routes,
// request/response types, and binding tags from the analyzed model and writes a
// deterministic zz_goboot_openapi.json alongside the generated wiring.
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/openapi@latest
//
// The spec is regenerated on every `goboot generate`.
package openapi

import (
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/plugin"
)

// outputFile is the generated spec's name; the zz_goboot_ prefix lets
// `goboot clean` remove it (§40).
const outputFile = "zz_goboot_openapi.json"

// New constructs the OpenAPI plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Plugin implements the Generator capability, emitting an OpenAPI document.
type Plugin struct{}

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "openapi" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.1" }

// Generate builds the OpenAPI document from the application's routes. It emits
// nothing when the application declares no routes.
func (*Plugin) Generate(app *model.Application) ([]plugin.File, error) {
	if len(app.Routes) == 0 {
		return nil, nil
	}
	content, err := buildDocument(app)
	if err != nil {
		return nil, err
	}
	return []plugin.File{{Name: outputFile, Content: content}}, nil
}

// Compile-time assertions of the implemented capabilities.
var (
	_ plugin.Plugin    = (*Plugin)(nil)
	_ plugin.Generator = (*Plugin)(nil)
)
