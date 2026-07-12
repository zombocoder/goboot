package asyncapi

import (
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// codeNoPayload warns that a handler has no message parameter.
const codeNoPayload = "GOBASY001"

// action is the AsyncAPI operation direction.
type action int

const (
	receiveAction action = iota // @Listener: the application receives
	sendAction                  // @Publisher: the application sends
)

func (a action) String() string {
	if a == sendAction {
		return "send"
	}
	return "receive"
}

// handler is a resolved @Listener/@Publisher declaration.
type handler struct {
	channel     string // channel address
	channelKey  string // sanitized key used in the document and $refs
	opID        string // operation id
	action      action
	payload     types.Type // message payload type, or nil
	messageName string
	summary     string
	pos         token.Position
}

// resolve reads every @Listener/@Publisher handler into the model, sorted for
// determinism, and returns diagnostics.
func resolve(app *model.Application) ([]handler, []*annotation.Diagnostic) {
	var handlers []handler
	var diags []*annotation.Diagnostic

	for _, d := range app.Declarations {
		if !hasOurAnnotation(d) {
			continue
		}
		for _, a := range d.Annotations {
			var act action
			switch a.Name {
			case annListener:
				act = receiveAction
			case annPublisher:
				act = sendAction
			default:
				continue
			}
			channel := stringArg(a, "channel")
			if channel == "" {
				continue // required by the schema; reported there
			}
			payload := payloadType(d.Signature)
			if payload == nil {
				diags = append(diags, diag(annotation.SeverityWarning, codeNoPayload, a.Position,
					"@%s handler %s has no message parameter; its operation carries an empty payload", a.Name, d.Name))
			}
			msg := stringArg(a, "message")
			if msg == "" {
				msg = messageName(payload)
			}
			handlers = append(handlers, handler{
				channel:     channel,
				channelKey:  sanitizeKey(channel),
				opID:        lowerFirst(d.Name),
				action:      act,
				payload:     payload,
				messageName: msg,
				summary:     stringArg(a, "summary"),
				pos:         a.Position,
			})
		}
	}
	sort.Slice(handlers, func(i, j int) bool {
		if handlers[i].channelKey != handlers[j].channelKey {
			return handlers[i].channelKey < handlers[j].channelKey
		}
		return handlers[i].opID < handlers[j].opID
	})
	return handlers, diags
}

// payloadType returns the first non-context parameter type of sig, or nil.
func payloadType(sig *types.Signature) types.Type {
	if sig == nil {
		return nil
	}
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		if t := params.At(i).Type(); !isContext(t) {
			return t
		}
	}
	return nil
}

func isContext(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	o := named.Obj()
	return o.Pkg() != nil && o.Pkg().Path() == "context" && o.Name() == "Context"
}

// messageName derives a message name from the payload type.
func messageName(t types.Type) string {
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
	}
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Name()
	}
	return "Message"
}

func lowerFirst(s string) string {
	if s == "" {
		return "operation"
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// sanitizeKey turns a channel address into a document/$ref-safe key.
func sanitizeKey(addr string) string {
	var b strings.Builder
	for _, r := range addr {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "channel"
	}
	return b.String()
}
