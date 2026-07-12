// Package config loads typed configuration properties from layered sources —
// defaults, YAML files, and environment variables — following the precedence of
// §28.2. Generated loaders call Bind to populate an @ConfigurationProperties
// struct; the reflection used here is runtime data transformation, not
// dependency discovery, and so is consistent with the framework's compile-time
// principles (§19.2).
package config

import (
	"os"
	"strings"
)

// Source supplies raw configuration values by dotted, lowercase key such as
// "server.read-timeout". Keys mirror the structure of the configuration and are
// independent of any particular backing store.
type Source interface {
	// Get returns the value for a key and whether it is present.
	Get(key string) (string, bool)
}

// MapSource is a Source backed by an in-memory map, typically produced by
// flattening a YAML document.
type MapSource map[string]string

// Get implements Source.
func (m MapSource) Get(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

// EnvSource reads values from environment variables. A dotted key is
// transformed to an upper-snake variable name, optionally prefixed: the key
// "server.read-timeout" with prefix "USERS" becomes USERS_SERVER_READ_TIMEOUT
// (§28.4).
type EnvSource struct {
	// Prefix is prepended (with an underscore) to every variable name; empty
	// means no prefix.
	Prefix string
	// Getenv looks up a variable; nil defaults to os.Getenv, allowing tests to
	// inject a fake environment.
	Getenv func(string) string
}

// Get implements Source.
func (e EnvSource) Get(key string) (string, bool) {
	getenv := e.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	name := EnvName(e.Prefix, key)
	if v := getenv(name); v != "" {
		return v, true
	}
	return "", false
}

// EnvName converts a dotted configuration key into an environment variable name:
// the prefix and key are joined, then every "." and "-" separator is replaced
// with "_" and the whole name upper-cased. The prefix is sanitized too, so a
// dotted or hyphenated prefix cannot produce an invalid variable name.
func EnvName(prefix, key string) string {
	name := key
	if prefix != "" {
		name = prefix + "_" + key
	}
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(name))
}

// Layered composes sources in priority order: the first source to report a key
// wins. Compose them highest priority first, e.g. Layered(env, file) so that
// environment variables override file values (§28.2).
type Layered []Source

// Get implements Source.
func (l Layered) Get(key string) (string, bool) {
	for _, s := range l {
		if v, ok := s.Get(key); ok {
			return v, true
		}
	}
	return "", false
}
