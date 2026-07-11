package di

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// renderLifecycle emits buildLifecycle, which registers each component's
// @PostConstruct and @PreDestroy hooks with a runtime.Lifecycle in construction
// order (§30). It returns the empty string when no component has a hook.
func renderLifecycle(app *model.Application, bindings []*binding, byID map[model.ComponentID]*binding, im *imports, f features) string {
	if !f.hasLifecycle {
		return ""
	}
	rt := func(sym string) string { return im.qualify(runtimePath, "runtime", sym) }
	ctxType := im.qualify("context", "context", "Context")

	var b strings.Builder
	b.WriteString("// buildLifecycle registers component lifecycle hooks in construction order.\n")
	fmt.Fprintf(&b, "func buildLifecycle(components *Components) *%s {\n", rt("Lifecycle"))
	fmt.Fprintf(&b, "\tlc := %s(0)\n", rt("NewLifecycle"))
	for _, bd := range bindings {
		c := bd.comp
		if !c.HasLifecycle() {
			continue
		}
		recv := "components." + bd.field
		start := hookClosure(c.PostConstruct, recv, ctxType)
		stop := hookClosure(c.PreDestroy, recv, ctxType)
		fmt.Fprintf(&b, "\tlc.Register(%s, %s, %s)\n", strconv.Quote(string(c.ID)), start, stop)
	}
	b.WriteString("\treturn lc\n}\n")
	return b.String()
}

// hookClosure renders a lifecycle hook as a func(ctx) error closure, adapting
// the method's actual signature (§30.2), or "nil" when the hook is absent.
func hookClosure(m *model.LifecycleMethod, recv, ctxType string) string {
	if m == nil {
		return "nil"
	}
	arg := ""
	if m.TakesContext {
		arg = "ctx"
	}
	call := fmt.Sprintf("%s.%s(%s)", recv, m.MethodName, arg)
	if m.ReturnsError {
		return fmt.Sprintf("func(ctx %s) error { return %s }", ctxType, call)
	}
	return fmt.Sprintf("func(ctx %s) error { %s; return nil }", ctxType, call)
}

// renderApplication emits NewApplication, which builds the components, registers
// routes, assembles the lifecycle, and returns a runtime.Application ready to
// Run (§32). It is emitted only when the application has routes or lifecycle
// hooks; a pure dependency graph needs only buildComponents.
func renderApplication(app *model.Application, im *imports, f features) string {
	if !f.hasRoutes && !f.hasLifecycle {
		return ""
	}
	rt := func(sym string) string { return im.qualify(runtimePath, "runtime", sym) }

	var params []string
	var buildArgs []string
	if f.hasConfig {
		params = append(params, "configSource "+im.qualify(configPath, "config", "Source"))
		buildArgs = append(buildArgs, "configSource")
	}
	if f.hasProxies {
		params = append(params, "proxyDeps "+rt("ProxyDependencies"))
		buildArgs = append(buildArgs, "proxyDeps")
	}
	if f.hasRoutes {
		params = append(params, "deps "+rt("HTTPHandlerDependencies"), "addr string")
	}

	var b strings.Builder
	b.WriteString("// NewApplication builds the components and assembles a runnable application.\n")
	fmt.Fprintf(&b, "func NewApplication(%s) (*%s, error) {\n", strings.Join(params, ", "), rt("Application"))
	fmt.Fprintf(&b, "\tcomponents, err := buildComponents(%s)\n", strings.Join(buildArgs, ", "))
	b.WriteString("\tif err != nil {\n\t\treturn nil, err\n\t}\n")

	if f.hasLifecycle {
		b.WriteString("\tlc := buildLifecycle(components)\n")
	} else {
		fmt.Fprintf(&b, "\tvar lc *%s\n", rt("Lifecycle"))
	}

	if f.hasRoutes {
		httpPkg := func(sym string) string { return im.qualify("net/http", "http", sym) }
		fmt.Fprintf(&b, "\tmux := %s()\n", httpPkg("NewServeMux"))
		b.WriteString("\tRegisterRoutes(mux, components, deps)\n")
		fmt.Fprintf(&b, "\tserver := &%s{Addr: addr, Handler: mux}\n", httpPkg("Server"))
		fmt.Fprintf(&b, "\treturn &%s{Server: server, Lifecycle: lc}, nil\n", rt("Application"))
	} else {
		fmt.Fprintf(&b, "\treturn &%s{Lifecycle: lc}, nil\n", rt("Application"))
	}
	b.WriteString("}\n")
	return b.String()
}
