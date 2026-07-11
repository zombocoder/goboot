# Dependency injection

Components are the injectable units. goboot discovers them from annotations,
resolves their constructor dependencies by type, orders construction
topologically, and generates a `buildComponents(...)` function.

## Declaring components

```go
// @Service(name="userService", implements="UserUseCase")
type UserService struct{ repo UserRepository }

func NewUserService(repo UserRepository) *UserService { return &UserService{repo} }
```

- `@Application` — the app root; exactly one per module.
- `@Service` / `@Component` — a component. `implements="Iface"` exposes it as an
  interface and enables an [interception proxy](interception.md) when it has
  intercepted methods.
- `@Configuration` + `@Nut` — a config grouping holding provider functions:

```go
// @Configuration
type Config struct{}

// @Nut(name="clock")
func (Config) Clock() Clock { return realClock{} }
```

## Constructors

A constructor is `func NewXxx(deps...) *Xxx`, `(*Xxx, error)`, or a form returning
an interface. Its parameters are the component's dependencies, resolved by type.
More than two returns, or a non-`error` second return, is rejected.

## Resolution rules

- Dependencies are matched with `go/types` — a parameter of interface type
  resolves to any component whose provided type implements it.
- When a dependency is **ambiguous**, mark the preferred candidate `@Primary`,
  or name candidates with `@Named` / the `name` argument.
- **Missing** dependencies, **ambiguous** ones, and **cycles** are reported as
  `GOBDI*` diagnostics with source positions — never runtime panics.

## Scope

`@Scope(singleton)` (default) or `@Scope(prototype)`. Singletons are built once
in `buildComponents`; prototypes are constructed per injection point.

!!! note "Interface-based proxies"
    If a service has intercepted methods (e.g. `@Transactional`), consumers must
    inject the **interface** it declares via `implements=`, not the concrete
    type. Injecting the concrete type when a proxy exists is a compile error
    (`GOBPRX001`).
