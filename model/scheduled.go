package model

import "time"

// ScheduledMethod is a component method that runs periodically (@Scheduled). The
// generator registers it with the runtime scheduler, adapting the method's
// signature to the scheduler's callback.
type ScheduledMethod struct {
	// MethodName is the method to invoke on the component instance.
	MethodName string
	// Interval is the fixed rate between runs, resolved from fixedRate/timeUnit
	// or a duration string.
	Interval time.Duration
	// InitialDelay optionally delays the first run.
	InitialDelay time.Duration
	// TakesContext reports whether the method accepts a context.Context.
	TakesContext bool
	// ReturnsError reports whether the method returns an error.
	ReturnsError bool
}
