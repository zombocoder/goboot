// Package bad is a fixture whose handler has no message parameter, exercising
// the analyzer's GOBASY001 warning.
package bad

import "context"

// @Application(name="bad-events")
type Application struct{}

// @Service(name="pinger")
type Pinger struct{}

// NewPinger constructs it.
func NewPinger() *Pinger { return &Pinger{} }

// OnPing has no payload parameter, only a context.
//
// @Listener(channel="pings")
func (p *Pinger) OnPing(ctx context.Context) error { return nil }
