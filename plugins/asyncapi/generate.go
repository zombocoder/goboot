package asyncapi

import (
	"encoding/json"
	"fmt"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/plugin"
)

// Analyze validates the @Listener/@Publisher handlers (§46.1).
func (*Plugin) Analyze(app *model.Application) []*annotation.Diagnostic {
	_, diags := resolve(app)
	return diags
}

// Generate builds the AsyncAPI 3.0 document from the message handlers. It emits
// nothing when the application declares none. Output is deterministic —
// json.MarshalIndent sorts object keys.
func (*Plugin) Generate(app *model.Application) ([]plugin.File, error) {
	handlers, _ := resolve(app) // diagnostics surface via Analyze
	if len(handlers) == 0 {
		return nil, nil
	}
	content, err := buildDocument(app, handlers)
	if err != nil {
		return nil, fmt.Errorf("asyncapi: %w", err)
	}
	return []plugin.File{{Name: outputFile, Content: content}}, nil
}

// buildDocument assembles the AsyncAPI 3.0 document: channels (with their
// messages), operations (send/receive), and message payload schemas.
func buildDocument(app *model.Application, handlers []handler) ([]byte, error) {
	channels := map[string]any{}
	operations := map[string]any{}
	schemas := map[string]any{}

	for _, h := range handlers {
		ch, ok := channels[h.channelKey].(map[string]any)
		if !ok {
			ch = map[string]any{"address": h.channel, "messages": map[string]any{}}
			channels[h.channelKey] = ch
		}
		payload := map[string]any{"type": "object"}
		if h.payload != nil {
			payload = schemaFor(h.payload, schemas)
		}
		ch["messages"].(map[string]any)[h.messageName] = map[string]any{
			"name":    h.messageName,
			"payload": payload,
		}

		op := map[string]any{
			"action":  h.action.String(),
			"channel": map[string]any{"$ref": "#/channels/" + h.channelKey},
			"messages": []any{
				map[string]any{"$ref": "#/channels/" + h.channelKey + "/messages/" + h.messageName},
			},
		}
		if h.summary != "" {
			op["summary"] = h.summary
		}
		operations[h.opID] = op
	}

	doc := map[string]any{
		"asyncapi":   "3.0.0",
		"info":       map[string]any{"title": titleOf(app), "version": "1.0.0"},
		"channels":   channels,
		"operations": operations,
	}
	if len(schemas) > 0 {
		doc["components"] = map[string]any{"schemas": schemas}
	}
	return json.MarshalIndent(doc, "", "  ")
}

// titleOf derives the document title from the application name.
func titleOf(app *model.Application) string {
	if app.Name != "" {
		return app.Name
	}
	return "goboot application"
}
