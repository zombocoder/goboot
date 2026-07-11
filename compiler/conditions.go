package compiler

import (
	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// extractConditions reads the profile and conditional annotations attached to a
// declaration into a model.Conditions (§29).
func extractConditions(decl *Declaration) model.Conditions {
	var c model.Conditions

	if ann, ok := decl.Find("Profile"); ok {
		if v, ok := ann.Positional(); ok {
			c.Profiles = stringList(v)
		}
	}
	for _, ann := range decl.FindAll("ConditionalOnProperty") {
		pc := model.PropertyCondition{}
		if s, ok := stringArgValue(ann, "name"); ok {
			pc.Name = s
		}
		if s, ok := stringArgValue(ann, "havingValue"); ok {
			pc.HavingValue = s
		}
		if b, ok := boolArg(ann, "matchIfMissing"); ok {
			pc.MatchIfMissing = b
		}
		c.Properties = append(c.Properties, pc)
	}
	for _, ann := range decl.FindAll("ConditionalOnNut") {
		if s, ok := stringArgValue(ann, "type"); ok {
			c.RequiredNuts = append(c.RequiredNuts, s)
		}
	}
	for _, ann := range decl.FindAll("ConditionalOnMissingNut") {
		if s, ok := stringArgValue(ann, "type"); ok {
			c.AbsentNuts = append(c.AbsentNuts, s)
		}
	}
	return c
}

// applyConditions removes components whose profiles or conditions are not
// satisfied (§29). Profile and property conditions are evaluated first (they
// depend only on the active set and property values), then nut-presence
// conditions are evaluated to a fixpoint, since removing one component can
// change whether another's @ConditionalOnNut/@ConditionalOnMissingNut holds.
func applyConditions(app *model.Application, opts Options) {
	active := toSet(opts.Profiles)

	var remaining []*model.Component
	for _, c := range app.Components {
		if profileActive(c.Conditions.Profiles, active) && propertiesHold(c.Conditions.Properties, opts.Properties) {
			remaining = append(remaining, c)
		}
	}

	for {
		var next []*model.Component
		changed := false
		for _, c := range remaining {
			if nutConditionsHold(c, remaining) {
				next = append(next, c)
			} else {
				changed = true
			}
		}
		remaining = next
		if !changed {
			break
		}
	}

	app.Components = remaining
}

// profileActive reports whether a component's profiles include an active one. An
// empty profile list means the component is active under every profile.
func profileActive(profiles []string, active map[string]bool) bool {
	if len(profiles) == 0 {
		return true
	}
	for _, p := range profiles {
		if active[p] {
			return true
		}
	}
	return false
}

// propertiesHold reports whether every property condition holds.
func propertiesHold(conditions []model.PropertyCondition, properties map[string]string) bool {
	for _, pc := range conditions {
		if !pc.Holds(properties) {
			return false
		}
	}
	return true
}

// nutConditionsHold reports whether a component's @ConditionalOnNut and
// @ConditionalOnMissingNut requirements hold against the remaining components.
func nutConditionsHold(c *model.Component, remaining []*model.Component) bool {
	for _, required := range c.Conditions.RequiredNuts {
		if !typeProvided(required, c, remaining) {
			return false
		}
	}
	for _, absent := range c.Conditions.AbsentNuts {
		if typeProvided(absent, c, remaining) {
			return false
		}
	}
	return true
}

// typeProvided reports whether some component other than self provides a type or
// component of the given name. Matching is by the provided type's name or the
// component's declared name.
func typeProvided(name string, self *model.Component, components []*model.Component) bool {
	for _, c := range components {
		if c.ID == self.ID {
			continue
		}
		if c.Name == name {
			return true
		}
		if c.Named != nil && c.Named.Obj().Name() == name {
			return true
		}
	}
	return false
}

// stringList extracts a []string from an annotation array value.
func stringList(v annotation.Value) []string {
	arr, ok := v.(annotation.ArrayValue)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr.Elements))
	for _, e := range arr.Elements {
		if s, ok := annotation.AsString(e); ok {
			out = append(out, s)
		}
	}
	return out
}

// boolArg extracts a boolean named argument from an annotation.
func boolArg(ann annotation.Annotation, name string) (bool, bool) {
	v, ok := ann.Arg(name)
	if !ok {
		return false, false
	}
	b, ok := v.(annotation.BoolValue)
	if !ok {
		return false, false
	}
	return b.Val, true
}

// toSet builds a set from a string slice.
func toSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, s := range items {
		set[s] = true
	}
	return set
}
