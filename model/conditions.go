package model

// Conditions captures the profile and conditional annotations attached to a
// component (§29). A component is included in the application only when all of
// its conditions hold; the analyzer evaluates them and drops components whose
// conditions fail before dependency resolution.
type Conditions struct {
	// Profiles are the profiles under which the component is active (§29.3).
	// Empty means active under every profile. When non-empty, at least one must
	// be in the active set.
	Profiles []string
	// Properties are @ConditionalOnProperty requirements (§29.1); all must hold.
	Properties []PropertyCondition
	// RequiredNuts are type names that must be provided by some other component
	// for this one to be included (@ConditionalOnNut, §29.2).
	RequiredNuts []string
	// AbsentNuts are type names that must NOT be provided by any other component
	// (@ConditionalOnMissingNut, §29.2).
	AbsentNuts []string
}

// IsEmpty reports whether the component has no conditions.
func (c Conditions) IsEmpty() bool {
	return len(c.Profiles) == 0 && len(c.Properties) == 0 &&
		len(c.RequiredNuts) == 0 && len(c.AbsentNuts) == 0
}

// PropertyCondition is a single @ConditionalOnProperty requirement (§29.1).
type PropertyCondition struct {
	// Name is the dotted property key, e.g. "cache.enabled".
	Name string
	// HavingValue, when non-empty, requires the property to equal it. When
	// empty, the property need only be present (unless MatchIfMissing applies).
	HavingValue string
	// MatchIfMissing makes the condition hold when the property is absent.
	MatchIfMissing bool
}

// Holds evaluates the property condition against a property set (§29.1).
func (p PropertyCondition) Holds(properties map[string]string) bool {
	value, present := properties[p.Name]
	if !present {
		return p.MatchIfMissing
	}
	if p.HavingValue == "" {
		return true // present is enough
	}
	return value == p.HavingValue
}
