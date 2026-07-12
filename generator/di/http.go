package di

import (
	"fmt"
	"go/types"
	"sort"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// runtimePath is the import path of the goboot runtime package the generated
// handlers depend on.
const runtimePath = "github.com/zombocoder/goboot/runtime"

// renderHTTP emits the HTTP handler proxies and the RegisterRoutes function for
// every route in the application (§21). It returns the empty string when the
// application declares no routes. byID maps component IDs to their bindings so
// handlers can reference the controller fields on the Components struct.
func renderHTTP(app *model.Application, byID map[model.ComponentID]*binding, im *imports) string {
	if len(app.Routes) == 0 {
		return ""
	}
	// Register the packages the generated handlers always use.
	rt := func(sym string) string { return im.qualify(runtimePath, "runtime", sym) }
	httpPkg := func(sym string) string { return im.qualify("net/http", "http", sym) }

	var b strings.Builder
	for _, route := range app.Routes {
		bd := byID[route.Controller]
		if bd == nil {
			continue
		}
		b.WriteString(renderHandler(route, bd, im, rt, httpPkg))
		b.WriteString("\n")
	}
	handlers := collectAdviceHandlers(app, byID)
	if len(handlers) > 0 {
		b.WriteString(renderExceptionDispatcher(handlers, im, rt, httpPkg))
		b.WriteString("\n")
	}
	b.WriteString(renderRegister(app, byID, im, rt, httpPkg, len(handlers) > 0))
	return b.String()
}

// adviceHandlerRef binds an @ExceptionHandler to the Components field holding
// the advice instance that owns it.
type adviceHandlerRef struct {
	field   string
	handler model.ExceptionHandler
}

// collectAdviceHandlers gathers every @ExceptionHandler across advice
// components, ordered so concrete handlers are tried before catch-alls and the
// result is deterministic.
func collectAdviceHandlers(app *model.Application, byID map[model.ComponentID]*binding) []adviceHandlerRef {
	var refs []adviceHandlerRef
	for _, c := range app.Components {
		if c.Kind != model.ComponentAdvice {
			continue
		}
		bd := byID[c.ID]
		if bd == nil {
			continue
		}
		for _, h := range c.ExceptionHandlers {
			refs = append(refs, adviceHandlerRef{field: bd.field, handler: h})
		}
	}
	sort.SliceStable(refs, func(i, j int) bool {
		hi, hj := refs[i].handler, refs[j].handler
		if hi.CatchAll != hj.CatchAll {
			return !hi.CatchAll // concrete handlers first
		}
		ki := types.TypeString(hi.ErrType, nil)
		kj := types.TypeString(hj.ErrType, nil)
		if ki != kj {
			return ki < kj
		}
		if refs[i].field != refs[j].field {
			return refs[i].field < refs[j].field
		}
		return hi.MethodName < hj.MethodName
	})
	return refs
}

// renderExceptionDispatcher emits an ErrorHandler that routes controller errors
// to the matching @ExceptionHandler method, falling back to the delegate (§20).
func renderExceptionDispatcher(handlers []adviceHandlerRef, im *imports, rt, httpPkg func(string) string) string {
	ctxType := im.qualify("context", "context", "Context")
	errorsAs := im.qualify("errors", "errors", "As")

	var b strings.Builder
	b.WriteString("// exceptionDispatcher routes controller errors to @ExceptionHandler methods,\n")
	b.WriteString("// falling back to the delegate ErrorHandler when none match.\n")
	b.WriteString("type exceptionDispatcher struct {\n")
	b.WriteString("\tcomponents *Components\n")
	fmt.Fprintf(&b, "\tdelegate   %s\n", rt("ErrorHandler"))
	fmt.Fprintf(&b, "\twriter     %s\n", rt("ResponseWriter"))
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "// newExceptionDispatcher wraps deps.ErrorHandler with advice dispatch.\n")
	fmt.Fprintf(&b, "func newExceptionDispatcher(components *Components, deps %s) *exceptionDispatcher {\n", rt("HTTPHandlerDependencies"))
	b.WriteString("\treturn &exceptionDispatcher{components: components, delegate: deps.ErrorHandler, writer: deps.ResponseWriter}\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func (d *exceptionDispatcher) Handle(ctx %s, w %s, r *%s, err error) {\n",
		ctxType, httpPkg("ResponseWriter"), httpPkg("Request"))

	i := 0
	for _, ref := range handlers {
		if ref.handler.CatchAll {
			continue // catch-alls are emitted as the fallback below
		}
		call := fmt.Sprintf("d.components.%s.%s", ref.field, ref.handler.MethodName)
		errVar := fmt.Sprintf("e%d", i)
		i++
		fmt.Fprintf(&b, "\tvar %s %s\n", errVar, renderType(ref.handler.ErrType, im))
		fmt.Fprintf(&b, "\tif %s(err, &%s) {\n", errorsAs, errVar)
		b.WriteString(renderHandlerBody(ref.handler, call, ctxType, fmt.Sprintf("ctx, %s", errVar)))
		b.WriteString("\t\treturn\n\t}\n")
	}

	// Fallback: the first catch-all handler, or the delegate.
	if fb, ok := firstCatchAll(handlers); ok {
		call := fmt.Sprintf("d.components.%s.%s", fb.field, fb.handler.MethodName)
		b.WriteString(renderHandlerBody(fb.handler, call, ctxType, "ctx, err"))
	} else {
		b.WriteString("\td.delegate.Handle(ctx, w, r, err)\n")
	}
	b.WriteString("}\n")
	return b.String()
}

// renderHandlerBody emits the body that invokes an advice handler and writes its
// result: for the response form it writes the body with the handler status; for
// the transform form it passes the returned error to the delegate.
func renderHandlerBody(h model.ExceptionHandler, call, ctxType, args string) string {
	var b strings.Builder
	if h.ResponseType != nil {
		fmt.Fprintf(&b, "\t\tresponse, herr := %s(%s)\n", call, args)
		b.WriteString("\t\tif herr != nil {\n\t\t\td.delegate.Handle(ctx, w, r, herr)\n\t\t\treturn\n\t\t}\n")
		fmt.Fprintf(&b, "\t\tif werr := d.writer.Write(ctx, w, %d, response); werr != nil {\n", h.SuccessStatus)
		b.WriteString("\t\t\td.delegate.Handle(ctx, w, r, werr)\n\t\t}\n")
	} else {
		fmt.Fprintf(&b, "\t\td.delegate.Handle(ctx, w, r, %s(%s))\n", call, args)
	}
	return b.String()
}

// firstCatchAll returns the first catch-all handler in the ordered set.
func firstCatchAll(handlers []adviceHandlerRef) (adviceHandlerRef, bool) {
	for _, ref := range handlers {
		if ref.handler.CatchAll {
			return ref, true
		}
	}
	return adviceHandlerRef{}, false
}

// registerRouteImports pre-registers the packages referenced by route request
// and response types, plus the runtime and net/http packages, so that local
// variable names allocated later avoid every import alias.
func registerRouteImports(app *model.Application, im *imports) {
	if len(app.Routes) == 0 {
		return
	}
	im.add(runtimePath, "runtime")
	im.add("net/http", "http")
	for _, route := range app.Routes {
		if route.RequestType != nil {
			renderType(route.RequestType, im)
		}
		if route.ResponseType != nil {
			renderType(route.ResponseType, im)
		}
	}
}

// renderHandler emits a single handler factory function for a route. The
// controller parameter uses the component's collision-safe local name so it
// never shadows an imported package (e.g. a parameter named "controller" would
// shadow the controller package that request types live in).
func renderHandler(route *model.Route, ctrl *binding, im *imports, rt, httpPkg func(string) string) string {
	name := handlerFuncName(ctrl.field, route.HandlerName)
	ctrlType := renderType(ctrl.comp.ProvidedType, im)
	recv := ctrl.local

	var b strings.Builder
	fmt.Fprintf(&b, "// %s handles %s %s.\n", name, route.Method, route.Pattern)
	fmt.Fprintf(&b, "func %s(%s %s, deps %s) %s {\n", name, recv, ctrlType, rt("HTTPHandlerDependencies"), httpPkg("HandlerFunc"))
	fmt.Fprintf(&b, "\treturn func(w %s, r *%s) {\n", httpPkg("ResponseWriter"), httpPkg("Request"))
	b.WriteString("\t\tctx := r.Context()\n")
	fmt.Fprintf(&b, "\t\tdefer %s(ctx, w, r, deps.ErrorHandler)\n\n", rt("Recover"))

	// Content negotiation runs first so an unacceptable media type fails fast
	// before binding (§19).
	if len(route.Consumes) > 0 || len(route.Produces) > 0 {
		fmt.Fprintf(&b, "\t\tif err := %s(r, %s, %s); err != nil {\n",
			rt("NegotiateContent"), stringSliceLit(route.Consumes), stringSliceLit(route.Produces))
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
	}

	// Authenticate and authorize before binding: a secured route rejects an
	// unauthenticated (401) or unauthorized (403) caller before the body is read.
	// The established principal is placed on ctx so downstream authorization
	// (e.g. service-proxy @Authorize) and the controller can read it.
	if len(route.Authorize) > 0 {
		b.WriteString("\t\tprincipal, err := deps.Authenticator.Authenticate(ctx, r)\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
		fmt.Fprintf(&b, "\t\tctx = %s(ctx, principal)\n", rt("WithPrincipal"))
		fmt.Fprintf(&b, "\t\tif err := deps.Authorizer.Authorize(ctx, %s{\n", rt("AuthorizationRequest"))
		fmt.Fprintf(&b, "\t\t\tRoles: %s,\n", stringSliceLit(route.Authorize))
		fmt.Fprintf(&b, "\t\t\tMode:  %s,\n", rt("AuthorizationModeAny"))
		b.WriteString("\t\t}); err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
	}

	callArgs := []string{"ctx"}
	if route.HasRequest() {
		reqType := renderType(route.RequestType, im)
		fmt.Fprintf(&b, "\t\tvar request %s\n", reqType)
		b.WriteString("\t\tif err := deps.Binder.Bind(ctx, r, &request); err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tif err := deps.Validator.Validate(ctx, request); err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
		if route.RequestPointer {
			callArgs = append(callArgs, "&request")
		} else {
			callArgs = append(callArgs, "request")
		}
	}

	call := fmt.Sprintf("%s.%s(%s)", recv, route.HandlerName, strings.Join(callArgs, ", "))
	status := strconv.Itoa(route.SuccessStatus)
	if route.HasResponse() {
		fmt.Fprintf(&b, "\t\tresponse, err := %s\n", call)
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
		fmt.Fprintf(&b, "\t\tif err := deps.ResponseWriter.Write(ctx, w, %s, response); err != nil {\n", status)
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
	} else {
		fmt.Fprintf(&b, "\t\tif err := %s; err != nil {\n", call)
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
		fmt.Fprintf(&b, "\t\tif err := deps.ResponseWriter.Write(ctx, w, %s, nil); err != nil {\n", status)
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
	}

	b.WriteString("\t}\n}\n")
	return b.String()
}

// renderRegister emits RegisterRoutes, which binds every route on a ServeMux.
func renderRegister(app *model.Application, byID map[model.ComponentID]*binding, im *imports, rt, httpPkg func(string) string, hasDispatcher bool) string {
	var b strings.Builder
	b.WriteString("// RegisterRoutes registers every generated handler on the mux.\n")
	fmt.Fprintf(&b, "func RegisterRoutes(mux *%s, components *Components, deps %s) {\n",
		httpPkg("ServeMux"), rt("HTTPHandlerDependencies"))
	if hasDispatcher {
		// Route errors through @ExceptionHandler advice before the delegate. The
		// wrapper captures the original ErrorHandler as its delegate.
		b.WriteString("\tdeps.ErrorHandler = newExceptionDispatcher(components, deps)\n")
	}
	for _, route := range app.Routes {
		bd := byID[route.Controller]
		if bd == nil {
			continue
		}
		pattern := route.Method + " " + route.Pattern
		fmt.Fprintf(&b, "\tmux.HandleFunc(%s, %s(components.%s, deps))\n",
			strconv.Quote(pattern), handlerFuncName(bd.field, route.HandlerName), bd.field)
	}
	b.WriteString("}\n")
	return b.String()
}

// handleAndReturn is the standard error branch shared by every handler step.
func handleAndReturn() string {
	return "\t\t\tdeps.ErrorHandler.Handle(ctx, w, r, err)\n\t\t\treturn\n"
}

// handlerFuncName builds the unique factory name for a route's handler.
func handlerFuncName(controllerField, handler string) string {
	return "make" + controllerField + handler + "Handler"
}

// stringSliceLit renders a []string literal.
func stringSliceLit(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = strconv.Quote(s)
	}
	return "[]string{" + strings.Join(quoted, ", ") + "}"
}
