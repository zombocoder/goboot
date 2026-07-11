# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

**M1–M7 plus the CLI and a plugin system are implemented.** `implementation-plan.md` remains the authoritative spec for architecture, scope, naming, and conventions — consult it (§ references throughout the code) before extending anything. One intentional deviation: the spec's `@Bean` provider annotation is named **`@Nut`** in this codebase (de-Java-ified).

Implemented packages:
- `annotation/` — lexer/parser, value model, schema registry, diagnostics (M1)
- `compiler/` — go/packages loader, comment→declaration association; discovery of components, routes, lifecycle hooks, service proxies, and repositories; dependency resolver (M2–M7)
- `model/` — the intermediate application model consumed by generators
- `graph/` — dependency graph, topological order, cycle detection, Mermaid
- `runtime/` — HTTP binding/validation/errors/response, lifecycle, Application, interception (TransactionManager/Tracer/MethodMetrics); `runtime/config/` typed config; `runtime/db/` driver-neutral DB abstraction
- `sqlgen/` — named-parameter SQL compiler with a pluggable `Dialect` (the driver seam)
- `generator/di/` — emits the wiring: `buildComponents`, config loaders, HTTP handler proxies + `RegisterRoutes`, `buildLifecycle`, `NewApplication`, **service proxies** (interception), and **repository implementations**
- `adapters/databasesql/` — reference driver binding over stdlib `database/sql` + a `TransactionManager`
- `plugin/` — the compile-time extension API (annotations, analyzers, generators, SQL dialects/drivers); `plugin/exampleplugin/` is a reference plugin exercising all four capabilities
- `cli/` — the importable CLI implementation (`cli.Main(plugins...)`, `cli.Run`): `generate`, `validate`, `graph`, `clean`, `doctor`, `init`, `plugins`, `version`; injected plugins live in the `hostPlugins` var
- `cmd/goboot/` — the thin default binary; a thin `main` calling `cli.Main()` with no plugins
- `internal/e2e`, `internal/cfge2e`, `internal/proxye2e`, `internal/repoe2e` — committed generated wiring + integration tests that drive it (kept in sync by staleness guards in `generator/di`)

Remaining: M8 hardening, more adapters (native pgx, OTel, Prometheus), OpenAPI, `@Profile`/conditional nuts. See §55–56.

Module path: `github.com/zombocoder/goboot` (Go 1.25).

## Extending via plugins

External packages extend goboot at compile time through the `plugin` package (§46) — no dynamic loading. A plugin implements `plugin.Plugin` plus any of the optional capability interfaces: `AnnotationProvider` (register annotation schemas), `Analyzer` (extra diagnostics), `Generator` (emit files), `DialectProvider` (register a SQL dialect / DB driver). A host builds a `plugin.Registry` (`plugin.New(...)`) which merges plugin annotations into the annotation registry, runs analyzers/generators with panic recovery, and resolves dialects. The CLI lives in the importable `cli` package: `cli.Main(pluginA.New(), ...)` injects plugins (stored in the `hostPlugins` var); the default `cmd/goboot` binary injects none. A project lists plugins in `goboot.yaml` (`plugins:`) and `goboot generate` **self-bootstraps** — it builds a plugin-aware CLI from that list (cached under `.goboot/`, keyed by the plugin set + toolchain, via `cli/bootstrap.go`), re-execs it, and the plugins are active. `GOBOOT_BOOTSTRAP=off` disables it; `goboot plugins sync` writes a committed `tools/goboot/main.go` for CI; `goboot plugins` lists configured vs. linked. `plugin/exampleplugin` is the in-module worked example; **official plugins live under `plugins/<name>/` as separate modules** (`plugins/oracle` is the first — an Oracle dialect). Plugin submodules have their own `go.mod` (a `replace => ../..` for local dev), so the root `go build/test ./...` skips them; CI builds/tests each under a dedicated step. See `PLUGINS.md`.

## What goboot is

An annotation-driven, **compile-time** application framework for Go — a Spring Boot–style developer experience implemented as a code generator, **not** a runtime DI container. Developers annotate Go types/methods with `// @Service`, `// @RestController`, `// @GetMapping(...)` etc. A CLI compiler (`cmd/goboot`) parses these annotations, builds a typed application model + dependency graph, validates it, and emits ordinary, readable Go source. The generated code compiles and runs with no runtime reflection for DI and no classpath scanning.

The single most important architectural constraint (§60): **annotations describe intent; the compiler validates intent into a semantic model; generators turn the model into plain Go.** Never move DI resolution, component discovery, or wiring into runtime — it all happens at generation time.

## Core architecture (the pipeline)

Work flows through a fixed pipeline (§7, §37). Each stage is a distinct package and should stay decoupled from the next by the intermediate model:

1. **`annotation/`** — lexer + parser for the `@Name(arg=value, ...)` comment language into an `Annotation{Name, Arguments map[string]Value, Position, Raw}` AST. Each annotation has a registered `Definition` schema (valid targets, argument types, defaults) used for validation. Parsers must **never panic on arbitrary input** (fuzz-tested).
2. **`compiler/`** — loads packages via `golang.org/x/tools/go/packages` (with `NeedTypes|NeedTypesInfo|NeedSyntax|...`), associates comments with declarations, and does type analysis with `go/types`. Type matching uses `go/types` (`types.Implements`, checking both `T` and `*T`), **never string comparison**.
3. **`model/`** — the intermediate `Application` model (components, controllers, routes, repositories, configurations, advice, graph, diagnostics). This model must **not** depend on any concrete router/DB implementation. Generators consume only this model.
4. **`graph/`** — directed dependency graph (consumer → dependency), topological sort for construction order, cycle detection, reverse-order shutdown, mermaid/dot/json export.
5. **`generator/`** — sub-generators (`di/`, `http/`, `proxy/`, `repository/`, `configuration/`, `lifecycle/`, `openapi/`) that emit Go source, then run `go/format` + `imports.Process`. Output must be **deterministic** (no map-iteration order, no timestamps, no absolute paths) and written atomically.
6. **`runtime/`** — minimal reusable abstractions the generated code depends on (lifecycle, HTTP binding, `Problem`/error handling, transactions, authorization, observability). Keep this small; it is not a container.
7. **`adapters/`** — integration points (`httpchi`, `httpstd`, `pgx`, `slog`, `otel`, `prometheus`). The compiler core must never import these directly — only through adapter/plugin seams (§6.6).

Generated files carry `// Code generated by goboot. DO NOT EDIT.`, use the `zz_goboot_*.gen.go` naming (§40), and land in `internal/generated/` of the target project.

## Non-negotiable invariants

- **Determinism** (§6.7): same input → byte-identical output. This is why generators sort everything and avoid `Date.now`-style nondeterminism.
- **Type safety via `go/types`** (§6.2): interface satisfaction and dependency compatibility are decided by the type checker, not strings.
- **Compile-time only** (§6.1): no runtime package scanning or global service locator.
- **Constructor injection** (§6.4): prefer `NewXxx(...) *Xxx` / `(*Xxx, error)` / interface returns. Reject >2 returns or a non-`error` second return (§13.4).
- **Interface-based proxies** (§24.3): methods needing interception (transactions, retries, tracing) require the consumer to inject the **interface**, not the concrete type. Injecting the concrete type when a proxy exists is a compile error.
- **Diagnostics, not panics**: user/annotation errors surface as source-positioned `Diagnostic`s with stable codes (`GOBANN*`, `GOBDI*`, `GOBHTTP*`, `GOBREP*`, `GOBCFG*`, `GOBLIF*`, `GOBPRX*`, `GOBPLG*` — §39.4), never panics.

## Scope discipline

Build strictly to the versioned scope. **MVP / v0.1** (§54) = annotation parser, compiler, DI wiring, Chi HTTP (GET/POST), config (YAML+env), lifecycle, and the CLI (`init generate validate graph doctor clean version`). Explicitly **excluded from v0.1** (§54.2): generated SQL repositories, transactions, service proxies, resilience, authorization impl, OTel, OpenAPI. Those arrive in v0.2/v0.3 (§55–56) — but design v0.1 seams so they can be added without breaking generated interfaces. Follow the milestone order in §58 (annotation language → scanner → DI → HTTP → config/lifecycle → proxies → repositories → hardening).

## Commands

```bash
go build ./...                 # build everything
go test ./...                  # run all tests (unit + golden + compile + integration)
go test -race ./...            # race detection (required in CI, §49.2)
go test ./annotation/          # test one package
go test ./annotation/ -run TestLexer   # run a single test by name
go test ./generator/di/ -run TestGenerateWiringGolden -update   # regenerate the golden
go vet ./...                   # static analysis (§49.1)
gofmt -l .                     # formatting check
```

**Golden / generated-wiring workflow.** `generator/di` has golden tests and *staleness guards* (`TestE2EWiringUpToDate`, `TestCfgE2EWiringUpToDate`) that compare the committed `internal/e2e/wiring.gen.go` and `internal/cfge2e/wiring.gen.go` against freshly generated output. If you change the generator, regenerate the golden (above) **and** the committed e2e wiring, or these tests fail. The e2e wiring is produced from the `compiler/testdata/diapp` and `compiler/testdata/cfgapp` fixtures.

**Running the CLI:**

```bash
go run ./cmd/goboot version
go run ./cmd/goboot validate -dir compiler ./testdata/diapp/...     # analyze, print diagnostics, no write
go run ./cmd/goboot graph -dir compiler -format mermaid ./testdata/diapp/...
go run ./cmd/goboot generate -dir <moduledir> -output internal/generated -package generated ./...
```

The CLI, once built, is driven by `go generate`:

```bash
go run ./cmd/goboot generate ./...     # generate into internal/generated
go run ./cmd/goboot validate ./...     # validate without writing files
go run ./cmd/goboot graph ./... --format mermaid
```

Recommended `go:generate` directive in target projects:
`//go:generate go run github.com/zombocoder/goboot/cmd/goboot generate ./...`

## Testing expectations (§48)

- **Golden-file tests** are mandatory for every generator: input fixtures in `internal/testdata/<feature>/`, expected output in a `golden/` subdir, comparing complete generated source.
- **Compile tests**: generated fixtures (valid apps + error cases: missing deps, cycles, duplicate routes, invalid annotations, ambiguous interfaces) must actually compile or produce the expected diagnostic.
- **Fuzz** the lexer, annotation parser, SQL named-param parser, path-template parser, config-key parser — they must never panic.
- Coverage floors (§49.4): parser/resolver/graph 90%, generators/runtime 80%. Coverage alone is insufficient — golden + compile tests are the real gate.
