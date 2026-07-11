// Package runtime provides the minimal reusable abstractions that goboot's
// generated HTTP code depends on: request binding, validation, response
// writing, centralized error handling, and the RFC 7807-inspired problem model
// (§22, §23). It contains no dependency-injection logic and imports no router
// or database — those are supplied by adapters and generated code. The runtime
// is deliberately small; generated handlers wire these interfaces together.
package runtime

// Problem is an RFC 7807-inspired error response body (§23.1). It is the single
// shape every error is rendered into, so clients see a consistent envelope.
type Problem struct {
	// Type is a stable, machine-readable error identifier, e.g. "validation_error".
	Type string `json:"type"`
	// Title is a short human-readable summary.
	Title string `json:"title"`
	// Status is the HTTP status code.
	Status int `json:"status"`
	// Detail is an optional human-readable explanation specific to this
	// occurrence.
	Detail string `json:"detail,omitempty"`
	// Instance optionally identifies the specific occurrence (e.g. a request ID).
	Instance string `json:"instance,omitempty"`
	// Code is an optional application-specific error code.
	Code string `json:"code,omitempty"`
	// Errors carries per-field validation failures.
	Errors []FieldError `json:"errors,omitempty"`
	// Extensions carries additional, problem-specific members.
	Extensions map[string]any `json:"extensions,omitempty"`
}

// FieldError describes a single field-level validation failure (§20.4).
type FieldError struct {
	// Field is the name of the offending field.
	Field string `json:"field"`
	// Code is a machine-readable reason, e.g. "required" or "email".
	Code string `json:"code"`
	// Message is a human-readable description.
	Message string `json:"message"`
}
