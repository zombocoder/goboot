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
		if types.AssignableTo(c.ProvidedType, want) {
			out = append(out, c)
		}
	}
	return out
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
