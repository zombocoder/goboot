# Config, lifecycle & scheduling

## Typed configuration

`@ConfigurationProperties(prefix="...")` generates a typed loader that binds a
struct from YAML and environment with defaults:

```go
// @ConfigurationProperties(prefix="server")
type ServerProperties struct {
    Host string `config:"host" default:"0.0.0.0"`
    Port int    `config:"port" default:"8080"`
    TLS  bool   `config:"tls"  required:"true"`
}
```

goboot generates `LoadServerProperties(source config.Source) (ServerProperties, error)`.
Tags: `config:"key"`, `default:"..."`, `required:"true"`.

## Lifecycle

`@PostConstruct` and `@PreDestroy` methods run on startup and shutdown:

```go
// @PostConstruct
func (s *Engine) Start(ctx context.Context) error { /* ... */ }

// @PreDestroy
func (s *Engine) Stop() { /* ... */ }
```

Hooks accept `()`, `() error`, `(context.Context)`, or `(context.Context) error`.
`@PostConstruct` hooks run in construction order (with rollback on failure);
`@PreDestroy` hooks run in reverse for graceful shutdown. `app.Run(ctx)` drives
both around the HTTP server.

## Scheduling

`@Scheduled` registers a background task on the runtime scheduler:

```go
// @Scheduled(fixedRate="15s")
func (r *Reporter) Report(ctx context.Context) error { /* ... */ }
```

Arguments: `fixedRate`, `fixedDelay`, `initialDelay`, `timeUnit` — e.g.
`@Scheduled(fixedRate=2, timeUnit=TimeUnit.MINUTES)` or the duration-string form
above.

## Profiles & conditions

Gate components on the active profile or a property:

```go
// @Profile(["prod", "staging"])
// @ConditionalOnProperty(name="cache.enabled", havingValue="true")
```

Also `@ConditionalOnNut(type="...")` / `@ConditionalOnMissingNut(type="...")`.
Pass active profiles and properties at generation time:

```bash
goboot generate -profile prod -property cache.enabled=true ./...
```

Excluded components are dropped before resolution, so proxies, routes, and the
graph only ever see the active set.
