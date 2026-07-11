# Repositories

Declare a `@Repository(generate=true)` **interface** with `@Query`/`@Exec`
methods; goboot generates the implementation over a driver-neutral DB seam and
injects it with a `db.DBProvider`. The SQL is compiled for the configured
dialect — switching drivers changes nothing else.

```go
// @Repository(generate=true, entity="User", table="users")
type UserRepository interface {
    // @Query(`SELECT id, name, email FROM users WHERE id = :id`)
    FindByID(ctx context.Context, id string) (*User, error)

    // @Query(`SELECT id, name, email FROM users ORDER BY name`)
    FindAll(ctx context.Context) ([]*User, error)

    // @Exec(`INSERT INTO users (id, name, email) VALUES (:u.ID, :u.Name, :u.Email)`)
    Insert(ctx context.Context, u User) error

    // @Exec(`DELETE FROM users WHERE id = :id`)
    Delete(ctx context.Context, id string) (int64, error)
}
```

## Named parameters

`:name` binds a method argument; `:arg.Field` binds a struct field. They compile
to the dialect's placeholders (`$1`, `?`, `@p1`, …). Field references keep call
sites tidy — pass the whole entity instead of many scalars.

## Return shapes

| Signature | Behaviour |
| --------- | --------- |
| `(*T, error)` | single row; no rows → `db.ErrNoRows` |
| `([]*T, error)` / `([]T, error)` | slice of rows |
| `(scalar, error)` | single column (int/string/bool/…) |
| `error` (`@Exec`) | run only |
| `(int64, error)` (`@Exec`) | rows affected |

## `@Batch` and `@Call`

- **`@Batch`** runs its statement once per element of a slice parameter; fields
  bind via the slice name (`:users.ID`). Returns `error` or `(int64, error)`.
- **`@Call`** invokes a stored procedure/function — scanned like `@Query` when it
  returns a value, run like `@Exec` when it returns only `error`.

## Dialects

Set `generation.dialect` in `goboot.yaml` or pass `-dialect`:

| Dialect | Placeholders | Drivers |
| ------- | ------------ | ------- |
| `postgres` (default) | `$1, $2` | pgx, pq |
| `mysql` / `question` | `?` | MySQL, SQLite |
| `sqlserver` | `@p1, @p2` | SQL Server |
| `oracle` | `:1, :2` | via the [oracle plugin](../plugins/index.md) |

## Transactions

Mark a service method `@Transactional`; the generated proxy runs it via the
configured `TransactionManager`, publishing the transaction on the context so the
repository joins it automatically. Provide a real manager from an
[adapter](../plugins/adapters.md):

```go
proxyDeps.Transactions = pgxadapter.NewTransactionManager(pool)
```
