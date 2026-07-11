# goboot

**An annotation-driven, compile-time application framework for Go.**

goboot gives you a Spring Boot–style developer experience while keeping everything Go loves: explicit dependencies, static typing, fast startup, and readable code. You annotate ordinary Go types and methods; a CLI compiler reads the annotations, builds a typed application model and dependency graph, validates it, and generates **ordinary, readable Go** — with **no runtime reflection for dependency injection** and **no classpath scanning**.

```go
// @Service(name="userService", implements="UserUseCase")
type UserService struct {
    repo domain.UserRepository
}

func NewUserService(repo domain.UserRepository) *UserService { return &UserService{repo: repo} }

// @Transactional
// @Traced(name="users.create")
func (s *UserService) CreateUser(ctx context.Context, cmd CreateUserCommand) (*domain.User, error) {
    // ...
}
```

```bash
go run github.com/zombocoder/goboot/cmd/goboot generate ./...
```

You get generated wiring that constructs your components in dependency order, registers HTTP routes, applies transactions/tracing/metrics through generated proxies, loads typed configuration, runs lifecycle hooks and scheduled tasks, and starts a server — all as plain Go you can read and debug.

## Why compile-time?

The single architectural principle: **annotations describe intent; the compiler validates that intent into a semantic model; generators turn the model into plain Go.** Dependency resolution, component discovery, and wiring all happen at generation time. The generated program does not scan packages or discover components at startup, and interface/dependency compatibility is checked with `go/types` — never strings.

## Features

- **Dependency injection** — constructor injection, interface resolution, `@Primary`, `@Nut` provider functions, deterministic construction order, cycle detection.
- **HTTP controllers** — `@RestController`/`@GetMapping`/`@PostMapping` generate handler proxies (bind → validate → authorize → invoke → write) with centralized RFC 7807 error handling.
- **Configuration** — typed `@ConfigurationProperties` bound from YAML and environment with defaults.
- **Lifecycle** — `@PostConstruct`/`@PreDestroy` with ordered startup, rollback, and graceful shutdown.
- **Service proxies (interception)** — `@Transactional`, `@Traced`, `@Timed` wrap methods through generated interface proxies.
- **Repositories** — generate implementations from `@Query`/`@Exec` interfaces with named SQL parameters; **driver-agnostic** via a pluggable dialect.
- **Conditions & profiles** — `@Profile`, `@ConditionalOnProperty`, `@ConditionalOnNut`, `@ConditionalOnMissingNut`.
- **Scheduling** — `@Scheduled` background tasks.
- **Plugin system** — external packages register annotations, analyzers, generators, and database dialects at compile time.

## Install

```bash
go install github.com/zombocoder/goboot/cmd/goboot@latest
```

Or pin it via `go:generate` in your project (recommended for reproducible builds):

```go
//go:generate go run github.com/zombocoder/goboot/cmd/goboot generate ./...
```

## CLI

```bash
goboot init                       # scaffold goboot.yaml
goboot generate ./...             # generate wiring into the output package
goboot validate ./...             # analyze and report diagnostics, no files written
goboot graph ./... --format mermaid
goboot clean                      # remove generated files
goboot doctor                     # environment checks
goboot version
```

Useful flags on `generate`/`validate`: `-profile prod,staging`, `-property cache.enabled=true`, `-dialect postgres|question`, `-strict`, `-tags`.

## Extending with plugins

goboot is extended at compile time through the `plugin` package — no dynamic loading. A plugin implements `plugin.Plugin` plus any of `AnnotationProvider`, `Analyzer`, `Generator`, or `DialectProvider` (a database driver). See [`plugin/exampleplugin`](plugin/exampleplugin) for a worked example.

## Status

goboot is under active development. The core framework — DI, HTTP, configuration, lifecycle, interception, repositories, conditions/profiles, scheduling, a CLI, and a plugin system — is implemented and tested. See [`implementation-plan.md`](implementation-plan.md) for the full technical specification and roadmap.

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) and our [Code of Conduct](CODE_OF_CONDUCT.md). Security issues: see [SECURITY.md](SECURITY.md).

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
