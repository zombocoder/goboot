package di

import (
	"fmt"
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
	b.WriteString(renderRegister(app, byID, im, rt, httpPkg))
	return b.String()
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

	if len(route.Authorize) > 0 {
		fmt.Fprintf(&b, "\t\tif err := deps.Authorizer.Authorize(ctx, %s{\n", rt("AuthorizationRequest"))
		fmt.Fprintf(&b, "\t\t\tRoles: %s,\n", stringSliceLit(route.Authorize))
		fmt.Fprintf(&b, "\t\t\tMode:  %s,\n", rt("AuthorizationModeAny"))
		b.WriteString("\t\t}); err != nil {\n")
		b.WriteString(handleAndReturn())
		b.WriteString("\t\t}\n")
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
func renderRegister(app *model.Application, byID map[model.ComponentID]*binding, im *imports, rt, httpPkg func(string) string) string {
	var b strings.Builder
	b.WriteString("// RegisterRoutes registers every generated handler on the mux.\n")
	fmt.Fprintf(&b, "func RegisterRoutes(mux *%s, components *Components, deps %s) {\n",
		httpPkg("ServeMux"), rt("HTTPHandlerDependencies"))
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
