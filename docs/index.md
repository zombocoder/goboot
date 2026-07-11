# goboot

**An annotation-driven, _compile-time_ application framework for Go.**

goboot gives you a Spring Boot–style developer experience that compiles down to
plain, readable Go — **no runtime reflection for dependency injection, no
classpath scanning**. You annotate ordinary Go types and methods; the `goboot`
CLI reads the annotations, builds a typed application model and dependency graph,
validates it, and generates ordinary Go you can read and step through in a
debugger.

```go
// @RestController
// @RequestMapping(path="/users")
type UserController struct{ users UserUseCase }

func NewUserController(users UserUseCase) *UserController { return &UserController{users} }

// @PostMapping(path="")
func (c *UserController) Create(ctx context.Context, req CreateRequest) (*UserResponse, error) {
    return c.users.Create(ctx, req.toInput())
}
```

```bash
goboot generate ./...
```

## Why compile-time?

The single architectural principle:

> **Annotations describe intent; the compiler validates that intent into a
> semantic model; generators turn the model into plain Go.**

Dependency resolution, component discovery, and wiring all happen at *generation
time*. The program that ships does no reflection-based DI and no startup scanning,
and interface/dependency compatibility is decided by `go/types` — never strings.

## What you get

<div class="grid cards" markdown>

- :material-source-branch: **Dependency injection** — constructor injection,
  interface resolution, `@Primary`, provider functions, deterministic order,
  cycle detection.
- :material-web: **HTTP** — controllers, all verbs, request binding/validation,
  RFC-7807 errors, `@ExceptionHandler`, content negotiation.
- :material-database: **Repositories** — generated SQL from `@Query`/`@Exec`/
  `@Batch`/`@Call`; driver-neutral, pluggable dialects.
- :material-shield-check: **Interception** — `@Transactional`, `@Traced`,
  `@Timed`, `@Logged`, `@Audit`, `@Retry`, `@Timeout`, `@CircuitBreaker`,
  `@RateLimit`, `@Bulkhead`, `@Authorize`.
- :material-cog: **Config & lifecycle** — typed config, `@PostConstruct`/
  `@PreDestroy`, `@Scheduled`, profiles & conditions.
- :material-puzzle: **Plugins & adapters** — extend generation at compile time;
  plug real backends (pgx, OTel, Prometheus) into the runtime seams.

</div>

## Next steps

- [Getting started](getting-started.md) — install and build your first app.
- [Architecture](guide/architecture.md) — the compile-time pipeline.
- [Annotation reference](reference/annotations.md) — every annotation.
