// Package asyncapi is a goboot plugin that emits an AsyncAPI 3.0 description of
// an application's event-driven surface (github.com/zombocoder/goboot issue #37)
// — the message-driven counterpart to the openapi plugin. It registers
// @Listener / @Publisher annotations (AnnotationProvider), reads them off the
// annotated handler methods via the deeper plugin API, and writes a
// deterministic zz_goboot_asyncapi.json (Generator):
//
//	// @Listener(channel="orders.created")
//	func (h *OrderHandler) OnOrderCreated(ctx context.Context, evt OrderCreated) error { ... }
//
// generates a channel `orders.created` carrying an `OrderCreated` message (its
// payload schema derived from the Go struct) and a `receive` operation. A
// @Publisher method yields a `send` operation instead.
//
// Register it in goboot.yaml:
//
//	plugins:
//	  - github.com/zombocoder/goboot/plugins/asyncapi
package asyncapi

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Annotation names this plugin owns.
const (
	annListener  = "Listener"
	annPublisher = "Publisher"
)

// outputFile is the generated document's name; the zz_goboot_ prefix lets
// `goboot clean` remove it (§40).
const outputFile = "zz_goboot_asyncapi.json"

// Plugin implements the AnnotationProvider, Analyzer, and Generator capabilities.
type Plugin struct{}

// New constructs the asyncapi plugin for injection into cli.Main.
func New() *Plugin { return &Plugin{} }

// Name identifies the plugin within a host.
func (*Plugin) Name() string { return "asyncapi" }

// Version is the plugin's own version.
func (*Plugin) Version() string { return "0.1.0" }

// Annotations registers @Listener and @Publisher on message-handler methods.
func (*Plugin) Annotations() []*annotation.Definition {
	args := map[string]annotation.ArgumentDefinition{
		"channel": {Type: annotation.ArgString, Required: true},
		"message": {Type: annotation.ArgString},
		"summary": {Type: annotation.ArgString},
	}
	return []*annotation.Definition{
		{Name: annListener, Targets: []annotation.Target{annotation.TargetMethod, annotation.TargetFunction}, Arguments: args},
		{Name: annPublisher, Targets: []annotation.Target{annotation.TargetMethod, annotation.TargetFunction}, Arguments: args},
	}
}

// hasOurAnnotation reports whether a declaration carries @Listener or @Publisher.
func hasOurAnnotation(d model.AnnotatedDecl) bool {
	return d.Has(annListener) || d.Has(annPublisher)
}
