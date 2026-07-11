package compiler

import (
	"go/types"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// resolve binds every component dependency to a providing component, following
// the resolution algorithm of §14.4: satisfy the dependency's type by exact
// match or interface implementation, prefer a @Primary candidate when several
// match, and report missing or ambiguous dependencies. Because the component
// list is pre-sorted by ID, candidate selection is deterministic.
func (a *analysis) resolve(app *model.Application) {
	for _, c := range app.Components {
		for i := range c.Dependencies {
			a.resolveDependency(app, c, &c.Dependencies[i])
		}
		// Keep the component's constructor params in sync with resolution so
		// later phases can read either.
		if c.Constructor != nil {
			c.Constructor.Params = c.Dependencies
		}
	}
}

func (a *analysis) resolveDependency(app *model.Application, consumer *model.Component, dep *model.Dependency) {
	// A pre-resolved dependency (e.g. a proxy wrapping its target) is left as
	// the analyzer set it.
	if dep.ResolvedTo != "" {
		return
	}
	// Reject injecting a proxied service by its concrete type: interception
	// only applies through the generated proxy, so the interface must be used
	// (§24.3).
	if target := proxiedConcreteMatch(app, consumer, dep.Type); target != nil {
		a.diags = append(a.diags, diagErr(CodeConcreteInjection, dep.Position,
			"%s injects %s by its concrete type %s, bypassing interception; inject the interface %s instead",
			consumer.Name, target.Name, typeString(dep.Type), typeString(target.Interface)))
		return
	}

	candidates := resolutionCandidates(app, consumer, dep.Type)
	switch len(candidates) {
	case 0:
		a.diags = append(a.diags, diagErr(CodeMissingDependency, dep.Position,
			"no component satisfies dependency %s %s required by %s",
			paramLabel(dep.Name), typeString(dep.Type), consumer.Name))
	case 1:
		dep.ResolvedTo = candidates[0].ID
	default:
		primaries := filterPrimary(candidates)
		if len(primaries) == 1 {
			dep.ResolvedTo = primaries[0].ID
			return
		}
		a.diags = append(a.diags, diagErr(CodeAmbiguousDependency, dep.Position,
			"dependency %s %s required by %s is ambiguous; candidates: %s (mark one @Primary or add a qualifier)",
			paramLabel(dep.Name), typeString(dep.Type), consumer.Name, candidateList(candidates)))
	}
}

// resolutionCandidates returns the components whose provided type can satisfy
// the dependency type, excluding the consumer itself. A component satisfies the
// dependency when its provided type is assignable to it, which covers both
// exact concrete matches and interface implementation (§14.3).
func resolutionCandidates(app *model.Application, consumer *model.Component, want types.Type) []*model.Component {
	var out []*model.Component
	for _, c := range app.Components {
		if c.ID == consumer.ID || c.ProvidedType == nil {
			continue
		}
		// A proxied target is reached only through its proxy; it never
		// satisfies a dependency directly (§24.3).
		if c.Proxied {
			continue
		}
		if types.AssignableTo(c.ProvidedType, want) {
			out = append(out, c)
		}
	}
	return out
}

// proxiedConcreteMatch returns a proxied component whose concrete type satisfies
// the dependency, when the dependency is not itself an interface — the concrete
// injection that §24.3 forbids. It returns nil otherwise.
func proxiedConcreteMatch(app *model.Application, consumer *model.Component, want types.Type) *model.Component {
	if isInterfaceType(want) {
		return nil
	}
	for _, c := range app.Components {
		if !c.Proxied || c.ID == consumer.ID {
			continue
		}
		if c.ProvidedType != nil && types.AssignableTo(c.ProvidedType, want) {
			return c
		}
	}
	return nil
}

// isInterfaceType reports whether t's underlying type is an interface.
func isInterfaceType(t types.Type) bool {
	_, ok := t.Underlying().(*types.Interface)
	return ok
}

// filterPrimary returns the primary components among the candidates.
func filterPrimary(candidates []*model.Component) []*model.Component {
	var out []*model.Component
	for _, c := range candidates {
		if c.Primary {
			out = append(out, c)
		}
	}
	return out
}

// candidateList renders candidate IDs for a diagnostic, sorted for determinism.
func candidateList(candidates []*model.Component) string {
	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = string(c.ID)
	}
	sort.Strings(ids)
	return strings.Join(ids, ", ")
}

// paramLabel renders a parameter name for diagnostics, tolerating the empty
// name of an unnamed parameter.
func paramLabel(name string) string {
	if name == "" {
		return "(unnamed)"
	}
	return name
}

// typeString renders a type for diagnostics without package-path noise.
func typeString(t types.Type) string {
	return types.TypeString(t, func(p *types.Package) string { return p.Name() })
}
