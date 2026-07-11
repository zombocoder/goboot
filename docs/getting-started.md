# Getting started

## Install

```bash
go install github.com/zombocoder/goboot/cmd/goboot@latest
```

Requires **Go 1.25+**. Verify:

```bash
goboot version
```

## Scaffold

In your module:

```bash
goboot init
```

This writes a `goboot.yaml`:

```yaml
application:
  name: my-service
  packages:
    - ./internal/...
generation:
  output: internal/generated
  package: generated
  clean: true
  dialect: postgres
```

## A first app

goboot works well with clean-architecture layering: **domain → repository →
service → controller**, each depending inward through an interface.

=== "domain"

    ```go
    package domain

    type Todo struct {
        ID    string
        Title string
        Done  bool
    }
    ```

=== "repository"

    ```go
    package repository

    // @Repository(generate=true, entity="Todo", table="todos")
    type TodoRepository interface {
        // @Exec(`INSERT INTO todos (id, title, done) VALUES (:t.ID, :t.Title, :t.Done)`)
        Insert(ctx context.Context, t domain.Todo) error

        // @Query(`SELECT id, title, done FROM todos WHERE id = :id`)
        FindByID(ctx context.Context, id string) (*domain.Todo, error)
    }
    ```

=== "service"

    ```go
    package service

    type TodoUseCase interface {
        Create(ctx context.Context, title string) (*domain.Todo, error)
    }

    // @Service(name="todoService", implements="TodoUseCase")
    type TodoService struct{ repo repository.TodoRepository }

    func NewTodoService(repo repository.TodoRepository) *TodoService {
        return &TodoService{repo}
    }

    // @Transactional
    func (s *TodoService) Create(ctx context.Context, title string) (*domain.Todo, error) {
        t := domain.Todo{ID: uuid.NewString(), Title: title}
        return &t, s.repo.Insert(ctx, t)
    }
    ```

=== "controller"

    ```go
    package controller

    // @RestController
    // @RequestMapping(path="/todos")
    type TodoController struct{ todos service.TodoUseCase }

    func NewTodoController(todos service.TodoUseCase) *TodoController {
        return &TodoController{todos}
    }

    // @PostMapping(path="")
    func (c *TodoController) Create(ctx context.Context, req CreateRequest) (*domain.Todo, error) {
        return c.todos.Create(ctx, req.Title)
    }
    ```

=== "app root"

    ```go
    package app

    // @Application(name="todo")
    type Application struct{}
    ```

## Generate

```bash
goboot generate ./...
```

goboot writes `internal/generated/zz_goboot_wiring.gen.go`, exposing
`NewApplication(...)`, `RegisterRoutes(...)`, and `buildComponents(...)`. It
constructs your components in dependency order, generates the SQL repository
implementation, and applies the `@Transactional` interceptor through a generated
proxy.

!!! tip "Reproducible builds"
    Add a directive so `go generate` keeps the wiring in sync:
    ```go
    //go:generate go run github.com/zombocoder/goboot/cmd/goboot generate ./...
    ```

## Wire and run

In `cmd/server/main.go`, hand goboot the concrete infrastructure — for PostgreSQL
via the [pgx adapter](plugins/adapters.md):

```go
pool, _ := pgxpool.New(ctx, dsn)
dbProvider := pgxadapter.NewProvider(pool)

proxyDeps := runtime.DefaultProxyDependencies()
proxyDeps.Transactions = pgxadapter.NewTransactionManager(pool) // enables @Transactional
httpDeps := runtime.DefaultHTTPHandlerDependencies()

app, _ := generated.NewApplication(proxyDeps, dbProvider, httpDeps, ":8080")
_ = app.Run(ctx) // serves HTTP, graceful shutdown
```

```bash
go run ./cmd/server
```

That's it — you now have a running service whose wiring is plain, readable,
version-controllable Go.

Next: the [developer guide](guide/architecture.md) and the
[annotation reference](reference/annotations.md).
