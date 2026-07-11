// Package plugindecls exercises AnnotatedDecl collection: a plugin-style
// annotation (@Exposed, unknown to the core registry) is still surfaced on the
// model alongside core annotations.
package plugindecls

import "context"

// @Application(name="plugindecls")
type Application struct{}

// @Service(name="svc")
type Svc struct{}

// NewSvc constructs the service.
func NewSvc() *Svc { return &Svc{} }

// Do carries a plugin-registered annotation.
//
// @Exposed
func (s *Svc) Do(ctx context.Context) error { return nil }
