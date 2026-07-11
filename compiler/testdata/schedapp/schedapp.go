// Package schedapp exercises @Scheduled background tasks.
package schedapp

import "context"

// @Application(name="sched-app")
type Application struct{}

// Reporter runs periodic background tasks.
//
// @Service(name="reporter")
type Reporter struct{}

func NewReporter() *Reporter { return &Reporter{} }

// Poll runs every 2 minutes, using the integer + timeUnit form.
//
// @Scheduled(fixedRate=2, timeUnit=TimeUnit.MINUTES)
func (r *Reporter) Poll(ctx context.Context) error { return nil }

// Heartbeat runs every 30 seconds using a Go duration string, after an initial
// delay.
//
// @Scheduled(fixedRate="30s", initialDelay="5s")
func (r *Reporter) Heartbeat() {}
