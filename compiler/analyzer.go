package compiler

import (
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/graph"
	"github.com/zombocoder/goboot/model"
)

// AnalysisResult is the semantic analysis output: the assembled application
// model, its dependency graph, and every diagnostic gathered from scanning,
// discovery, resolution, and cycle detection.
type AnalysisResult struct {
	App         *model.Application
	Graph       *graph.Graph
	Diagnostics []*annotation.Diagnostic
}

// HasErrors reports whether any diagnostic is error severity.
func (r *AnalysisResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == annotation.SeverityError {
			return true
		}
	}
	return false
}

// componentKind maps a struct-target component annotation to its model kind, in
// the priority order used when a declaration carries more than one.
var componentKind = []struct {
	annotation string
	kind       model.ComponentKind
}{
	{"Service", model.ComponentService},
	{"Repository", model.ComponentRepository},
	{"RestController", model.ComponentController},
	{"ControllerAdvice", model.ComponentAdvice},
	{"ConfigurationProperties", model.ComponentConfigProperties},
	{"Configuration", model.ComponentConfiguration},
	{"Component", model.ComponentGeneric},
}

// analysis accumulates state across the discovery, resolution, and graph phases.
type analysis struct {
	diags    []*annotation.Diagnostic
	appCount int
}

// Analyze performs semantic analysis (§37.5–§37.7) over a scan result: it
// discovers components and their constructors, resolves the dependency graph,
// and reports diagnostics. Scan diagnostics are carried through so callers see
// a single combined list.
func Analyze(scan *ScanResult) *AnalysisResult {
	a := &analysis{}
	a.diags = append(a.diags, scan.Diagnostics...)

	app := &model.Application{}
	for _, decl := range scan.Declarations {
		a.discover(decl, app)
	}
	a.checkAppRoot()
	app.SortComponents()

	a.resolve(app)
	a.discoverRoutes(scan, app)
	a.discoverLifecycle(scan, app)

	g := graph.Build(app.Components)
	if _, cyc := g.ConstructionOrder(); cyc != nil {
		a.diags = append(a.diags, cycleDiagnostic(cyc, app))
	}

	return &AnalysisResult{App: app, Graph: g, Diagnostics: a.diags}
}

// discover creates model components from a single annotated declaration.
func (a *analysis) discover(decl *Declaration, app *model.Application) {
	if decl.Has("Application") {
		a.handleAppRoot(decl, app)
	}
	// A bean provider is a function or method annotated @Bean.
	if decl.Has("Bean") && (decl.Target == annotation.TargetFunction || decl.Target == annotation.TargetMethod) {
		a.discoverBean(decl, app)
		return
	}
	if decl.Target == annotation.TargetStruct || decl.Target == annotation.TargetType {
		a.discoverComponent(decl, app)
	}
}

// handleAppRoot records the @Application declaration and enforces that exactly
// one exists.
func (a *analysis) handleAppRoot(decl *Declaration, app *model.Application) {
	a.appCount++
	if a.appCount > 1 {
		a.diags = append(a.diags, diagErr(CodeApplicationRoot, decl.Pos,
			"multiple @Application declarations found; only one is allowed"))
		return
	}
	if name, ok := stringArg(decl, "Application", "name"); ok {
		app.Name = name
	}
	app.RootPackage = decl.PkgPath
}

// checkAppRoot warns when no application root was declared.
func (a *analysis) checkAppRoot() {
	if a.appCount == 0 {
		a.diags = append(a.diags, &annotation.Diagnostic{
			Severity: annotation.SeverityWarning,
			Code:     CodeApplicationRoot,
			Message:  "no @Application declaration found",
		})
	}
}

// discoverComponent builds a component from a struct/type declaration.
func (a *analysis) discoverComponent(decl *Declaration, app *model.Application) {
	kind, ok := componentKindOf(decl)
	if !ok || decl.TypeName == nil {
		return
	}
	if kind == model.ComponentConfigProperties {
		a.discoverConfigProperties(decl, app)
		return
	}
	fset := decl.Pkg.Fset

	var ctor *model.Constructor
	if fn := lookupConstructor(decl.TypeName); fn != nil {
		c, ds := buildConstructor(fn, false, fset)
		a.diags = append(a.diags, ds...)
		if c == nil {
			return
		}
		ctor = c
	} else if cl := constructorlessFor(decl.TypeName, fset); cl != nil {
		ctor = cl
	} else {
		a.diags = append(a.diags, diagErr(CodeMissingConstructor, decl.Pos,
			"component %s has required fields but no constructor New%s; add one so it can be injected",
			decl.TypeName.Name(), decl.TypeName.Name()))
		return
	}

	name := decl.TypeName.Name()
	if explicit, ok := componentName(decl); ok {
		name = explicit
	}
	app.Components = append(app.Components, &model.Component{
		ID:           model.NewComponentID(decl.PkgPath, decl.TypeName.Name()),
		Name:         name,
		PackagePath:  decl.PkgPath,
		ProvidedType: ctor.ReturnType,
		Named:        namedOf(ctor.ReturnType),
		Kind:         kind,
		Scope:        scopeOf(decl),
		Primary:      decl.Has("Primary"),
		Constructor:  ctor,
		Dependencies: append([]model.Dependency(nil), ctor.Params...),
		Position:     decl.Pos,
	})
}

// discoverBean builds a component from an @Bean provider function.
func (a *analysis) discoverBean(decl *Declaration, app *model.Application) {
	if decl.Func == nil {
		return
	}
	ctor, ds := buildConstructor(decl.Func, true, decl.Pkg.Fset)
	a.diags = append(a.diags, ds...)
	if ctor == nil {
		return
	}
	beanName, _ := stringArg(decl, "Bean", "name")
	name := beanName
	if name == "" {
		name = decl.Func.Name()
	}
	app.Components = append(app.Components, &model.Component{
		ID:           model.NewBeanID(decl.PkgPath, decl.Func.Name(), beanName),
		Name:         name,
		PackagePath:  decl.PkgPath,
		ProvidedType: ctor.ReturnType,
		Named:        namedOf(ctor.ReturnType),
		Kind:         model.ComponentBean,
		Scope:        model.ScopeSingleton,
		Primary:      decl.Has("Primary"),
		Constructor:  ctor,
		Dependencies: append([]model.Dependency(nil), ctor.Params...),
		Position:     decl.Pos,
	})
}

// componentKindOf returns the component kind for a declaration's highest
// priority component annotation. Interface targets are never components (they
// are injection targets satisfied by implementations).
func componentKindOf(decl *Declaration) (model.ComponentKind, bool) {
	if decl.Target == annotation.TargetInterface {
		return 0, false
	}
	for _, ck := range componentKind {
		if decl.Has(ck.annotation) {
			return ck.kind, true
		}
	}
	return 0, false
}

// componentName returns an explicit component name from whichever naming
// annotation is present.
func componentName(decl *Declaration) (string, bool) {
	for _, ck := range componentKind {
		if s, ok := stringArg(decl, ck.annotation, "name"); ok {
			return s, true
		}
	}
	return "", false
}

// scopeOf resolves a component's scope from @Service(scope=...) or @Scope.
func scopeOf(decl *Declaration) model.Scope {
	if s, ok := stringArg(decl, "Service", "scope"); ok && s == "prototype" {
		return model.ScopePrototype
	}
	if ann, ok := decl.Find("Scope"); ok {
		if v, ok := ann.Positional(); ok {
			if s, ok := annotation.AsString(v); ok && s == "prototype" {
				return model.ScopePrototype
			}
		}
	}
	return model.ScopeSingleton
}

// stringArg extracts a string-valued named argument from a specific annotation.
func stringArg(decl *Declaration, annName, argName string) (string, bool) {
	ann, ok := decl.Find(annName)
	if !ok {
		return "", false
	}
	v, ok := ann.Arg(argName)
	if !ok {
		return "", false
	}
	return annotation.AsString(v)
}

// cycleDiagnostic renders a dependency cycle into a readable diagnostic (§15.1).
func cycleDiagnostic(cyc *graph.Cycle, app *model.Application) *annotation.Diagnostic {
	names := make([]string, 0, len(cyc.Path)+1)
	for _, id := range cyc.Path {
		names = append(names, componentDisplay(app, id))
	}
	if len(cyc.Path) > 0 {
		names = append(names, componentDisplay(app, cyc.Path[0]))
	}
	var pos = app.Components[0].Position
	if c := app.ComponentByID(cyc.Path[0]); c != nil {
		pos = c.Position
	}
	return diagErr(CodeDependencyCycle, pos,
		"dependency cycle detected:\n  %s", strings.Join(names, "\n  -> "))
}

// componentDisplay returns a component's name for diagnostics, falling back to
// its ID.
func componentDisplay(app *model.Application, id model.ComponentID) string {
	if c := app.ComponentByID(id); c != nil {
		return c.Name
	}
	return string(id)
}
