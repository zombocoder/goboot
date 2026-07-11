// Package app is a fixture that uses the example plugin's @Exposed annotation.
package app

import "context"

// @Application(name="plugin-demo")
type Application struct{}

// @Service(name="svc")
type Svc struct{}

func NewSvc() *Svc { return &Svc{} }

// Do is marked with the plugin-provided @Exposed annotation, which must be
// recognized (not reported as unknown) once the plugin is registered.
//
// @Exposed
func (s *Svc) Do(ctx context.Context) error { return nil }
