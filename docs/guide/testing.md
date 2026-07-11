# Testing

Because goboot generates **plain Go** and every layer depends inward through an
interface, you test the way you always have — no framework test harness.

## Service layer (no database)

Depend on the repository **interface**, and a hand-written in-memory fake tests
your business rules with zero infrastructure:

```go
type fakeRepo struct{ items map[string]domain.Todo }

func (f *fakeRepo) FindByID(_ context.Context, id string) (*domain.Todo, error) {
    t, ok := f.items[id]
    if !ok {
        return nil, db.ErrNoRows
    }
    return &t, nil
}
// ...implement the rest of the interface...

func TestCreate(t *testing.T) {
    svc := service.NewTodoService(&fakeRepo{items: map[string]domain.Todo{}})
    todo, err := svc.Create(context.Background(), "buy milk")
    // assert on todo/err
}
```

## HTTP layer

Register the generated routes on an `httptest` server and drive real requests:

```go
components, _ := generated.BuildComponents(fakeDB)
mux := http.NewServeMux()
generated.RegisterRoutes(mux, components, runtime.DefaultHTTPHandlerDependencies())
srv := httptest.NewServer(mux)
defer srv.Close()

resp, _ := http.Get(srv.URL + "/todos/42")
```

## Repository layer

Repositories depend only on the `runtime/db` interfaces, so a fake `db.DBTX`
returning canned rows tests scanning and argument binding without a database — or
run an integration test against a real DB via an adapter, gated on an env var:

```go
func TestPostgres(t *testing.T) {
    dsn := os.Getenv("TEST_DSN")
    if dsn == "" {
        t.Skip("set TEST_DSN to run")
    }
    // pgxpool.New(...) → pgxadapter.NewProvider(...) → drive the repo
}
```

## Generated wiring

Commit the generated files and let `go build ./...` and CI compile them — a
compile failure means the wiring drifted from the annotated code. Regenerate with
`goboot generate ./...` (it self-heals stale output before loading).
