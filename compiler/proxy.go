package compiler

import (
	"go/token"
	"go/types"
	"time"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Service-proxy diagnostic codes (GOBPRX family, §39.4).
const (
	// CodeConcreteInjection is a consumer injecting a proxied service by its
	// concrete type instead of its interface (§24.3).
	CodeConcreteInjection = "GOBPRX001"
	// CodeMissingInterface is an intercepted service that does not declare the
	// interface it is exposed as.
	CodeMissingInterface = "GOBPRX002"
	// CodeInterfaceNotImplemented is a service that does not implement its
	// declared interface.
	CodeInterfaceNotImplemented = "GOBPRX003"
	// CodeInvalidInterceptedMethod is an intercepted method whose signature is
	// unsupported.
	CodeInvalidInterceptedMethod = "GOBPRX004"
	// CodeUnknownInterface is an @Service(implements=...) name that is not a
	// known interface.
	CodeUnknownInterface = "GOBPRX005"
)

// interceptAnnotations are the method annotations that trigger proxying.
var interceptAnnotations = []string{"Transactional", "Traced", "Timed", "Timeout", "Retry", "Authorize", "RolesAllowed", "Logged", "Audit"}

// resolveInterface looks up the interface named by @Service(implements=...) in
// the service's package and returns its type.
func resolveInterface(tn *types.TypeName, name string, pos token.Position) (types.Type, *annotation.Diagnostic) {
	if tn.Pkg() == nil {
		return nil, diagErr(CodeUnknownInterface, pos, "cannot resolve interface %q", name)
	}
	obj := tn.Pkg().Scope().Lookup(name)
	itn, ok := obj.(*types.TypeName)
	if !ok {
		return nil, diagErr(CodeUnknownInterface, pos,
			"@Service(implements=%q) does not name a type in package %s", name, tn.Pkg().Name())
	}
	if _, ok := itn.Type().Underlying().(*types.Interface); !ok {
		return nil, diagErr(CodeUnknownInterface, pos,
			"@Service(implements=%q) is not an interface", name)
	}
	return itn.Type(), nil
}

// discoverProxies attaches interception metadata to services, validates them,
// and synthesizes a proxy component for each intercepted service (§24).
func (a *analysis) discoverProxies(scan *ScanResult, app *model.Application) {
	byType := make(map[string]*model.Component, len(app.Components))
	for _, c := range app.Components {
		byType[string(c.ID)] = c
	}

	// Collect intercepted methods per component from method declarations.
	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		if !hasAny(decl, interceptAnnotations) {
			continue
		}
		comp := byType[typeKey(decl.PkgPath, decl.Recv.Name())]
		if comp == nil {
			continue
		}
		method, ok := a.interceptedMethod(decl)
		if ok {
			comp.Intercepted = append(comp.Intercepted, method)
		}
	}

	// Synthesize proxies for services that ended up with intercepted methods.
	var proxies []*model.Component
	for _, c := range app.Components {
		if len(c.Intercepted) == 0 {
			continue
		}
		if proxy := a.buildProxy(c); proxy != nil {
			proxies = append(proxies, proxy)
		}
	}
	app.Components = append(app.Components, proxies...)
}

// buildProxy validates an intercepted service and returns its proxy component,
// or nil (with diagnostics) when the service cannot be proxied.
func (a *analysis) buildProxy(target *model.Component) *model.Component {
	if target.Interface == nil {
		a.diags = append(a.diags, diagErr(CodeMissingInterface, target.Position,
			"service %s has intercepted methods but declares no interface; add @Service(implements=\"...\")",
			target.Name))
		return nil
	}
	iface, _ := target.Interface.Underlying().(*types.Interface)
	if iface != nil && target.ProvidedType != nil && !types.Implements(target.ProvidedType, iface) {
		a.diags = append(a.diags, diagErr(CodeInterfaceNotImplemented, target.Position,
			"service %s does not implement its declared interface %s",
			target.Name, typeString(target.Interface)))
		return nil
	}

	target.Proxied = true
	proxyName := target.Named.Obj().Name() + "Proxy"

	return &model.Component{
		ID:           model.ComponentID(string(target.ID) + "$proxy"),
		Name:         proxyName,
		PackagePath:  target.PackagePath,
		ProvidedType: target.Interface,
		Named:        namedOf(target.Interface),
		Kind:         model.ComponentProxy,
		Scope:        model.ScopeSingleton,
		Primary:      target.Primary,
		Interface:    target.Interface,
		Intercepted:  target.Intercepted,
		ProxyTarget:  target.ID,
		Constructor: &model.Constructor{
			PackagePath: target.PackagePath,
			FuncName:    "New" + proxyName,
			ReturnType:  target.Interface,
		},
		// The proxy's single dependency is its target, pre-resolved so the
		// resolver does not treat it as a normal (and now excluded) candidate.
		Dependencies: []model.Dependency{{
			Name:       "target",
			Type:       target.ProvidedType,
			ResolvedTo: target.ID,
			Position:   target.Position,
		}},
		Position: target.Position,
	}
}

// interceptedMethod builds an InterceptedMethod from a method declaration,
// validating that its signature supports interception (§36.2, §26).
func (a *analysis) interceptedMethod(decl *Declaration) (model.InterceptedMethod, bool) {
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return model.InterceptedMethod{}, false
	}
	params := sig.Params()
	if params.Len() == 0 || !isContextType(params.At(0).Type()) {
		a.diags = append(a.diags, diagErr(CodeInvalidInterceptedMethod, decl.Pos,
			"intercepted method %s must take context.Context as its first parameter", decl.Name))
		return model.InterceptedMethod{}, false
	}
	results := sig.Results()
	if results.Len() == 0 || !isErrorType(results.At(results.Len()-1).Type()) {
		a.diags = append(a.diags, diagErr(CodeInvalidInterceptedMethod, decl.Pos,
			"intercepted method %s must return error as its last result", decl.Name))
		return model.InterceptedMethod{}, false
	}

	m := model.InterceptedMethod{Name: decl.Name}
	if ann, ok := decl.Find("Traced"); ok {
		m.Traced = true
		if s, ok := stringArgValue(ann, "name"); ok {
			m.TraceName = s
		}
	}
	if ann, ok := decl.Find("Timed"); ok {
		m.Timed = true
		if s, ok := stringArgValue(ann, "name"); ok {
			m.MetricName = s
		}
	}
	if ann, ok := decl.Find("Transactional"); ok {
		m.Transactional = true
		m.Tx = txOptions(ann)
	}
	if ann, ok := decl.Find("Timeout"); ok {
		if v, ok := ann.Positional(); ok {
			if s, ok := annotation.AsString(v); ok {
				if d, err := time.ParseDuration(s); err == nil {
					m.Timeout = d
				}
			}
		}
	}
	if ann, ok := decl.Find("Retry"); ok {
		m.Retry = retryPolicy(ann)
	}
	m.Authorize = authorizeSpec(decl)
	if ann, ok := decl.Find("Logged"); ok {
		m.Logged = true
		m.LogLevel = "info"
		if s, ok := stringArgValue(ann, "level"); ok {
			m.LogLevel = s
		}
	}
	if ann, ok := decl.Find("Audit"); ok {
		spec := &model.AuditSpec{}
		if s, ok := stringArgValue(ann, "action"); ok {
			spec.Action = s
		}
		if s, ok := stringArgValue(ann, "resource"); ok {
			spec.Resource = s
		}
		m.Audit = spec
	}
	return m, true
}

// authorizeSpec reads @Authorize/@RolesAllowed into an AuthorizeSpec, or nil.
func authorizeSpec(decl *Declaration) *model.AuthorizeSpec {
	spec := &model.AuthorizeSpec{}
	found := false
	if ann, ok := decl.Find("Authorize"); ok {
		found = true
		spec.Roles = arrayArg(ann, "roles")
		spec.Permissions = arrayArg(ann, "permissions")
		if s, ok := stringArgValue(ann, "mode"); ok {
			spec.Mode = s
		}
	}
	if ann, ok := decl.Find("RolesAllowed"); ok {
		found = true
		if v, ok := ann.Positional(); ok {
			spec.Roles = append(spec.Roles, stringList(v)...)
		}
	}
	if !found {
		return nil
	}
	return spec
}

// arrayArg extracts a []string from an annotation array argument.
func arrayArg(ann annotation.Annotation, name string) []string {
	v, ok := ann.Arg(name)
	if !ok {
		return nil
	}
	return stringList(v)
}

// retryPolicy reads @Retry arguments into a model.RetryPolicy.
func retryPolicy(ann annotation.Annotation) *model.RetryPolicy {
	p := &model.RetryPolicy{MaxAttempts: 3}
	if v, ok := ann.Arg("maxAttempts"); ok {
		if iv, ok := v.(annotation.IntValue); ok {
			p.MaxAttempts = int(iv.Val)
		}
	}
	if s, ok := stringArgValue(ann, "delay"); ok {
		if d, err := time.ParseDuration(s); err == nil {
			p.Delay = d
		}
	}
	if v, ok := ann.Arg("multiplier"); ok {
		switch t := v.(type) {
		case annotation.FloatValue:
			p.Multiplier = t.Val
		case annotation.IntValue:
			p.Multiplier = float64(t.Val)
		}
	}
	if s, ok := stringArgValue(ann, "maxDelay"); ok {
		if d, err := time.ParseDuration(s); err == nil {
			p.MaxDelay = d
		}
	}
	return p
}

// txOptions reads @Transactional arguments into model.TxOptions.
func txOptions(ann annotation.Annotation) model.TxOptions {
	var tx model.TxOptions
	if v, ok := ann.Arg("readOnly"); ok {
		if b, ok := v.(annotation.BoolValue); ok {
			tx.ReadOnly = b.Val
		}
	}
	if s, ok := stringArgValue(ann, "isolation"); ok {
		tx.Isolation = s
	}
	if s, ok := stringArgValue(ann, "propagation"); ok {
		tx.Propagation = s
	}
	if s, ok := stringArgValue(ann, "timeout"); ok {
		if d, err := time.ParseDuration(s); err == nil {
			tx.Timeout = d
		}
	}
	return tx
}

// stringArgValue extracts a string/identifier argument from an annotation.
func stringArgValue(ann annotation.Annotation, name string) (string, bool) {
	v, ok := ann.Arg(name)
	if !ok {
		return "", false
	}
	return annotation.AsString(v)
}

// hasAny reports whether a declaration carries any of the given annotations.
func hasAny(decl *Declaration, names []string) bool {
	for _, n := range names {
		if decl.Has(n) {
			return true
		}
	}
	return false
}
