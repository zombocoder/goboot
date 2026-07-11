package model

import "time"

// InterceptedMethod records the interception a single service method requests
// via @Traced, @Timed, and @Transactional (§24, §25). The generator wraps the
// method with the corresponding runtime collaborators in a fixed order.
type InterceptedMethod struct {
	// Name is the method name.
	Name string

	// Traced requests a trace span (§35.1); TraceName overrides the span name.
	Traced    bool
	TraceName string

	// Timed requests success/failure metrics (§35.2); MetricName overrides the
	// metric name.
	Timed      bool
	MetricName string

	// Transactional requests a transaction wrapper (§26); Tx holds its options.
	Transactional bool
	Tx            TxOptions

	// Timeout, when > 0, wraps the call in a context with this timeout (§36.2).
	Timeout time.Duration
	// Retry, when non-nil, retries the call on error per the policy (§36.1).
	Retry *RetryPolicy
}

// RetryPolicy mirrors the @Retry arguments (§36.1) for the generator to render
// into a runtime.RetryPolicy literal.
type RetryPolicy struct {
	MaxAttempts int
	Delay       time.Duration
	Multiplier  float64
	MaxDelay    time.Duration
}

// TxOptions mirrors the @Transactional arguments (§26.1) in a form the generator
// renders into a runtime.TransactionOptions literal.
type TxOptions struct {
	ReadOnly    bool
	Isolation   string // "", "default", "read_committed", "repeatable_read", "serializable"
	Propagation string // "", "required", "requires_new", "supports", "not_supported"
	Timeout     time.Duration
}

// Intercepts reports whether the method requests any interception.
func (m InterceptedMethod) Intercepts() bool {
	return m.Traced || m.Timed || m.Transactional || m.Timeout > 0 || m.Retry != nil
}
