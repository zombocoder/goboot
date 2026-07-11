# Contributing to goboot

Thanks for your interest in contributing! This document explains how to get set up, the conventions we follow, and how to propose changes. By participating you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md).

## Getting started

goboot is a standard Go module. You need **Go 1.25 or newer**.

```bash
git clone https://github.com/zombocoder/goboot
cd goboot
go build ./...
go test ./...
```

There are no external services required to run the test suite; everything runs against in-memory fakes and the standard library. (Live-database repository tests are behind a build tag and are optional.)

## The source of truth

`implementation-plan.md` is the authoritative technical specification. Code and comments reference it by section (e.g. `§24.4`). Before designing a change, check whether the spec already describes the behavior, and keep new work consistent with the architecture in §6 and §60.

One intentional deviation from the spec: the provider annotation is named **`@Nut`** (the spec calls it `@Bean`).

## Architecture at a glance

The pipeline is: **annotations → scan → analyze → generate**.

- `annotation/` — the annotation lexer/parser, value model, schema registry, diagnostics
- `compiler/` — loads packages (`go/packages`), associates comments with declarations, discovers components/routes/lifecycle/proxies/repositories, resolves the dependency graph
- `model/` — the intermediate application model
- `graph/` — dependency graph, topological order, cycle detection
- `runtime/` — the small set of abstractions generated code depends on (HTTP, config, lifecycle, interception, scheduling, db)
- `generator/di/` — emits the wiring
- `sqlgen/` — named-parameter SQL compiler with pluggable dialects
- `adapters/` — driver bindings (e.g. `databasesql`)
- `plugin/` — the compile-time extension API
- `cmd/goboot/` — the CLI

## Non-negotiable invariants

Please preserve these when contributing (see §6, §60):

- **Compile-time only.** No runtime package scanning or global service locator. DI resolution happens at generation time.
- **Type safety via `go/types`.** Interface satisfaction and dependency compatibility are decided by the type checker, never by string comparison.
- **Deterministic output.** The same input must produce byte-identical generated code. Sort everything; never depend on map iteration order, timestamps, or absolute paths.
- **Diagnostics, not panics.** User and annotation errors surface as source-positioned `Diagnostic`s with stable codes (`GOBANN*`, `GOBDI*`, `GOBHTTP*`, `GOBCFG*`, `GOBLIF*`, `GOBPRX*`, `GOBREP*`, `GOBSCH*`, `GOBPLG*`). Parsers must never panic on arbitrary input (they are fuzz-tested).
- **Adapters stay out of the core.** The compiler and generators must not import a specific router, database driver, or telemetry library. Those are adapters/plugins.

## Development workflow

Before opening a pull request, make sure all of the following pass:

```bash
go build ./...
go vet ./...
gofmt -l .            # must print nothing
go test -race ./...
```

### Generated code and golden tests

Some tests compare generated output against committed golden files and against committed integration wiring in `internal/`. If you change a generator, you must regenerate them:

- Update the generator golden: `go test ./generator/di/ -run TestGenerateWiringGolden -update`
- Regenerate the committed integration wiring in `internal/e2e`, `internal/cfge2e`, `internal/proxye2e`, `internal/repoe2e`, `internal/schede2e` (the staleness-guard tests in `generator/di` will tell you which are out of date and how they were produced).

Never hand-edit a `zz_goboot_*.gen.go` or `wiring.gen.go` file; regenerate it.

## Testing expectations

- **Unit tests** for new logic, tables where it helps.
- **Golden + compile tests** for generators: assert the generated source and confirm it compiles.
- **Integration tests** for runtime behavior: drive the generated code (see the `internal/*e2e` packages).
- **Fuzz targets** for any new parser; it must never panic.
- Keep package coverage in line with the existing packages (roughly 85%+).

## Commit and pull request conventions

- Use focused commits with clear messages. We follow a Conventional-Commits-style prefix: `feat(scope): ...`, `fix(scope): ...`, `refactor(scope): ...`, `docs: ...`, `test: ...`, `chore: ...`.
- Reference the relevant spec section(s) in the body when applicable.
- One logical change per pull request. Include tests and update docs.
- Ensure the full checklist above passes locally; CI runs the same checks.

## Reporting bugs and requesting features

Use the GitHub issue templates. For bugs, include a minimal reproduction (an annotated snippet plus the generated output or diagnostic). For security issues, do **not** open a public issue — see [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions are licensed under the [Apache License, Version 2.0](LICENSE).
