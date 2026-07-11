package di

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// configPath is the import path of the goboot config package the generated
// loaders depend on.
const configPath = "github.com/zombocoder/goboot/runtime/config"

// features records which optional wiring sections an application needs, so the
// generator can emit only what is required and keep signatures minimal.
type features struct {
	hasConfig    bool
	hasRoutes    bool
	hasLifecycle bool
	hasProxies   bool
	hasRepos     bool
	hasScheduled bool
}

// detectFeatures inspects the application for configuration properties, routes,
// lifecycle hooks, and service proxies.
func detectFeatures(app *model.Application) features {
	var f features
	f.hasRoutes = len(app.Routes) > 0
	for _, c := range app.Components {
		if c.Constructor != nil && c.Constructor.ConfigLoader {
			f.hasConfig = true
		}
		if c.HasLifecycle() {
			f.hasLifecycle = true
		}
		if c.Kind == model.ComponentProxy {
			f.hasProxies = true
		}
		if c.Repository != nil {
			f.hasRepos = true
		}
		if len(c.Scheduled) > 0 {
			f.hasScheduled = true
		}
	}
	return f
}

// dbPath is the import path of the goboot db package generated repositories
// depend on.
const dbPath = "github.com/zombocoder/goboot/runtime/db"

// buildComponentsParam returns the parameter list for buildComponents: a config
// source when the application has configuration properties and proxy
// dependencies when it has service proxies.
func buildComponentsParam(f features, im *imports) string {
	var params []string
	if f.hasConfig {
		params = append(params, "configSource "+im.qualify(configPath, "config", "Source"))
	}
	if f.hasProxies {
		params = append(params, "proxyDeps "+im.qualify(runtimePath, "runtime", "ProxyDependencies"))
	}
	if f.hasRepos {
		params = append(params, "dbProvider "+im.qualify(dbPath, "db", "DBProvider"))
	}
	return strings.Join(params, ", ")
}

// renderConfigLoaders emits a typed Load<Type> function for each configuration
// properties component (§28.5). Each loader binds its struct from the config
// source under the component's prefix.
func renderConfigLoaders(app *model.Application, im *imports) string {
	var b strings.Builder
	for _, c := range app.Components {
		if c.Constructor == nil || !c.Constructor.ConfigLoader {
			continue
		}
		typeRef := renderType(c.ProvidedType, im)
		bindRef := im.qualify(configPath, "config", "Bind")
		sourceRef := im.qualify(configPath, "config", "Source")
		im.add("fmt", "fmt")

		fmt.Fprintf(&b, "// %s loads %s from configuration under the %q prefix.\n",
			c.Constructor.FuncName, typeRef, c.ConfigPrefix)
		fmt.Fprintf(&b, "func %s(source %s) (%s, error) {\n", c.Constructor.FuncName, sourceRef, typeRef)
		fmt.Fprintf(&b, "\tvar out %s\n", typeRef)
		fmt.Fprintf(&b, "\tif err := %s(%s, source, &out); err != nil {\n", bindRef, strconv.Quote(c.ConfigPrefix))
		fmt.Fprintf(&b, "\t\treturn out, fmt.Errorf(%q, err)\n", "load "+c.Name+": %w")
		b.WriteString("\t}\n")
		b.WriteString("\treturn out, nil\n")
		b.WriteString("}\n")
	}
	return b.String()
}
