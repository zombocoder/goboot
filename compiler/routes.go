package compiler

import (
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// HTTP controller diagnostic codes (GOBHTTP family, §39.4).
const (
	// CodeInvalidHandler is a controller method with an unsupported signature.
	CodeInvalidHandler = "GOBHTTP001"
	// CodeDuplicateRoute is two handlers mapped to the same method and path.
	CodeDuplicateRoute = "GOBHTTP004"
)

// httpMapping describes a method-mapping annotation.
type httpMapping struct {
	annotation    string
	method        string
	defaultStatus int
}

var httpMappings = []httpMapping{
	{"GetMapping", "GET", 200},
	{"PostMapping", "POST", 201},
	{"PutMapping", "PUT", 200},
	{"PatchMapping", "PATCH", 200},
	{"DeleteMapping", "DELETE", 204},
}

// discoverRoutes finds controllers and their route methods, validates handler
// signatures, assembles routes, and reports duplicate routes (§17, §33). It
// populates app.Controllers and app.Routes.
func (a *analysis) discoverRoutes(scan *ScanResult, app *model.Application) {
	controllers := a.indexControllers(scan, app)

	seen := map[string]token.Position{}
	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		mapping, ok := mappingFor(decl)
		if !ok {
			continue
		}
		ctrl := controllers[typeKey(decl.PkgPath, decl.Recv.Name())]
		if ctrl == nil {
			continue // route method on a non-controller type; ignore
		}
		route := a.buildRoute(decl, ctrl, mapping)
		if route == nil {
			continue
		}
		key := route.Method + " " + route.Pattern
		if prev, dup := seen[key]; dup {
			a.diags = append(a.diags, diagErr(CodeDuplicateRoute, route.Position,
				"duplicate route %s %s (previously declared at %s)",
				route.Method, route.Pattern, prev))
			continue
		}
		seen[key] = route.Position
		ctrl.Routes = append(ctrl.Routes, route)
	}

	finalizeControllers(controllers, app)
}

// indexControllers builds controllers for every discovered @RestController
// component, keyed by their type.
func (a *analysis) indexControllers(scan *ScanResult, app *model.Application) map[string]*model.Controller {
	controllers := map[string]*model.Controller{}
	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetStruct || !decl.Has("RestController") || decl.TypeName == nil {
			continue
		}
		comp := app.ComponentByID(model.NewComponentID(decl.PkgPath, decl.TypeName.Name()))
		if comp == nil {
			continue // component discovery failed for this controller
		}
		base, _ := stringArg(decl, "RequestMapping", "path")
		controllers[typeKey(decl.PkgPath, decl.TypeName.Name())] = &model.Controller{
			Component: comp,
			BasePath:  base,
		}
	}
	return controllers
}

// buildRoute validates a handler method and assembles its route, or returns nil
// with a diagnostic on an invalid signature.
func (a *analysis) buildRoute(decl *Declaration, ctrl *model.Controller, mapping httpMapping) *model.Route {
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return nil
	}
	reqType, reqPtr, respType, err := validateHandler(sig)
	if err != "" {
		a.diags = append(a.diags, diagErr(CodeInvalidHandler, decl.Pos,
			"controller method %s has an unsupported signature: %s", decl.Name, err))
		return nil
	}

	subPath, _ := stringArg(decl, mapping.annotation, "path")
	return &model.Route{
		Method:         mapping.method,
		Pattern:        joinPath(ctrl.BasePath, subPath),
		Controller:     ctrl.Component.ID,
		HandlerName:    decl.Name,
		RequestType:    reqType,
		RequestPointer: reqPtr,
		ResponseType:   respType,
		SuccessStatus:  successStatus(decl, mapping),
		Consumes:       mediaTypes(decl, mapping.annotation, "consumes", "Consumes"),
		Produces:       mediaTypes(decl, mapping.annotation, "produces", "Produces"),
		Authorize:      authorizeRoles(decl),
		Position:       decl.Pos,
	}
}

// validateHandler checks a controller method signature against the supported
// forms (§17.3) and extracts request and response types. It returns a non-empty
// reason string when the signature is invalid.
func validateHandler(sig *types.Signature) (reqType types.Type, reqPtr bool, respType types.Type, reason string) {
	params := sig.Params()
	if params.Len() < 1 || !isContextType(params.At(0).Type()) {
		return nil, false, nil, "first parameter must be context.Context"
	}
	if params.Len() > 2 {
		return nil, false, nil, "expected at most (context.Context, request)"
	}
	if params.Len() == 2 {
		reqType = params.At(1).Type()
		if ptr, ok := reqType.(*types.Pointer); ok {
			reqPtr = true
			reqType = ptr.Elem()
		}
	}

	results := sig.Results()
	if results.Len() == 0 || !isErrorType(results.At(results.Len()-1).Type()) {
		return nil, false, nil, "last return value must be error"
	}
	switch results.Len() {
	case 1: // (error)
	case 2: // (response, error)
		respType = results.At(0).Type()
	default:
		return nil, false, nil, "expected (response, error) or (error)"
	}
	return reqType, reqPtr, respType, ""
}

// successStatus resolves a route's success status: an explicit status argument
// on the mapping, else the smallest 2xx from @Response annotations, else the
// method default (§18.4).
func successStatus(decl *Declaration, mapping httpMapping) int {
	if s, ok := intArg(decl, mapping.annotation, "status"); ok {
		return s
	}
	best := 0
	for _, resp := range decl.FindAll("Response") {
		if v, ok := resp.Arg("status"); ok {
			if iv, ok := v.(annotation.IntValue); ok {
				s := int(iv.Val)
				if s >= 200 && s < 300 && (best == 0 || s < best) {
					best = s
				}
			}
		}
	}
	if best != 0 {
		return best
	}
	return mapping.defaultStatus
}

// authorizeRoles extracts the roles from an @Authorize annotation, if present.
func authorizeRoles(decl *Declaration) []string {
	ann, ok := decl.Find("Authorize")
	if !ok {
		return nil
	}
	v, ok := ann.Arg("roles")
	if !ok {
		return nil
	}
	arr, ok := v.(annotation.ArrayValue)
	if !ok {
		return nil
	}
	var roles []string
	for _, e := range arr.Elements {
		if s, ok := annotation.AsString(e); ok {
			roles = append(roles, s)
		}
	}
	return roles
}

// mediaTypes gathers a route's media-type constraint from both the mapping
// argument (e.g. @PostMapping(consumes=[...])) and the standalone marker
// (@Consumes([...])), preserving order and de-duplicating.
func mediaTypes(decl *Declaration, mappingAnn, argName, standaloneAnn string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(items []string) {
		for _, s := range items {
			if s != "" && !seen[s] {
				seen[s] = true
				out = append(out, s)
			}
		}
	}
	if ann, ok := decl.Find(mappingAnn); ok {
		add(arrayArg(ann, argName))
	}
	if ann, ok := decl.Find(standaloneAnn); ok {
		if v, ok := ann.Positional(); ok {
			add(stringList(v))
		}
	}
	return out
}

// finalizeControllers sorts controllers and their routes deterministically and
// records them on the application.
func finalizeControllers(controllers map[string]*model.Controller, app *model.Application) {
	list := make([]*model.Controller, 0, len(controllers))
	for _, c := range controllers {
		sort.Slice(c.Routes, func(i, j int) bool {
			if c.Routes[i].Pattern != c.Routes[j].Pattern {
				return c.Routes[i].Pattern < c.Routes[j].Pattern
			}
			return c.Routes[i].Method < c.Routes[j].Method
		})
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Component.ID < list[j].Component.ID })

	app.Controllers = list
	for _, c := range list {
		app.Routes = append(app.Routes, c.Routes...)
	}
	sort.Slice(app.Routes, func(i, j int) bool {
		if app.Routes[i].Pattern != app.Routes[j].Pattern {
			return app.Routes[i].Pattern < app.Routes[j].Pattern
		}
		return app.Routes[i].Method < app.Routes[j].Method
	})
}

// mappingFor returns the HTTP mapping for a method declaration, if any.
func mappingFor(decl *Declaration) (httpMapping, bool) {
	for _, m := range httpMappings {
		if decl.Has(m.annotation) {
			return m, true
		}
	}
	return httpMapping{}, false
}

// intArg extracts an integer-valued named argument from an annotation.
func intArg(decl *Declaration, annName, argName string) (int, bool) {
	ann, ok := decl.Find(annName)
	if !ok {
		return 0, false
	}
	v, ok := ann.Arg(argName)
	if !ok {
		return 0, false
	}
	iv, ok := v.(annotation.IntValue)
	if !ok {
		return 0, false
	}
	return int(iv.Val), true
}

// isContextType reports whether t is context.Context.
func isContextType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Name() == "Context" && obj.Pkg() != nil && obj.Pkg().Path() == "context"
}

// joinPath joins a controller base path and a method sub-path into a single
// pattern with a leading slash.
func joinPath(base, sub string) string {
	base = strings.TrimRight(base, "/")
	sub = strings.TrimLeft(sub, "/")
	var pattern string
	switch {
	case sub == "":
		pattern = base
	case base == "":
		pattern = "/" + sub
	default:
		pattern = base + "/" + sub
	}
	if pattern == "" || pattern[0] != '/' {
		pattern = "/" + pattern
	}
	return pattern
}

// typeKey is the map key identifying a controller type.
func typeKey(pkgPath, name string) string { return pkgPath + ":" + name }
