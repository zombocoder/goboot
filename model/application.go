package model

import "sort"

// Application is the assembled intermediate model consumed by the code
// generators (§38). It intentionally holds no reference to any concrete router
// or database implementation. The dependency graph and diagnostics are tracked
// alongside it by the analyzer rather than embedded here, keeping this package
// free of the graph dependency.
type Application struct {
	// Name is the application name from @Application.
	Name string
	// RootPackage is the import path of the package declaring @Application.
	RootPackage string
	// Components are all discovered components. Analyzers should keep this
	// sorted by ID for deterministic output; SortComponents enforces it.
	Components []*Component
	// Controllers are the discovered HTTP controllers with their routes, sorted
	// by component ID.
	Controllers []*Controller
	// Routes is the flattened list of every route across all controllers,
	// sorted by (pattern, method) for deterministic registration.
	Routes []*Route
}

// SortComponents orders components by their stable ID so that any downstream
// iteration is deterministic regardless of discovery order (§6.7).
func (a *Application) SortComponents() {
	sort.Slice(a.Components, func(i, j int) bool {
		return a.Components[i].ID < a.Components[j].ID
	})
}

// ComponentByID returns the component with the given ID, or nil.
func (a *Application) ComponentByID(id ComponentID) *Component {
	for _, c := range a.Components {
		if c.ID == id {
			return c
		}
	}
	return nil
}
