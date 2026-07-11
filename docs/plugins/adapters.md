# Adapters

Generated code depends only on small runtime interfaces; **adapters** plug real
backends into those seams. Each adapter is its own module (isolating its
dependency) — `go get` the ones you need.

## Database — `adapters/pgx`

Native PostgreSQL over `jackc/pgx/v5`. Supplies a `db.DBProvider` for repositories
and a `TransactionManager` for `@Transactional`. Pair with the default `postgres`
dialect.

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    pgxadapter "github.com/zombocoder/goboot/adapters/pgx"
)

pool, _ := pgxpool.New(ctx, dsn)
dbProvider := pgxadapter.NewProvider(pool)

proxyDeps := runtime.DefaultProxyDependencies()
proxyDeps.Transactions = pgxadapter.NewTransactionManager(pool)
```

`adapters/databasesql` (in the core module) does the same over any stdlib
`database/sql` driver.

## Tracing — `adapters/otel`

Makes `@Traced` emit real OpenTelemetry spans (with error recording).

```go
import (
    "go.opentelemetry.io/otel"
    goboototel "github.com/zombocoder/goboot/adapters/otel"
)

proxyDeps.Tracer = goboototel.NewTracer(otel.Tracer("my-service"))
```

Configure a global `TracerProvider` in your app as usual.

## Metrics — `adapters/prometheus`

Makes `@Timed` increment `goboot_method_calls_total{method,outcome}`.

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    gobootprom "github.com/zombocoder/goboot/adapters/prometheus"
)

reg := prometheus.NewRegistry()
proxyDeps.Metrics = gobootprom.NewMetrics(reg)
http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
```

## Logging & audit

`@Logged` and `@Audit` have no default backend — implement `runtime.MethodLogger`
and `runtime.AuditSink` (a few lines over `slog`) and set `proxyDeps.Logger` /
`proxyDeps.Audit`.
