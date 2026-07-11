package compiler

import (
	"go/types"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Configuration and lifecycle diagnostic codes (GOBCFG/GOBLIF families, §39.4).
const (
	// CodeInvalidLifecycle is a lifecycle method with an unsupported signature.
	CodeInvalidLifecycle = "GOBLIF001"
)

// discoverConfigProperties creates a component for an @ConfigurationProperties
// struct. Unlike ordinary components it has no NewXxx constructor; a synthetic
// config-loader constructor records the prefix and drives generation of a typed
// loader that binds the struct from the application's configuration source
// (§28).
func (a *analysis) discoverConfigProperties(decl *Declaration, app *model.Application) {
	tn := decl.TypeName
	if _, ok := tn.Type().Underlying().(*types.Struct); !ok {
		return
	}
	prefix, _ := stringArg(decl, "ConfigurationProperties", "prefix")
	provided := tn.Type() // bound and injected by value

	app.Components = append(app.Components, &model.Component{
		ID:           model.NewComponentID(decl.PkgPath, tn.Name()),
		Name:         tn.Name(),
		PackagePath:  decl.PkgPath,
		ProvidedType: provided,
		Named:        namedOf(provided),
		Kind:         model.ComponentConfigProperties,
		Scope:        model.ScopeSingleton,
		ConfigPrefix: prefix,
		Constructor: &model.Constructor{
			PackagePath:  decl.PkgPath,
			PackageName:  tn.Pkg().Name(),
			FuncName:     "Load" + tn.Name(),
			ReturnType:   provided,
			ReturnsError: true,
			ConfigLoader: true,
		},
		Conditions: extractConditions(decl),
		Position:   decl.Pos,
	})
}

// discoverLifecycle attaches @PostConstruct and @PreDestroy hooks to their
// owning components (§30.1), validating each method's signature.
func (a *analysis) discoverLifecycle(scan *ScanResult, app *model.Application) {
	byType := make(map[string]*model.Component, len(app.Components))
	for _, c := range app.Components {
		byType[string(c.ID)] = c
	}

	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		isPost := decl.Has("PostConstruct")
		isPre := decl.Has("PreDestroy")
		if !isPost && !isPre {
			continue
		}
		comp := byType[typeKey(decl.PkgPath, decl.Recv.Name())]
		if comp == nil {
			continue
		}
		hook, ok := a.lifecycleHook(decl)
		if !ok {
			continue
		}
		if isPost {
			comp.PostConstruct = hook
		}
		if isPre {
			comp.PreDestroy = hook
		}
	}
}

// lifecycleHook validates a lifecycle method signature (§30.2) and returns its
// descriptor. Supported forms: func(), func() error, func(context.Context), and
// func(context.Context) error.
func (a *analysis) lifecycleHook(decl *Declaration) (*model.LifecycleMethod, bool) {
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return nil, false
	}
	params := sig.Params()
	takesContext := false
	switch {
	case params.Len() == 0:
	case params.Len() == 1 && isContextType(params.At(0).Type()):
		takesContext = true
	default:
		a.diags = append(a.diags, diagErr(CodeInvalidLifecycle, decl.Pos,
			"lifecycle method %s must take no parameters or a single context.Context", decl.Name))
		return nil, false
	}

	results := sig.Results()
	returnsError := false
	switch {
	case results.Len() == 0:
	case results.Len() == 1 && isErrorType(results.At(0).Type()):
		returnsError = true
	default:
		a.diags = append(a.diags, diagErr(CodeInvalidLifecycle, decl.Pos,
			"lifecycle method %s must return nothing or a single error", decl.Name))
		return nil, false
	}

	return &model.LifecycleMethod{
		MethodName:   decl.Name,
		TakesContext: takesContext,
		ReturnsError: returnsError,
	}, true
}
