package model

import (
	"go/token"
	"go/types"
)

// Route is a single HTTP endpoint derived from a controller method (§33). It is
// router-independent metadata; adapters turn it into registration code.
type Route struct {
	// Method is the uppercase HTTP method, e.g. GET or POST.
	Method string
	// Pattern is the full path pattern including the controller's base path and
	// path parameters in {name} form, e.g. /api/v1/users/{id}.
	Pattern string
	// Name is an optional route name.
	Name string
	// Consumes and Produces are optional media-type constraints.
	Consumes []string
	Produces []string
	// Controller is the component that owns the handler method.
	Controller ComponentID
	// HandlerName is the controller method name, e.g. GetUser.
	HandlerName string

	// RequestType is the handler's request parameter type, or nil when the
	// handler takes only a context (§17.3).
	RequestType types.Type
	// RequestPointer reports whether the request parameter is a pointer.
	RequestPointer bool
	// ResponseType is the handler's response value type, or nil when the handler
	// returns only an error.
	ResponseType types.Type
	// SuccessStatus is the HTTP status written on success (§18.4).
	SuccessStatus int

	// Authorize holds the roles required by an @Authorize annotation, if any.
	Authorize []string

	// Position is the source location of the handler method.
	Position token.Position
}

// HasRequest reports whether the handler takes a request parameter.
func (r *Route) HasRequest() bool { return r.RequestType != nil }

// HasResponse reports whether the handler returns a response value.
func (r *Route) HasResponse() bool { return r.ResponseType != nil }
