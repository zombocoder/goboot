package model

// ComponentID stably identifies a component across builds (§12.3). The format
// is "<package-import-path>:<type-or-function-name>" for ordinary components,
// with a "#<bean-name>" suffix for named bean providers, e.g.
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

// NewBeanID builds an ID for a named bean provider.
func NewBeanID(pkgPath, funcName, beanName string) ComponentID {
	id := pkgPath + ":" + funcName
	if beanName != "" {
		id += "#" + beanName
	}
	return ComponentID(id)
}

// String returns the ID as a plain string.
func (id ComponentID) String() string { return string(id) }
