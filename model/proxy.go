package model

import (
	"regexp"
	"time"
)

// cacheKeyPlaceholder matches a #{name} placeholder in a cache key template.
var cacheKeyPlaceholder = regexp.MustCompile(`#\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// CacheKeySegment is one piece of a parsed cache key template: either a literal
// string (Param == "") or a #{param} placeholder (Literal == "").
type CacheKeySegment struct {
	Literal string
	Param   string
}

// ParseCacheKey splits a key template into ordered literal and placeholder
// segments, so the compiler can validate placeholder names and the generator
// can render the key expression.
func ParseCacheKey(key string) []CacheKeySegment {
	var segs []CacheKeySegment
	last := 0
	for _, m := range cacheKeyPlaceholder.FindAllStringSubmatchIndex(key, -1) {
		if m[0] > last {
			segs = append(segs, CacheKeySegment{Literal: key[last:m[0]]})
		}
		segs = append(segs, CacheKeySegment{Param: key[m[2]:m[3]]})
		last = m[1]
	}
	if last < len(key) {
		segs = append(segs, CacheKeySegment{Literal: key[last:]})
	}
	return segs
}

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
	// Authorize, when non-nil, checks authorization before invoking the target
	// (§34).
	Authorize *AuthorizeSpec

	// Logged requests structured logging around the call (§35.3); LogLevel is
	// the level ("debug"|"info"|"warn"|"error", default "info").
	Logged   bool
	LogLevel string

	// Audit, when non-nil, records an audit event after the call (§35.4).
	Audit *AuditSpec

	// CircuitBreaker, RateLimit, and Bulkhead, when non-nil, gate the call with
	// the corresponding resilience interceptor (§36.3–§36.5).
	CircuitBreaker *CircuitBreakerSpec
	RateLimit      *RateLimitSpec
	Bulkhead       *BulkheadSpec

	// Cacheable, when non-nil, wraps the call in read-through caching: a hit
	// returns the cached result without invoking the target (§32). Requires the
	// method to return exactly one value and an error.
	Cacheable *CacheSpec
	// CacheEvict, when non-nil, deletes the keyed cache entry after the method
	// succeeds (§32).
	CacheEvict *CacheSpec
}

// CacheSpec mirrors @Cacheable / @CacheEvict for the generator. Key is the raw
// template (for diagnostics); Parts is its resolution into literals and argument
// references, so the generator can render a key expression independent of the
// interface method's parameter names. TTL applies to @Cacheable only (0 means no
// expiry).
type CacheSpec struct {
	Key   string
	TTL   time.Duration
	Parts []CacheKeyPart
}

// CacheKeyPart is one piece of a resolved cache key: a literal string, or a
// reference to a method argument by its index in the parameter list (the leading
// context parameter is index 0, so argument references are >= 1).
type CacheKeyPart struct {
	Literal  string
	ArgIndex int
	IsArg    bool
}

// CircuitBreakerSpec mirrors @CircuitBreaker (§36.3). Zero fields let the
// runtime apply its defaults.
type CircuitBreakerSpec struct {
	Name             string
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenMax      int
}

// RateLimitSpec mirrors @RateLimit (§36.4).
type RateLimitSpec struct {
	Name   string
	Limit  int
	Period time.Duration
	Burst  int
}

// BulkheadSpec mirrors @Bulkhead (§36.5).
type BulkheadSpec struct {
	Name          string
	MaxConcurrent int
	MaxWait       time.Duration
}

// AuditSpec mirrors @Audit for the generator to render an AuditEvent (§35.4).
type AuditSpec struct {
	Action   string
	Resource string
}

// AuthorizeSpec mirrors @Authorize/@RolesAllowed for the generator to render an
// authorization check (§34.1).
type AuthorizeSpec struct {
	Roles       []string
	Permissions []string
	// Mode is "any" (default) or "all".
	Mode string
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
	return m.Traced || m.Timed || m.Transactional || m.Timeout > 0 || m.Retry != nil ||
		m.Authorize != nil || m.Logged || m.Audit != nil ||
		m.CircuitBreaker != nil || m.RateLimit != nil || m.Bulkhead != nil ||
		m.Cacheable != nil || m.CacheEvict != nil
}
