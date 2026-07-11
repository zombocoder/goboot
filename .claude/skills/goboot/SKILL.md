---
name: goboot
description: >
  Reference for building applications with the goboot framework — an
  annotation-driven, compile-time DI/HTTP/config framework for Go. Use when
  writing or reviewing goboot-annotated Go code (types/methods with // @Service,
  // @RestController, // @Nut, // @Query, etc.), running the goboot CLI
  (generate/validate/graph), authoring goboot plugins, or answering questions
  about which annotation to use and what code it generates. Covers every
  annotation, its arguments, valid targets, generated behavior, and
  implementation status.
---

# Working with goboot

goboot is a **compile-time** framework: you annotate ordinary Go types and methods in comments; the `goboot` CLI reads the annotations, builds a typed model + dependency graph, validates it, and generates ordinary readable Go. There is **no runtime reflection for DI** and **no classpath scanning**. Dependency resolution uses `go/types` (never string matching), and generated output is deterministic.

Authoritative spec: `implementation-plan.md` (referenced by `§` throughout the code). One deviation: the provider annotation is **`@Nut`**.

## The workflow

1. Write annotated Go (annotations are `// @Name(arg=value, ...)` doc comments on the declaration).
2. Run generation:
   ```bash
   go run github.com/zombocoder/goboot/cmd/goboot generate ./...
   # or add a directive and use `go generate`:
   //go:generate go run github.com/zombocoder/goboot/cmd/goboot generate ./...
   ```
3. The generated `zz_goboot_wiring.gen.go` provides `buildComponents(...)`, `RegisterRoutes`, `NewApplication`, etc. Wire it into your `main` and run.

CLI: `generate`, `validate` (analyze, no write), `graph --format mermaid|dot|json|text`, `clean`, `doctor`, `init`, `version`. Useful flags on generate/validate: `-profile prod,staging`, `-property key=value`, `-dialect postgres|question`, `-strict`, `-tags`.

## Annotation syntax

- Attached as doc comments immediately above the declaration.
- `// @Service` (marker), `// @Service(name="x", scope="singleton")` (named args), `// @Profile(["prod","staging"])` (array), `// @Timeout("2s")` (positional), ``// @Query(`SELECT ...`)`` (raw string / multi-line).
- Values: strings, ints, floats, bools, `null`, identifiers (e.g. `singleton`, `TimeUnit.MINUTES`), arrays `[...]`, objects `{k=v}`.

## Method signatures (conventions)

- **Constructor**: `func NewXxx(deps...) *Xxx` or `(*Xxx, error)`; may return an interface. Reject >2 returns or a non-`error` second return.
- **HTTP handler**: `func(ctx context.Context, req Request) (*Response, error)` (also `(ctx) (Response, error)`, `(ctx, req) error`, `(ctx) error`). First param must be `context.Context`.
- **Lifecycle / Scheduled hook**: `func()`, `func() error`, `func(context.Context)`, or `func(context.Context) error`.
- **`@ExceptionHandler`** (on a `@ControllerAdvice`): `func(ctx context.Context, err *T) (*Response, error)` (response form — writes the body with `@ResponseStatus`, default 500) or `func(ctx, err *T) error` (transform form — the returned error is rendered as a Problem by the delegate). The caught type is the second parameter (matched via `errors.As`); `err error` makes it a catch-all, tried after all concrete handlers.
- **Intercepted service method** (@Transactional/@Traced/@Timed/@Retry/@Timeout/@Authorize/@Logged/@Audit/@CircuitBreaker/@RateLimit/@Bulkhead): first param `context.Context`, last result `error`.
- **Request binding tags**: `path:"id"`, `query:"expand"`, `header:"X-Request-ID"`, `cookie:"c"`, `json:"name"`.
- **Config tags**: `config:"host" default:"0.0.0.0"` (and `required:"true"`).

## Annotation reference

Status: ✅ implemented · 🚧 planned (parse/generate not yet wired).

### Components & DI

| Annotation       | Target      | Args                                                 | Generates                                                                          | Status |
| ---------------- | ----------- | ---------------------------------------------------- | ---------------------------------------------------------------------------------- | ------ |
| `@Application`   | struct      | `name` (req), `scan`, `profiles`, `configuration`    | app root; one per target                                                           | ✅     |
| `@Service`       | struct      | `name`, `scope` (singleton\|prototype), `implements` | a component; `implements` enables a proxy when the service has intercepted methods | ✅     |
| `@Component`     | struct      | `name`                                               | a generic component                                                                | ✅     |
| `@Configuration` | struct      | —                                                    | a config grouping (holds `@Nut` providers)                                         | ✅     |
| `@Nut`           | func/method | `name`                                               | a provider-function component                                                      | ✅     |
| `@Primary`       | struct/func | —                                                    | preferred candidate when a dependency is ambiguous                                 | ✅     |
| `@Named`         | struct/func | positional string                                    | a component name                                                                   | ✅     |
| `@Scope`         | struct      | positional (singleton\|prototype)                    | component scope                                                                    | ✅     |
| `@Qualifier`     | —           | —                                                    | qualifier-based resolution                                                         | 🚧     |
| `@Lazy`          | struct      | —                                                    | lazy initialization                                                                | 🚧     |

### HTTP

| Annotation                                          | Target              | Args                                                        | Generates                                                      | Status          |
| --------------------------------------------------- | ------------------- | ----------------------------------------------------------- | -------------------------------------------------------------- | --------------- |
| `@RestController`                                   | struct              | —                                                           | an HTTP controller component                                   | ✅              |
| `@RequestMapping`                                   | struct              | `path`, `host`, `headers`                                   | controller base path                                           | ✅              |
| `@GetMapping` / `@PostMapping`                      | method              | `path`, `name`, `consumes`, `produces`, `timeout`, `status` | a route + handler proxy (bind→validate→authorize→invoke→write) | ✅              |
| `@PutMapping` / `@PatchMapping` / `@DeleteMapping`   | method              | same                                                        | a route + handler proxy for that verb (defaults: PUT/PATCH 200, DELETE 204 no-body) | ✅  |
| `@Response`                                         | method (repeatable) | `status`, `type`, `error`, `contentType`                    | response/status metadata; overrides default status             | ✅              |
| `@ResponseStatus`                                   | method              | positional int                                              | success status                                                 | ✅              |
| `@ControllerAdvice`                                 | struct              | —                                                           | advice component holding `@ExceptionHandler` methods           | ✅              |
| `@ExceptionHandler`                                 | method              | `type` (optional; caught type is read from the 2nd param)   | typed error→response dispatch (see below)                      | ✅              |
| `@Consumes`/`@Produces`                             | method              | —                                                           | media-type constraints                                         | 🚧              |

### Configuration & lifecycle

| Annotation                 | Target | Args           | Generates                                                     | Status |
| -------------------------- | ------ | -------------- | ------------------------------------------------------------- | ------ |
| `@ConfigurationProperties` | struct | `prefix` (req) | a typed `Load<Type>` loader bound from YAML/env with defaults | ✅     |
| `@PostConstruct`           | method | —              | startup hook (ordered)                                        | ✅     |
| `@PreDestroy`              | method | —              | shutdown hook (reverse order)                                 | ✅     |
| `@Value`                   | field  | —              | property injection into a field                               | 🚧     |

### Conditions & profiles

| Annotation                 | Target    | Args                                          | Behavior                                                 | Status |
| -------------------------- | --------- | --------------------------------------------- | -------------------------------------------------------- | ------ |
| `@Profile`                 | component | positional `[]string`                         | active only under a listed active profile (`-profile`)   | ✅     |
| `@ConditionalOnProperty`   | component | `name` (req), `havingValue`, `matchIfMissing` | gated on a property value (`-property`)                  | ✅     |
| `@ConditionalOnNut`        | component | `type` (req)                                  | present only if a component of that name/type is present | ✅     |
| `@ConditionalOnMissingNut` | component | `type` (req)                                  | present only if absent                                   | ✅     |

### Interception (service proxies — require `@Service(implements="Iface")`)

Chain order (§25): timeout → tracing → logging → audit → metrics → **bulkhead → circuit breaker → rate limit** → authorize → retry → transaction → target. The resilience gates default their registry key to `Type.Method` unless given an explicit `name` (shared name → shared state). Unlike the no-op observability defaults, the gate providers ship **real in-memory implementations** (they need no external system), so the protection is active out of the box.

| Annotation        | Target | Args                                              | Generates                                       | Status           |
| ----------------- | ------ | ------------------------------------------------- | ----------------------------------------------- | ---------------- |
| `@Transactional`  | method | `readOnly`, `isolation`, `propagation`, `timeout` | wraps in `TransactionManager.WithinTransaction` | ✅               |
| `@Traced`         | method | `name`                                            | tracing span around the call                    | ✅               |
| `@Timed`          | method | `name`                                            | success/failure metrics                         | ✅               |
| `@Timeout`        | method | positional duration (`"2s"`)                      | `context.WithTimeout` around the call           | ✅               |
| `@Retry`          | method | `maxAttempts`, `delay`, `multiplier`, `maxDelay`  | `runtime.Retry` with backoff                    | ✅               |
| `@Authorize`      | method | `roles`, `permissions`, `mode` (any\|all)         | authorization check before invoke               | ✅               |
| `@RolesAllowed`   | method | positional `[]string`                             | shorthand for `@Authorize(roles=...)`           | ✅ |
| `@Logged`         | method | `level` (debug\|info\|warn\|error, default info)  | structured logging around the call via `MethodLogger` | ✅          |
| `@Audit`          | method | `action`, `resource`                              | audit event via an `AuditSink` after the call   | ✅               |
| `@CircuitBreaker` | method | `name`, `failureThreshold`, `resetTimeout`, `halfOpenMax` | fail-fast breaker (`ErrCircuitOpen` when open) via `CircuitBreakerProvider` | ✅ |
| `@RateLimit`      | method | `name`, `limit`, `period`, `burst`                | token-bucket throttle (`ErrRateLimited`) via `RateLimiterProvider` | ✅ |
| `@Bulkhead`       | method | `name`, `maxConcurrent`, `maxWait`                | concurrency-limit semaphore (`ErrBulkheadFull`) via `BulkheadProvider` | ✅ |

Note: `@Authorize` also works at the **HTTP route** level today (roles are passed to the `Authorizer` in the generated handler); both HTTP route level and service-method level (a proxy interceptor calling the Authorizer before invoking the target).

### Repositories (`@Repository(generate=true)` interface)

| Annotation         | Target              | Args                                   | Generates                                                           | Status |
| ------------------ | ------------------- | -------------------------------------- | ------------------------------------------------------------------- | ------ |
| `@Repository`      | struct or interface | `name`, `entity`, `table`, `generate`  | component-mode (struct) or generated impl (interface)               | ✅     |
| `@Query`           | interface method    | positional SQL (`:name`, `:arg.Field`) | a query method (single/slice/scalar scan, no-rows → `db.ErrNoRows`) | ✅     |
| `@Exec`            | interface method    | positional SQL                         | an exec method (error or `(int64, error)` rows-affected)            | ✅     |
| `@Batch` / `@Call` | interface method    | —                                      | batch / stored-proc                                                 | 🚧     |

SQL dialect is a pluggable seam: `-dialect postgres` (`$1`, default) or `question` (`?`); plugins add more (e.g. SQL Server `@p1`).

### Scheduling

| Annotation   | Target | Args                                                  | Generates                                                                                                                            | Status |
| ------------ | ------ | ----------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| `@Scheduled` | method | `fixedRate`, `fixedDelay`, `initialDelay`, `timeUnit` | a background task on `runtime.Scheduler`; e.g. `@Scheduled(fixedRate=2, timeUnit=TimeUnit.MINUTES)` or `@Scheduled(fixedRate="30s")` | ✅     |

### Observability, security, resilience details

Runtime interfaces backing the interceptors live in `runtime/`: `TransactionManager`, `Tracer`/`Span`, `MethodMetrics`, `MethodLogger`, `AuditSink`/`AuditEvent`, `Authorizer`/`AuthorizationRequest`, `RetryPolicy`, `CircuitBreakerProvider`, `RateLimiterProvider`, `BulkheadProvider`. Defaults are no-op/permit-all/direct so generated code runs before adapters are configured. Provide real implementations via `runtime.ProxyDependencies` (proxies), `runtime.HTTPHandlerDependencies` (HTTP), and a `db.DBProvider` (repositories).

## Diagnostics

Errors are source-positioned with stable codes: `GOBANN*` (annotation), `GOBDI*` (DI), `GOBHTTP*` (HTTP), `GOBCFG*`/`GOBLIF*` (config/lifecycle), `GOBPRX*` (proxies), `GOBREP*` (repositories), `GOBSCH*` (scheduling), `GOBPLG*` (plugins). Common ones: missing/ambiguous dependency, dependency cycle, duplicate route, invalid handler signature, concrete injection of a proxied service (inject the interface instead).

## Extending with plugins

Plugins are compile-time (no dynamic loading). Implement `plugin.Plugin` plus any of `AnnotationProvider` (register annotations), `Analyzer` (diagnostics), `Generator` (files), `DialectProvider` (a DB driver's placeholder style). See `plugin/exampleplugin`. Host them by building a small `main` that injects them.

## When editing this framework's own source

Preserve the invariants: compile-time only, `go/types`-based resolution, deterministic output, diagnostics-not-panics, adapters out of the core. If you change a generator, regenerate the golden and the committed `internal/*e2e` wirings (staleness-guard tests will tell you which). Run `go build ./... && go vet ./... && gofmt -l . && go test -race ./...` before committing.
