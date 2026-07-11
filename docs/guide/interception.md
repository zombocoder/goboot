# Interception (service proxies)

Cross-cutting behavior is applied through **generated interface proxies**. When a
`@Service(implements="Iface")` has methods carrying interceptor annotations,
goboot generates a proxy implementing `Iface` that wraps the target; consumers
inject the interface and transparently get the interceptors.

```go
// @Service(name="orders", implements="OrderService")
type Orders struct{ repo OrderRepository }

// @Transactional
// @Traced
// @Timed
// @Retry(maxAttempts=3, delay="20ms")
func (s *Orders) Place(ctx context.Context, in PlaceInput) (*Order, error) { /* ... */ }
```

An intercepted method takes `context.Context` first and returns `error` last.

## The chain

Interceptors apply in a fixed order (outermost first):

```
timeout → tracing → logging → audit → metrics
        → bulkhead → circuit breaker → rate limit
        → authorize → retry → transaction → target
```

| Annotation | Effect | Default impl |
| ---------- | ------ | ------------ |
| `@Timeout("2s")` | `context.WithTimeout` around the call | built-in |
| `@Traced` | tracing span (observes the error) | no-op → [otel adapter](../plugins/adapters.md) |
| `@Logged(level="info")` | structured log around the call | no-op → app-provided |
| `@Audit(action, resource)` | audit event with the outcome | no-op → app-provided |
| `@Timed` | success/failure metrics | no-op → [prometheus adapter](../plugins/adapters.md) |
| `@Bulkhead(maxConcurrent=16)` | concurrency isolation | built-in |
| `@CircuitBreaker(...)` | fail-fast breaker | built-in |
| `@RateLimit(limit, period)` | token-bucket throttle | built-in |
| `@Authorize(roles, mode)` / `@RolesAllowed([...])` | authorization check | permit-all → app-provided |
| `@Retry(maxAttempts, delay, ...)` | retry with backoff | built-in |
| `@Transactional` | run in a DB transaction | direct → [adapter](repositories.md#transactions) |

The resilience gates (`@Timeout`/`@Retry`/`@CircuitBreaker`/`@RateLimit`/
`@Bulkhead`) ship with **real in-memory implementations** — they work with no
extra wiring. The observability seams (`@Traced`/`@Timed`/`@Logged`/`@Audit`) and
`@Authorize` default to no-op/permit-all; provide real implementations via
`runtime.ProxyDependencies`:

```go
proxyDeps := runtime.DefaultProxyDependencies()
proxyDeps.Tracer  = goboototel.NewTracer(otel.Tracer("app"))
proxyDeps.Metrics = gobootprom.NewMetrics(reg)
proxyDeps.Logger  = myLogger  // implements runtime.MethodLogger
proxyDeps.Audit   = myAudit   // implements runtime.AuditSink
```
