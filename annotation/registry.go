package annotation

import "fmt"

// Registry holds annotation schemas keyed by name and validates parsed
// annotations against them. It is the schema authority described in §9.5 and is
// the extension point through which plugins contribute annotations (§46.1).
type Registry struct {
	defs map[string]*Definition
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{defs: map[string]*Definition{}}
}

// Register adds a definition. It returns an error if the name is empty or
// already registered, so that duplicate or plugin-conflicting annotations are
// reported deterministically rather than silently overwriting.
func (r *Registry) Register(d *Definition) error {
	if d == nil || d.Name == "" {
		return fmt.Errorf("annotation: cannot register definition with empty name")
	}
	if _, exists := r.defs[d.Name]; exists {
		return fmt.Errorf("annotation: %q is already registered", d.Name)
	}
	r.defs[d.Name] = d
	return nil
}

// MustRegister is like Register but panics on error. It is intended for static
// initialization of built-in definitions, never for untrusted input.
func (r *Registry) MustRegister(d *Definition) {
	if err := r.Register(d); err != nil {
		panic(err)
	}
}

// Lookup returns the definition for name and whether it exists.
func (r *Registry) Lookup(name string) (*Definition, bool) {
	d, ok := r.defs[name]
	return d, ok
}

// Validate checks a single annotation against its registered schema for the
// given target. An unregistered annotation yields a single warning-severity
// diagnostic (GOBANN002) rather than an error, so that unknown or plugin
// annotations do not block a build unless strict mode promotes the warning.
func (r *Registry) Validate(ann Annotation, target Target) []*Diagnostic {
	def, ok := r.defs[ann.Name]
	if !ok {
		return []*Diagnostic{{
			Severity: SeverityWarning,
			Code:     CodeUnknownAnnotation,
			Message:  fmt.Sprintf("unknown annotation @%s", ann.Name),
			Position: ann.Position,
		}}
	}
	return def.validate(ann, target)
}

// ValidateGroup validates every annotation attached to a single declaration,
// additionally enforcing that non-repeatable annotations appear at most once
// (GOBANN009).
func (r *Registry) ValidateGroup(anns []Annotation, target Target) []*Diagnostic {
	var diags []*Diagnostic
	counts := map[string]int{}
	for _, ann := range anns {
		counts[ann.Name]++
		diags = append(diags, r.Validate(ann, target)...)
	}
	for name, n := range counts {
		if n <= 1 {
			continue
		}
		if def, ok := r.defs[name]; ok && !def.Repeatable {
			// Anchor the diagnostic at the first duplicate occurrence.
			for _, ann := range anns {
				if ann.Name == name {
					diags = append(diags, newError(CodeNotRepeatable, ann.Position,
						"annotation @%s may not be repeated on the same declaration", name))
					break
				}
			}
		}
	}
	return diags
}

// arg is a small helper for building ArgumentDefinition maps concisely.
func arg(t ArgumentType) ArgumentDefinition { return ArgumentDefinition{Type: t} }

func required(t ArgumentType) ArgumentDefinition {
	return ArgumentDefinition{Type: t, Required: true}
}

func enum(allowed ...string) ArgumentDefinition {
	return ArgumentDefinition{Type: ArgStringOrIdent, Allowed: allowed}
}

// DefaultRegistry returns a registry preloaded with the v0.1 core annotation
// catalogue (§54.1) plus the closely related DI annotations they depend on.
// Annotations outside v0.1 scope are intentionally omitted; plugins register
// the remainder.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, d := range coreDefinitions() {
		r.MustRegister(d)
	}
	return r
}

func coreDefinitions() []*Definition {
	return []*Definition{
		// ---- Application root -------------------------------------------
		{
			Name:    "Application",
			Targets: []Target{TargetStruct, TargetType},
			Arguments: map[string]ArgumentDefinition{
				"name":          required(ArgString),
				"scan":          arg(ArgStringArray),
				"profiles":      arg(ArgStringArray),
				"configuration": arg(ArgString),
			},
		},

		// ---- Core components --------------------------------------------
		{Name: "Component", Targets: []Target{TargetStruct, TargetType},
			Arguments: map[string]ArgumentDefinition{"name": arg(ArgString)}},
		{Name: "Service", Targets: []Target{TargetStruct, TargetType},
			Arguments: map[string]ArgumentDefinition{
				"name":       arg(ArgString),
				"scope":      enum("singleton", "prototype"),
				"implements": arg(ArgString),
			}},
		{Name: "Repository", Targets: []Target{TargetStruct, TargetInterface, TargetType},
			Arguments: map[string]ArgumentDefinition{
				"name":     arg(ArgString),
				"entity":   arg(ArgString),
				"table":    arg(ArgString),
				"generate": arg(ArgBoolean),
			}},
		{Name: "Configuration", Targets: []Target{TargetStruct, TargetType}},
		{Name: "Nut", Targets: []Target{TargetFunction, TargetMethod},
			Arguments: map[string]ArgumentDefinition{"name": arg(ArgString)}},
		{Name: "Primary", Targets: []Target{TargetStruct, TargetType, TargetFunction, TargetMethod}},
		{Name: "Named", Targets: []Target{TargetStruct, TargetType, TargetFunction, TargetMethod},
			Positional: &ArgumentDefinition{Type: ArgString, Required: true}},
		{Name: "Scope", Targets: []Target{TargetStruct, TargetType},
			Positional: &ArgumentDefinition{Type: ArgStringOrIdent, Required: true,
				Allowed: []string{"singleton", "prototype"}}},

		// ---- HTTP -------------------------------------------------------
		{Name: "RestController", Targets: []Target{TargetStruct, TargetType}},
		{Name: "RequestMapping", Targets: []Target{TargetStruct, TargetType},
			Arguments: map[string]ArgumentDefinition{
				"path":    arg(ArgString),
				"host":    arg(ArgString),
				"headers": arg(ArgStringArray),
			}},
		{Name: "GetMapping", Targets: []Target{TargetMethod}, Arguments: methodMappingArgs()},
		{Name: "PostMapping", Targets: []Target{TargetMethod}, Arguments: methodMappingArgs()},
		{Name: "Response", Targets: []Target{TargetMethod}, Repeatable: true,
			Arguments: map[string]ArgumentDefinition{
				"status":      arg(ArgInteger),
				"type":        arg(ArgString),
				"error":       arg(ArgString),
				"contentType": arg(ArgString),
			}},
		{Name: "ResponseStatus", Targets: []Target{TargetMethod},
			Positional: &ArgumentDefinition{Type: ArgInteger, Required: true}},

		// ---- Error handling ---------------------------------------------
		{Name: "ControllerAdvice", Targets: []Target{TargetStruct, TargetType}},
		{Name: "ExceptionHandler", Targets: []Target{TargetMethod},
			Arguments: map[string]ArgumentDefinition{"type": required(ArgString)}},

		// ---- Configuration properties -----------------------------------
		{Name: "ConfigurationProperties", Targets: []Target{TargetStruct, TargetType},
			Arguments: map[string]ArgumentDefinition{"prefix": required(ArgString)}},

		// ---- Lifecycle --------------------------------------------------
		{Name: "PostConstruct", Targets: []Target{TargetMethod}},
		{Name: "PreDestroy", Targets: []Target{TargetMethod}},

		// ---- Scheduling -------------------------------------------------
		{Name: "Scheduled", Targets: []Target{TargetMethod},
			Arguments: map[string]ArgumentDefinition{
				"fixedRate":    arg(ArgAny), // integer with timeUnit, or a duration string
				"fixedDelay":   arg(ArgAny),
				"initialDelay": arg(ArgAny),
				"timeUnit":     arg(ArgStringOrIdent),
			}},

		// ---- Conditions and profiles ------------------------------------
		{Name: "Profile", Targets: componentAndProviderTargets(),
			Positional: &ArgumentDefinition{Type: ArgStringArray, Required: true}},
		{Name: "ConditionalOnProperty", Targets: componentAndProviderTargets(),
			Arguments: map[string]ArgumentDefinition{
				"name":           required(ArgString),
				"havingValue":    arg(ArgString),
				"matchIfMissing": arg(ArgBoolean),
			}},
		{Name: "ConditionalOnNut", Targets: componentAndProviderTargets(),
			Arguments: map[string]ArgumentDefinition{"type": required(ArgString)}},
		{Name: "ConditionalOnMissingNut", Targets: componentAndProviderTargets(),
			Arguments: map[string]ArgumentDefinition{"type": required(ArgString)}},

		// ---- Interception (service proxies) -----------------------------
		{Name: "Transactional", Targets: []Target{TargetMethod, TargetType},
			Arguments: map[string]ArgumentDefinition{
				"readOnly":    arg(ArgBoolean),
				"isolation":   enum("default", "read_committed", "repeatable_read", "serializable"),
				"propagation": enum("required", "requires_new", "supports", "not_supported"),
				"timeout":     arg(ArgDuration),
			}},
		{Name: "Traced", Targets: []Target{TargetMethod, TargetType},
			Arguments: map[string]ArgumentDefinition{"name": arg(ArgString)}},
		{Name: "Timed", Targets: []Target{TargetMethod, TargetType},
			Arguments: map[string]ArgumentDefinition{"name": arg(ArgString)}},

		// ---- Repository queries -----------------------------------------
		{Name: "Query", Targets: []Target{TargetMethod},
			Positional: &ArgumentDefinition{Type: ArgString},
			Arguments:  map[string]ArgumentDefinition{"file": arg(ArgString)}},
		{Name: "Exec", Targets: []Target{TargetMethod},
			Positional: &ArgumentDefinition{Type: ArgString},
			Arguments:  map[string]ArgumentDefinition{"file": arg(ArgString)}},
	}
}

// componentAndProviderTargets is the set of declarations a condition or profile
// annotation may be attached to: component types and nut/bean provider funcs.
func componentAndProviderTargets() []Target {
	return []Target{TargetStruct, TargetType, TargetInterface, TargetFunction, TargetMethod}
}

func methodMappingArgs() map[string]ArgumentDefinition {
	return map[string]ArgumentDefinition{
		"path":     arg(ArgString),
		"name":     arg(ArgString),
		"consumes": arg(ArgStringArray),
		"produces": arg(ArgStringArray),
		"timeout":  arg(ArgDuration),
		"status":   arg(ArgInteger),
	}
}
