# goboot plugins

goboot is extended at **compile time** by plugins — ordinary Go modules linked
into the CLI (there is no runtime `.so` loading, §46). A plugin contributes any
combination of: annotation schemas, extra analysis (diagnostics), generated
files, and SQL dialects / database drivers.

## Using plugins

List them in `goboot.yaml`. Each entry is either a shorthand `module@version`
scalar or an explicit mapping:

```yaml
plugins:
  - github.com/zombocoder/goboot/plugins/oracle@latest        # shorthand
  - module:  github.com/acme/goboot-redis                     # explicit form
    version: v1.3.1
    import:  github.com/acme/goboot-redis/gobootx             # optional, default = module
    new:     New                                              # optional, default = New
```

Then run generation as usual:

```bash
goboot generate ./...
```

The stock `goboot` binary detects the `plugins:` block and **self-bootstraps**: it
builds a plugin-aware CLI from the list (cached under `.goboot/`), re-execs it,
and your plugins are active. A changed plugin set transparently triggers a
rebuild. Inspect the setup with `goboot plugins` (configured vs. linked).

- `GOBOOT_BOOTSTRAP=off goboot generate …` runs the plugin-free binary.
- For reproducible / offline CI, `goboot plugins sync` pins the modules and writes
  a committed `tools/goboot/main.go`; drive generation with
  `go run ./tools/goboot generate ./...`.

## Official plugins

Official plugins live in this repository under `plugins/<name>/`, each its own Go
module (`github.com/zombocoder/goboot/plugins/<name>`).

| Plugin   | Module                                          | Capability                          |
| -------- | ----------------------------------------------- | ----------------------------------- |
| `oracle` | `github.com/zombocoder/goboot/plugins/oracle`   | Oracle SQL dialect (`:1`, `:2` …)   |

Community plugins can live in any module; tag your repo with the `goboot-plugin`
GitHub topic so others can find it.

## Writing a plugin

A plugin is a module that depends on `github.com/zombocoder/goboot` and exports a
constructor (by convention `New`) returning a value that implements
`plugin.Plugin` plus any capability interfaces. Minimal dialect plugin:

```go
package oracle

import (
	"strconv"

	"github.com/zombocoder/goboot/plugin"
	"github.com/zombocoder/goboot/sqlgen"
)

func New() *Plugin { return &Plugin{} }

type Plugin struct{}

func (*Plugin) Name() string    { return "oracle" }
func (*Plugin) Version() string { return "0.1.0" }

// DialectProvider capability.
func (*Plugin) Dialects() map[string]sqlgen.Dialect {
	return map[string]sqlgen.Dialect{"oracle": Dialect{}}
}

type Dialect struct{}

func (Dialect) Name() string             { return "oracle" }
func (Dialect) Placeholder(i int) string { return ":" + strconv.Itoa(i) }
```

Capability interfaces (implement any subset — see `plugin/plugin.go`):

- `AnnotationProvider` — register annotation schemas the compiler recognizes.
- `Analyzer` — contribute extra diagnostics over the analyzed model.
- `Generator` — emit additional files alongside the wiring.
- `DialectProvider` — register SQL dialects / database drivers.

Contract rules (§46.4): return diagnostics rather than panic, and produce
deterministic output. `plugin.APIVersion` is the contract version a plugin links
against; an incompatible major fails to compile (compile-time safety). See
`plugin/exampleplugin` for a plugin exercising all four capabilities, and
`plugins/oracle` for a minimal standalone module.
