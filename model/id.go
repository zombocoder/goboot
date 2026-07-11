package model

// ComponentID stably identifies a component across builds (§12.3). The format
// is "<package-import-path>:<type-or-function-name>" for ordinary components,
// with a "#<nut-name>" suffix for named nut providers, e.g.
//
//	github.com/acme/users/internal/service:UserService
//	github.com/acme/users/internal/config:ProvideDatabase#primaryDatabase
//
// IDs are derived only from source identity, never from map iteration order or
// filesystem order, so that generated output is deterministic (§6.7).
type ComponentID string

// NewComponentID builds an ID from a package path and declared name.
func NewComponentID(pkgPath, name string) ComponentID {
	return ComponentID(pkgPath + ":" + name)
}

// NewNutID builds an ID for a named nut provider.
func NewNutID(pkgPath, funcName, nutName string) ComponentID {
	id := pkgPath + ":" + funcName
	if nutName != "" {
		id += "#" + nutName
	}
	return ComponentID(id)
}

// String returns the ID as a plain string.
func (id ComponentID) String() string { return string(id) }
