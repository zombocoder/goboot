// Package annotation implements goboot's `@Name(arg=value, ...)` comment
// annotation language (§9): the lexer and parser, the value model, and the
// schema registry that validates parsed annotations against their definitions.
// Parsers never panic on arbitrary input — malformed comments yield diagnostics,
// not crashes.
package annotation
