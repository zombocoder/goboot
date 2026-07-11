# Technical Specification

# Annotation-Driven Compile-Time Framework for Go

## 1. Document purpose

This document defines the technical requirements for creating an annotation-driven application framework for Go.

The framework must provide a Spring Boot–inspired developer experience while preserving the core properties of Go:

* explicit dependencies;
* static typing;
* compile-time validation;
* predictable application startup;
* readable generated code;
* minimal runtime reflection;
* compatibility with standard Go tooling;
* no hidden runtime classpath scanning;
* no mandatory global service locator.

The framework will use annotations written in Go comments to describe application components, HTTP controllers, repositories, configuration, lifecycle hooks, transactions, authorization, observability, resilience policies, and other infrastructure concerns.

A dedicated CLI compiler will scan Go packages, parse annotations, validate application structure, build a dependency graph, and generate type-safe Go source files.

---

# 2. Product vision

The framework must allow developers to write application code in the following style:

```go
// @Service
type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) *UserService {
	return &UserService{
		repository: repository,
	}
}
```

```go
// @RestController
// @RequestMapping(path="/api/v1/users")
type UserController struct {
	service UserUseCase
}

func NewUserController(service UserUseCase) *UserController {
	return &UserController{
		service: service,
	}
}
```

```go
// @GetMapping(path="/{id}")
// @Authorize(roles=["users.read"])
// @Response(status=200)
// @Response(status=404, error="user_not_found")
func (c *UserController) GetUser(
	ctx context.Context,
	request GetUserRequest,
) (*UserResponse, error) {
	return c.service.GetUser(ctx, request.ID)
}
```

The framework must generate:

* dependency injection wiring;
* controller HTTP handlers;
* request decoding;
* validation calls;
* authorization calls;
* error handling;
* response serialization;
* middleware and interceptor chains;
* service proxies;
* repository implementations;
* application startup and shutdown orchestration;
* configuration loading;
* optional OpenAPI descriptions.

The generated code must be regular Go code that can be inspected, debugged, tested, and compiled without a custom runtime.

---

# 3. Working project name

The temporary project name used in this specification is:

```text
goboot
```

The final product name may be changed later without affecting the architecture.

Proposed modules:

```text
github.com/<organization>/goboot
github.com/<organization>/goboot/runtime
github.com/<organization>/goboot/annotations
github.com/<organization>/goboot/http
github.com/<organization>/goboot/repository
github.com/<organization>/goboot/testing
```

CLI installation:

```bash
go install github.com/<organization>/goboot/cmd/goboot@latest
```

---

# 4. Goals

## 4.1 Primary goals

The framework must:

1. Provide annotation-driven component declarations.
2. Generate dependency injection code at compile time.
3. Generate HTTP proxy handlers around controller methods.
4. Provide centralized error handling.
5. Support repository implementation generation.
6. Support declarative transactions.
7. Support application lifecycle management.
8. Support configuration binding and validation.
9. Detect dependency and annotation errors before application startup.
10. Generate readable and deterministic Go code.
11. Work with standard Go modules.
12. Support multiple infrastructure adapters.
13. Avoid runtime reflection for dependency injection.
14. Support incremental adoption in existing Go applications.
15. Allow extension through a compiler plugin architecture.

## 4.2 Secondary goals

The framework should eventually support:

* OpenAPI generation;
* tracing;
* metrics;
* structured logging;
* retries;
* circuit breakers;
* rate limiting;
* authentication and authorization;
* background workers;
* Kafka consumers and producers;
* Temporal workers;
* testing utilities;
* test dependency overrides;
* mock generation;
* build-time application graph visualization.

---

# 5. Non-goals

The framework must not:

* recreate the Java Virtual Machine model;
* perform runtime package scanning;
* depend on Java-style reflection;
* hide all dependency construction from developers;
* generate domain business logic;
* require a specific ORM;
* require a specific HTTP router;
* require a specific logging library;
* use dynamic Go plugins as the main extension mechanism;
* replace the Go compiler;
* introduce a custom source language;
* modify method bodies;
* require a proprietary build system;
* make generated files the source of business truth.

The user-authored Go code remains the primary source of application behavior.

---

# 6. Architectural principles

## 6.1 Compile-time over runtime

Application structure must be analyzed during code generation.

The generated application should not scan packages or discover components at startup.

## 6.2 Type safety

Dependency compatibility must be checked using `go/types`.

Type matching must not be based only on strings.

## 6.3 Generated code visibility

All generated source files must be available in the project and readable by developers.

Generated files must contain:

```go
// Code generated by goboot. DO NOT EDIT.
```

## 6.4 Explicit constructors

Components should normally define constructors.

```go
func NewUserService(repository UserRepository) *UserService
```

The framework must prefer constructor injection over field injection.

## 6.5 Interface-based proxies

Services that require transactions, retries, tracing, authorization, or other method-level interception must be injected through interfaces.

## 6.6 Adapter-based integrations

The compiler core must not be tightly coupled to:

* Chi;
* Gin;
* Echo;
* Fiber;
* pgx;
* `database/sql`;
* Zap;
* slog;
* OpenTelemetry.

Integrations must be implemented through adapters.

## 6.7 Deterministic generation

The same source code and configuration must produce byte-equivalent generated output, excluding explicitly allowed metadata.

Generated output must not depend on:

* map iteration order;
* filesystem discovery order;
* local timestamps;
* machine-specific absolute paths.

---

# 7. High-level architecture

```text
┌───────────────────────────┐
│ User Go source code       │
│                           │
│ Components                │
│ Controllers               │
│ Repositories              │
│ Configurations            │
│ Annotations               │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Package Loader            │
│ go/packages               │
│ go/ast                    │
│ go/types                  │
│ go/token                  │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Annotation Parser         │
│                           │
│ Tokenization              │
│ Values                    │
│ Arrays                    │
│ Named arguments           │
│ Source positions          │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Semantic Analyzer         │
│                           │
│ Component discovery       │
│ Constructor discovery     │
│ Route discovery           │
│ Repository discovery      │
│ Configuration discovery   │
│ Method validation         │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Application Model         │
│                           │
│ Components                │
│ Dependency graph          │
│ Controllers               │
│ Routes                    │
│ Repositories              │
│ Interceptors              │
│ Lifecycle hooks           │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Validation                │
│                           │
│ Dependency cycles         │
│ Ambiguous bindings        │
│ Duplicate routes          │
│ Invalid signatures        │
│ Invalid annotations       │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Code generators           │
│                           │
│ Dependency injection      │
│ HTTP handlers             │
│ Service proxies           │
│ Repositories              │
│ Configuration             │
│ Lifecycle                 │
│ OpenAPI                   │
└─────────────┬─────────────┘
              │
              ▼
┌───────────────────────────┐
│ Generated Go source       │
│                           │
│ go/format                 │
│ imports processing        │
│ compile verification      │
└───────────────────────────┘
```

---

# 8. Repository structure

```text
goboot/
├── cmd/
│   └── goboot/
│       ├── main.go
│       ├── generate.go
│       ├── validate.go
│       ├── graph.go
│       ├── doctor.go
│       └── version.go
│
├── annotation/
│   ├── parser.go
│   ├── lexer.go
│   ├── token.go
│   ├── value.go
│   ├── schema.go
│   ├── target.go
│   ├── registry.go
│   ├── diagnostics.go
│   └── parser_test.go
│
├── compiler/
│   ├── compiler.go
│   ├── loader.go
│   ├── scanner.go
│   ├── analyzer.go
│   ├── package.go
│   ├── comments.go
│   ├── constructors.go
│   ├── signatures.go
│   ├── diagnostics.go
│   └── source_position.go
│
├── model/
│   ├── application.go
│   ├── package.go
│   ├── component.go
│   ├── component_id.go
│   ├── constructor.go
│   ├── dependency.go
│   ├── controller.go
│   ├── route.go
│   ├── repository.go
│   ├── query.go
│   ├── configuration.go
│   ├── lifecycle.go
│   ├── interceptor.go
│   └── diagnostic.go
│
├── graph/
│   ├── graph.go
│   ├── builder.go
│   ├── resolver.go
│   ├── topological_sort.go
│   ├── cycles.go
│   └── mermaid.go
│
├── generator/
│   ├── generator.go
│   ├── output.go
│   ├── file.go
│   ├── imports.go
│   ├── names.go
│   ├── formatter.go
│   ├── di/
│   ├── http/
│   ├── proxy/
│   ├── repository/
│   ├── configuration/
│   ├── lifecycle/
│   └── openapi/
│
├── runtime/
│   ├── application.go
│   ├── lifecycle.go
│   ├── errors.go
│   ├── problem.go
│   ├── validation.go
│   ├── binding.go
│   ├── response.go
│   ├── transactions.go
│   ├── authorization.go
│   ├── observability.go
│   └── shutdown.go
│
├── adapters/
│   ├── httpchi/
│   ├── httpstd/
│   ├── pgx/
│   ├── databasesql/
│   ├── slog/
│   ├── otel/
│   └── prometheus/
│
├── plugin/
│   ├── plugin.go
│   ├── registry.go
│   └── context.go
│
├── testing/
│   ├── application.go
│   ├── overrides.go
│   ├── httptest.go
│   └── mocks.go
│
├── examples/
│   ├── hello-world/
│   ├── rest-api/
│   ├── postgres-api/
│   ├── transactional-service/
│   └── modular-application/
│
└── internal/
    ├── testdata/
    └── golden/
```

---

# 9. Annotation syntax

## 9.1 General syntax

Annotations are stored in Go comments.

```go
// @Service
type UserService struct{}
```

Annotations may have named arguments:

```go
// @Service(name="userService", scope="singleton")
type UserService struct{}
```

Annotations may span multiple lines:

```go
// @Authorize(
//   roles=["admin", "support"],
//   mode="any"
// )
```

## 9.2 Supported value types

The parser must support:

* strings;
* integers;
* floating-point numbers;
* booleans;
* arrays;
* nested objects;
* identifiers;
* duration strings;
* enum-like values;
* null.

Examples:

```go
// @Retry(maxAttempts=3, delay="100ms")
// @Response(status=200, contentType="application/json")
// @Authorize(roles=["admin", "support"])
// @Cache(ttl="5m", key="#request.ID")
// @Custom(options={enabled=true, size=10})
```

## 9.3 Parsed representation

```go
type Annotation struct {
	Name      string
	Arguments map[string]Value
	Position  token.Position
	Raw       string
}
```

```go
type Value interface {
	Kind() ValueKind
}
```

```go
type ValueKind uint8

const (
	ValueString ValueKind = iota
	ValueInteger
	ValueFloat
	ValueBoolean
	ValueArray
	ValueObject
	ValueIdentifier
	ValueNull
)
```

## 9.4 Annotation targets

Supported targets:

```go
type Target uint8

const (
	TargetPackage Target = iota
	TargetType
	TargetStruct
	TargetInterface
	TargetFunction
	TargetMethod
	TargetField
	TargetParameter
)
```

Annotations must declare valid targets.

For example:

```text
@Service            → struct
@RestController     → struct
@GetMapping         → method
@Bean               → function or method
@Configuration      → struct
@Transactional      → method or type
```

## 9.5 Annotation schema

Each annotation must have a registered schema.

```go
type Definition struct {
	Name       string
	Targets    []Target
	Arguments  map[string]ArgumentDefinition
	Repeatable bool
	Validator  Validator
}
```

```go
type ArgumentDefinition struct {
	Type         ArgumentType
	Required     bool
	DefaultValue Value
	Allowed      []Value
}
```

The schema must be used for:

* argument validation;
* default values;
* diagnostics;
* documentation generation;
* IDE support in the future.

---

# 10. Initial annotation catalogue

## 10.1 Core annotations

```text
@Application
@Component
@Service
@Repository
@Configuration
@Bean
@Primary
@Qualifier
@Named
@Lazy
@Scope
```

## 10.2 HTTP annotations

```text
@RestController
@RequestMapping
@GetMapping
@PostMapping
@PutMapping
@PatchMapping
@DeleteMapping
@OptionsMapping
@HeadMapping
@Response
@ResponseStatus
@Consumes
@Produces
```

## 10.3 Error handling annotations

```text
@ControllerAdvice
@ExceptionHandler
@ErrorCode
@ResponseStatus
```

## 10.4 Persistence annotations

```text
@Repository
@Query
@Exec
@Transactional
@ReadOnly
@Isolation
```

## 10.5 Configuration annotations

```text
@Configuration
@Bean
@ConfigurationProperties
@Value
@Profile
@ConditionalOnProperty
@ConditionalOnBean
@ConditionalOnMissingBean
```

## 10.6 Lifecycle annotations

```text
@PostConstruct
@PreDestroy
```

## 10.7 Security annotations

```text
@Authorize
@Authenticated
@PermitAll
@RolesAllowed
```

## 10.8 Observability annotations

```text
@Traced
@Timed
@Logged
@Audit
```

## 10.9 Resilience annotations

```text
@Retry
@Timeout
@CircuitBreaker
@RateLimit
@Bulkhead
```

Only a subset will be implemented in the first release.

---

# 11. Application declaration

The application root may be declared as follows:

```go
package main

// @Application(
//   name="users-service",
//   scan=["./internal/..."]
// )
type Application struct{}
```

The `@Application` annotation must support:

```text
name           string, required
scan           []string, optional
profiles       []string, optional
configuration  string, optional
```

Only one root application may exist per generated application target.

Multiple application roots may be supported in one repository when separate output targets are configured.

---

# 12. Component model

## 12.1 Component kinds

```go
type ComponentKind uint8

const (
	ComponentGeneric ComponentKind = iota
	ComponentService
	ComponentRepository
	ComponentController
	ComponentConfiguration
	ComponentBean
	ComponentAdvice
)
```

## 12.2 Component representation

```go
type Component struct {
	ID           ComponentID
	Name         string
	PackagePath  string
	Type         types.Type
	NamedType    *types.Named
	Kind         ComponentKind
	Scope        Scope
	Primary      bool
	Lazy         bool
	Constructor  *Constructor
	Dependencies []Dependency
	Annotations  []annotation.Annotation
	Position     token.Position
}
```

## 12.3 Component ID

Component IDs must be stable.

Recommended format:

```text
<package-import-path>:<type-or-function-name>
```

Example:

```text
github.com/acme/users/internal/service:UserService
```

For named beans:

```text
github.com/acme/users/internal/config:ProvideDatabase#primaryDatabase
```

## 12.4 Component scope

MVP scopes:

```text
singleton
prototype
```

Default:

```text
singleton
```

Future scopes:

```text
request
session
worker
```

Request scope must not be implemented until the runtime context ownership model is clearly defined.

---

# 13. Constructor discovery

## 13.1 Naming convention

For a component named:

```go
type UserService struct{}
```

the default constructor is:

```go
func NewUserService(...) *UserService
```

## 13.2 Explicit constructor annotation

A custom constructor may be declared:

```go
// @Constructor(for="UserService")
func BuildUserService(...) (*UserService, error)
```

## 13.3 Supported return signatures

```go
func NewService(...) *Service
func NewService(...) ServiceInterface
func NewService(...) (*Service, error)
func NewService(...) (ServiceInterface, error)
```

## 13.4 Invalid constructors

The compiler must reject:

* more than two return values;
* second return value that is not `error`;
* variadic dependency parameters unless explicitly supported;
* generic constructors that cannot be instantiated;
* constructors returning unrelated types;
* constructors with unresolvable dependencies.

## 13.5 Constructorless components

Constructorless initialization may be supported only for structs with no required fields:

```go
// @Component
type Clock struct{}
```

Generated initialization:

```go
clock := &clock.Clock{}
```

For MVP, explicit constructors should be recommended.

---

# 14. Dependency injection

## 14.1 Constructor injection

Constructor parameters define component dependencies.

```go
func NewUserService(
	repository UserRepository,
	logger *slog.Logger,
) *UserService
```

## 14.2 Dependency representation

```go
type Dependency struct {
	Name       string
	Type       types.Type
	Qualifier  string
	Optional   bool
	Multiple   bool
	Lazy       bool
	Position   token.Position
	ResolvedTo []ComponentID
}
```

## 14.3 Interface resolution

A component satisfies an interface when:

```go
types.Implements(componentType, interfaceType)
```

Pointer and value receiver behavior must be handled correctly.

The compiler must check both:

```go
types.Implements(T, I)
types.Implements(types.NewPointer(T), I)
```

where appropriate.

## 14.4 Resolution algorithm

For every constructor dependency:

1. Find exact concrete type matches.
2. Find interface implementations.
3. Apply qualifier filters.
4. Apply profile and condition filters.
5. Prefer a primary component.
6. Detect ambiguity.
7. Detect missing dependencies.
8. Add graph edge.
9. Validate component scope compatibility.

## 14.5 Ambiguous dependencies

Given:

```go
type UserRepository interface{}
```

and two implementations:

```go
// @Repository(name="postgresUserRepository")
type PostgresUserRepository struct{}

// @Repository(name="cachedUserRepository")
type CachedUserRepository struct{}
```

the dependency is ambiguous unless:

* one is marked `@Primary`;
* a qualifier is provided;
* one component is disabled by a condition or profile.

## 14.6 Qualifiers

Because parameter comments are difficult to associate reliably and elegantly, MVP should support qualifiers through provider functions.

```go
// @Bean(name="userRepository")
func ProvideUserRepository(
	postgres *PostgresUserRepository,
	cache Cache,
) UserRepository {
	return NewCachedUserRepository(postgres, cache)
}
```

A future version may support:

```go
repository goboot.Qualified[UserRepository, CachedRepositoryQualifier]
```

## 14.7 Collections

The framework should eventually support injecting all implementations:

```go
func NewProcessor(processors []Processor) *ProcessorService
```

MVP may support slices only when explicitly enabled:

```go
// @InjectAll
```

Collection order must be deterministic and configurable through an order annotation.

---

# 15. Dependency graph

The framework must build a directed graph:

```text
consumer component → dependency component
```

The graph is used for:

* cycle detection;
* startup order;
* shutdown order;
* code generation;
* graph visualization;
* diagnostics.

## 15.1 Cycle detection

The compiler must detect circular dependencies before generation.

Example diagnostic:

```text
internal/service/user.go:18:2:
dependency cycle detected:

UserService
  -> NotificationService
  -> AuditService
  -> UserService
```

## 15.2 Construction order

Singletons must be initialized using topological ordering.

## 15.3 Shutdown order

Lifecycle shutdown hooks must execute in reverse construction order.

---

# 16. Bean provider functions

Configuration modules may declare beans.

```go
// @Configuration
type DatabaseConfiguration struct{}
```

```go
// @Bean(name="primaryDatabase")
func ProvideDatabase(
	config DatabaseProperties,
) (*pgxpool.Pool, error) {
	return pgxpool.New(context.Background(), config.URL)
}
```

Bean functions are treated as constructors.

Supported signatures follow the same constructor rules.

A bean may return an interface.

```go
// @Bean
func ProvideClock() Clock {
	return systemClock{}
}
```

Bean methods with configuration receivers may be considered in a later version.

For MVP, package-level provider functions are preferred.

---

# 17. HTTP controller model

## 17.1 Controller declaration

```go
// @RestController
// @RequestMapping(path="/api/v1/users")
type UserController struct {
	service UserUseCase
}
```

## 17.2 Route methods

```go
// @GetMapping(path="/{id}")
func (c *UserController) GetUser(
	ctx context.Context,
	request GetUserRequest,
) (*UserResponse, error)
```

## 17.3 Supported controller signatures

MVP must support:

```go
func(ctx context.Context, request Request) (Response, error)
func(ctx context.Context, request *Request) (*Response, error)
func(ctx context.Context, request Request) error
func(ctx context.Context) (Response, error)
func(ctx context.Context) error
```

Optional future support:

```go
func(ctx context.Context, request Request) (Response, Metadata, error)
func(ctx context.Context, request Request) goboot.Result[Response]
```

## 17.4 Forbidden controller dependencies

Controller methods must not directly accept framework-specific HTTP writer and request objects in the primary programming model.

An escape hatch may be provided:

```go
func(ctx context.Context, exchange goboot.HTTPExchange) error
```

but such methods may have reduced automatic generation features.

---

# 18. HTTP annotations

## 18.1 Request mapping

```go
// @RequestMapping(path="/api/v1/users")
```

Arguments:

```text
path      string
host      string, optional
headers   []string, optional
```

## 18.2 Method mapping

```go
// @GetMapping(path="/{id}")
// @PostMapping(path="")
```

Common arguments:

```text
path         string
name         string, optional
consumes     []string, optional
produces     []string, optional
timeout      duration, optional
status       integer, optional
```

## 18.3 Response declaration

```go
// @Response(status=200, type="UserResponse")
// @Response(status=404, error="user_not_found")
// @Response(status=500, error="internal_error")
```

The annotation may be repeated.

## 18.4 Default status rules

Recommended defaults:

```text
GET      200
POST     201
PUT      200
PATCH    200
DELETE   204
```

Explicit response annotations override defaults.

---

# 19. Request binding

The generated HTTP proxy must bind request values into a request structure.

```go
type GetUserRequest struct {
	ID       uuid.UUID `path:"id" validate:"required"`
	Expand   []string  `query:"expand"`
	Locale   string    `header:"Accept-Language"`
	RequestID string   `header:"X-Request-ID"`
}
```

Supported sources:

```text
path
query
header
cookie
body
form
multipart
context
```

## 19.1 Body binding

```go
type CreateUserRequest struct {
	Name  string `json:"name" validate:"required,min=2,max=128"`
	Email string `json:"email" validate:"required,email"`
}
```

For POST, PUT, and PATCH methods, untagged request structs may be treated as JSON bodies when there are no path, query, or header tags.

However, explicit binding should be preferred for predictable behavior.

## 19.2 Binding interface

```go
type Binder interface {
	Bind(
		ctx context.Context,
		request *http.Request,
		target any,
	) error
}
```

The first implementation may use reflection for struct field binding.

Reflection is acceptable here because request binding is runtime data transformation, not dependency discovery.

## 19.3 Generated binders

A future optimization may generate type-specific request binders.

Example:

```go
func bindGetUserRequest(r *http.Request) (GetUserRequest, error)
```

---

# 20. Validation

## 20.1 Validator interface

```go
type Validator interface {
	Validate(ctx context.Context, value any) error
}
```

## 20.2 Default adapter

The first adapter may support `go-playground/validator`.

The framework core must depend only on the `Validator` interface.

## 20.3 Validation sequence

Generated HTTP flow:

1. Decode path values.
2. Decode query values.
3. Decode headers and cookies.
4. Decode request body.
5. Normalize values.
6. Validate request.
7. Call authorization.
8. Invoke controller.

## 20.4 Validation errors

Validation errors must be converted into a standardized problem response.

```json
{
  "type": "validation_error",
  "title": "Request validation failed",
  "status": 400,
  "errors": [
    {
      "field": "email",
      "code": "email",
      "message": "must be a valid email address"
    }
  ]
}
```

---

# 21. Generated controller proxies

For this controller:

```go
// @GetMapping(path="/{id}")
// @Authorize(roles=["users.read"])
// @Response(status=200)
func (c *UserController) GetUser(
	ctx context.Context,
	request GetUserRequest,
) (*UserResponse, error)
```

the framework must generate a handler equivalent to:

```go
func makeUserControllerGetUserHandler(
	controller *controller.UserController,
	dependencies HTTPHandlerDependencies,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		request, err := bindGetUserRequest(r)
		if err != nil {
			dependencies.ErrorHandler.Handle(ctx, w, r, err)
			return
		}

		if err := dependencies.Validator.Validate(ctx, request); err != nil {
			dependencies.ErrorHandler.Handle(ctx, w, r, err)
			return
		}

		if err := dependencies.Authorizer.Authorize(
			ctx,
			AuthorizationRequest{
				Roles: []string{"users.read"},
				Mode:  AuthorizationModeAny,
			},
		); err != nil {
			dependencies.ErrorHandler.Handle(ctx, w, r, err)
			return
		}

		response, err := controller.GetUser(ctx, request)
		if err != nil {
			dependencies.ErrorHandler.Handle(ctx, w, r, err)
			return
		}

		if err := dependencies.ResponseWriter.Write(
			ctx,
			w,
			http.StatusOK,
			response,
		); err != nil {
			dependencies.ErrorHandler.Handle(ctx, w, r, err)
			return
		}
	}
}
```

The generated proxy must support:

* panic recovery;
* request ID extraction or generation;
* trace propagation;
* request binding;
* validation;
* authorization;
* controller invocation;
* error handling;
* response serialization;
* metrics;
* logging;
* timeout enforcement.

These concerns must be configurable.

---

# 22. HTTP runtime abstractions

```go
type HTTPHandlerDependencies struct {
	Binder         Binder
	Validator      Validator
	Authorizer     Authorizer
	ErrorHandler   ErrorHandler
	ResponseWriter ResponseWriter
	Observer       HTTPObserver
}
```

```go
type ResponseWriter interface {
	Write(
		ctx context.Context,
		writer http.ResponseWriter,
		status int,
		value any,
	) error
}
```

```go
type ErrorHandler interface {
	Handle(
		ctx context.Context,
		writer http.ResponseWriter,
		request *http.Request,
		err error,
	)
}
```

```go
type HTTPObserver interface {
	Begin(
		ctx context.Context,
		operation HTTPRequestOperation,
	) (context.Context, HTTPRequestObservation)
}
```

```go
type HTTPRequestObservation interface {
	End(status int, err error)
}
```

---

# 23. Centralized error handling

## 23.1 Standard problem model

The runtime must provide an RFC 7807–inspired response model.

```go
type Problem struct {
	Type       string            `json:"type"`
	Title      string            `json:"title"`
	Status     int               `json:"status"`
	Detail     string            `json:"detail,omitempty"`
	Instance   string            `json:"instance,omitempty"`
	Code       string            `json:"code,omitempty"`
	Errors     []FieldError      `json:"errors,omitempty"`
	Extensions map[string]any    `json:"extensions,omitempty"`
}
```

## 23.2 Typed application errors

```go
type CodedError interface {
	error
	Code() string
}
```

```go
type HTTPStatusError interface {
	error
	HTTPStatus() int
}
```

## 23.3 Controller advice

```go
// @ControllerAdvice
type APIErrorAdvice struct{}
```

```go
// @ExceptionHandler(type="domain.UserNotFoundError")
// @ResponseStatus(404)
func (a *APIErrorAdvice) HandleUserNotFound(
	ctx context.Context,
	err *domain.UserNotFoundError,
) Problem {
	return Problem{
		Type:   "user_not_found",
		Title:  "User not found",
		Status: http.StatusNotFound,
		Detail: err.Error(),
	}
}
```

## 23.4 Generated error dispatcher

The generator must produce type-safe dispatch logic using `errors.As`.

```go
func dispatchGeneratedError(
	ctx context.Context,
	err error,
	advice *APIErrorAdvice,
) (Problem, bool) {
	var userNotFound *domain.UserNotFoundError
	if errors.As(err, &userNotFound) {
		return advice.HandleUserNotFound(ctx, userNotFound), true
	}

	return Problem{}, false
}
```

## 23.5 Resolution priority

Error handlers must be ordered by:

1. exact concrete error handler;
2. nearest compatible interface handler;
3. coded error mapping;
4. HTTP status error mapping;
5. default framework error handler.

Ambiguous handlers must produce a compile-time diagnostic.

---

# 24. Service proxies

## 24.1 Purpose

Service proxies provide declarative method behavior for:

* transactions;
* retries;
* circuit breakers;
* timeouts;
* tracing;
* metrics;
* audit logging;
* authorization.

## 24.2 Interface requirement

Given:

```go
type UserUseCase interface {
	CreateUser(
		ctx context.Context,
		command CreateUserCommand,
	) (*User, error)
}
```

```go
// @Service(implements="UserUseCase")
type UserService struct{}
```

the generated proxy must implement `UserUseCase`.

```go
type UserServiceProxy struct {
	target      *UserService
	transaction TransactionManager
	tracer      Tracer
	metrics     MethodMetrics
}
```

## 24.3 Concrete injection limitation

The compiler must reject interception when a consumer injects the concrete implementation directly.

Invalid:

```go
func NewController(service *UserService) *Controller
```

Diagnostic:

```text
UserService contains intercepted methods but is injected as concrete type
*UserService.

Inject the UserUseCase interface instead so the generated proxy can be used.
```

## 24.4 Proxy generation

```go
func (p *UserServiceProxy) CreateUser(
	ctx context.Context,
	command CreateUserCommand,
) (*User, error) {
	ctx, observation := p.tracer.Begin(
		ctx,
		"UserService.CreateUser",
	)
	defer observation.End()

	var result *User

	err := p.transaction.WithinTransaction(
		ctx,
		TransactionOptions{},
		func(txCtx context.Context) error {
			var callErr error
			result, callErr = p.target.CreateUser(txCtx, command)
			return callErr
		},
	)
	if err != nil {
		p.metrics.RecordFailure("UserService.CreateUser")
		return nil, err
	}

	p.metrics.RecordSuccess("UserService.CreateUser")

	return result, nil
}
```

---

# 25. Interceptor ordering

The framework must define a deterministic default order.

Recommended default:

```text
1. Panic recovery
2. Timeout
3. Tracing
4. Metrics
5. Audit logging
6. Authorization
7. Rate limiting
8. Circuit breaker
9. Retry
10. Transaction
11. Target method
```

The exact ordering must be configurable globally.

Example configuration:

```yaml
interceptors:
  order:
    - recovery
    - timeout
    - tracing
    - metrics
    - authorization
    - circuit-breaker
    - retry
    - transaction
```

The compiler must detect incompatible interceptor combinations where possible.

Example:

```text
retry outside transaction
```

should be the default because each retry attempt should normally receive its own transaction.

---

# 26. Transaction management

## 26.1 Annotation

```go
// @Transactional
func (s *OrderService) CreateOrder(...) (*Order, error)
```

Arguments:

```text
readOnly       bool
isolation      string
propagation    string
timeout        duration
rollbackFor    []string
noRollbackFor  []string
```

## 26.2 Transaction manager interface

```go
type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		options TransactionOptions,
		callback func(context.Context) error,
	) error
}
```

## 26.3 Transaction options

```go
type TransactionOptions struct {
	ReadOnly    bool
	Isolation   IsolationLevel
	Propagation Propagation
	Timeout     time.Duration
}
```

## 26.4 Propagation modes

Initial support:

```text
required
requires_new
supports
not_supported
```

MVP may implement only:

```text
required
```

## 26.5 Transaction context

Repositories must retrieve the active transaction from `context.Context`.

The runtime must not store active transactions in global variables.

Example:

```go
type DBProvider interface {
	DB(ctx context.Context) DBTX
}
```

The `DBProvider` returns:

* current transaction when present;
* normal connection pool otherwise.

## 26.6 Rollback rules

Default behavior:

* rollback on any non-nil error;
* commit on nil error;
* panic causes rollback and is rethrown or mapped by recovery logic.

---

# 27. Repository generation

## 27.1 Repository interface

```go
// @Repository(entity="User", table="users")
type UserRepository interface {
	// @Query(`
	//   SELECT id, name, email, created_at
	//   FROM users
	//   WHERE id = :id
	// `)
	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*domain.User, error)

	// @Exec(`
	//   INSERT INTO users (
	//     id,
	//     name,
	//     email,
	//     created_at
	//   ) VALUES (
	//     :id,
	//     :name,
	//     :email,
	//     :createdAt
	//   )
	// `)
	Save(
		ctx context.Context,
		user *domain.User,
	) error
}
```

## 27.2 Repository modes

Two repository modes must exist.

### Component mode

The user writes the implementation.

```go
// @Repository
type PostgresUserRepository struct {
	db DBProvider
}
```

The framework only handles DI and proxies.

### Generated implementation mode

The user declares an annotated interface.

```go
// @Repository(generate=true)
type UserRepository interface{}
```

The framework generates the implementation.

## 27.3 Query annotations

```text
@Query
@Exec
@Batch
@Call
```

MVP:

```text
@Query
@Exec
```

## 27.4 Named SQL parameters

SQL may contain named parameters:

```sql
WHERE id = :id
AND organization_id = :organizationID
```

The generator must translate named parameters into driver-specific placeholders.

For PostgreSQL:

```sql
WHERE id = $1
AND organization_id = $2
```

## 27.5 Parameter resolution

Named query parameters may map to:

* method arguments;
* fields of method argument structs;
* explicitly defined expressions in a future version.

Example:

```go
Save(ctx context.Context, user *domain.User)
```

SQL:

```sql
INSERT INTO users(id, email)
VALUES (:user.ID, :user.Email)
```

MVP may limit parameter expressions to direct method argument names.

## 27.6 Return signatures

Supported query return signatures:

```go
(Entity, error)
(*Entity, error)
([]Entity, error)
([]*Entity, error)
(int64, error)
(bool, error)
```

Supported exec signatures:

```go
error
(int64, error)
(Result, error)
```

## 27.7 No rows behavior

For pointer entity returns:

```go
(*Entity, error)
```

`sql.ErrNoRows` must be mapped to a configurable not-found error.

For slices:

```go
([]Entity, error)
```

no rows must return an empty slice and nil error.

## 27.8 Row mapping

Initial implementation options:

1. Generate explicit `Scan` calls based on entity field order.
2. Require an explicit row mapper.
3. Integrate with a mapping library.

Recommended MVP approach:

```go
type RowMapper[T any] interface {
	MapRow(Row) (T, error)
}
```

Generated repositories may receive mappers.

A later release can generate scan code from annotations and struct tags.

## 27.9 SQL files

Queries should also support external files:

```go
// @Query(file="./queries/find_user_by_id.sql")
```

The compiler must include the file content during generation.

SQL files must participate in incremental build hashing.

## 27.10 SQL validation

MVP validation:

* named parameters exist;
* method parameters are used correctly;
* query and exec annotations match return signatures;
* duplicate named parameter resolution is valid.

Future validation:

* parse PostgreSQL SQL grammar;
* validate against a database schema;
* infer result columns;
* generate row scanning.

---

# 28. Configuration system

## 28.1 Configuration properties

```go
// @ConfigurationProperties(prefix="server")
type ServerProperties struct {
	Host            string        `config:"host" default:"0.0.0.0"`
	Port            int           `config:"port" default:"8080"`
	ReadTimeout     time.Duration `config:"read-timeout" default:"15s"`
	ShutdownTimeout time.Duration `config:"shutdown-timeout" default:"30s"`
}
```

## 28.2 Configuration sources

Default priority, from lowest to highest:

1. defaults;
2. configuration file;
3. profile-specific configuration file;
4. environment variables;
5. command-line arguments;
6. programmatic overrides.

## 28.3 Supported file formats

MVP:

```text
YAML
```

Future:

```text
JSON
TOML
```

## 28.4 Environment naming

Example property:

```text
server.read-timeout
```

Environment variable:

```text
SERVER_READ_TIMEOUT
```

## 28.5 Generated loaders

The framework should generate type-aware configuration loading code.

```go
func LoadServerProperties(
	source ConfigSource,
) (ServerProperties, error)
```

The loader must:

* apply defaults;
* parse values;
* report path-aware errors;
* validate required properties;
* optionally validate struct tags;
* reject unknown properties when strict mode is enabled.

## 28.6 Secret references

The core framework must not implement a secret store.

It should support secret resolver abstraction:

```go
type SecretResolver interface {
	Resolve(ctx context.Context, reference string) (string, error)
}
```

Configuration may contain:

```yaml
database:
  password: secret://vault/databases/users/password
```

Secret adapters may integrate with:

* environment variables;
* HashiCorp Vault;
* AWS Secrets Manager;
* Azure Key Vault;
* Google Secret Manager;
* Kubernetes Secrets.

---

# 29. Conditional components

## 29.1 Property condition

```go
// @ConditionalOnProperty(
//   name="cache.enabled",
//   havingValue="true",
//   matchIfMissing=false
// )
```

## 29.2 Bean conditions

```go
// @ConditionalOnBean(type="Cache")
```

```go
// @ConditionalOnMissingBean(type="Clock")
```

## 29.3 Profiles

```go
// @Profile(["production", "staging"])
```

## 29.4 Evaluation phase

Conditions depending only on static configuration may be evaluated during generated application startup.

The compiler must still validate all possible dependency graph branches where practical.

For MVP, conditional components may be delayed until version 0.2.

---

# 30. Lifecycle management

## 30.1 Lifecycle annotations

```go
// @PostConstruct
func (c *KafkaConsumer) Start(ctx context.Context) error
```

```go
// @PreDestroy
func (c *KafkaConsumer) Stop(ctx context.Context) error
```

## 30.2 Supported signatures

```go
func() error
func(context.Context) error
func()
func(context.Context)
```

## 30.3 Startup sequence

1. Load configuration.
2. Create infrastructure dependencies.
3. Create application components.
4. Execute post-construction hooks in dependency order.
5. Register HTTP routes and background workers.
6. Start application servers.
7. Mark application as ready.

## 30.4 Startup rollback

If a startup hook fails:

1. stop further startup;
2. invoke shutdown hooks for successfully initialized components;
3. use reverse initialization order;
4. return a wrapped startup error.

## 30.5 Shutdown sequence

1. Mark application as not ready.
2. Stop accepting new work.
3. Shut down HTTP listeners.
4. Stop consumers and workers.
5. Wait for in-flight work within timeout.
6. Invoke pre-destroy hooks.
7. Close databases, queues, and telemetry providers.

## 30.6 Shutdown timeout

Global default:

```text
30 seconds
```

Configurable through application configuration.

---

# 31. Application runtime

```go
type Application interface {
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
```

Generated implementation:

```go
type GeneratedApplication struct {
	server     *http.Server
	lifecycle  *runtime.Lifecycle
	components generatedComponents
}
```

`Run` must:

* start lifecycle;
* start servers and workers;
* wait for context cancellation or fatal component error;
* execute graceful shutdown;
* return the final error.

---

# 32. Generated application wiring

Example generated code:

```go
func NewApplication(
	options ...runtime.ApplicationOption,
) (*GeneratedApplication, error) {
	config, err := loadGeneratedConfiguration(options)
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	logger := provideLogger(config.Logging)

	database, err := configpkg.ProvideDatabase(config.Database)
	if err != nil {
		return nil, fmt.Errorf("provide database: %w", err)
	}

	transactionManager := pgxadapter.NewTransactionManager(database)
	dbProvider := pgxadapter.NewDBProvider(database)

	userRepository := generatedrepository.NewUserRepository(dbProvider)

	userServiceTarget := service.NewUserService(userRepository)

	userService := generatedproxy.NewUserServiceProxy(
		userServiceTarget,
		transactionManager,
		provideTracer(config.Tracing),
		provideMetrics(config.Metrics),
	)

	userController := controller.NewUserController(userService)

	httpDependencies := runtime.HTTPHandlerDependencies{
		Binder:         provideBinder(),
		Validator:      provideValidator(),
		Authorizer:     provideAuthorizer(),
		ErrorHandler:   provideErrorHandler(logger),
		ResponseWriter: provideResponseWriter(),
		Observer:       provideHTTPObserver(),
	}

	router := httpchi.NewRouter()

	registerUserControllerRoutes(
		router,
		userController,
		httpDependencies,
	)

	server := provideHTTPServer(config.Server, router)

	lifecycle := runtime.NewLifecycle(
		config.Server.ShutdownTimeout,
	)

	return &GeneratedApplication{
		server:    server,
		lifecycle: lifecycle,
	}, nil
}
```

---

# 33. Router abstraction

The compiler must generate router-independent route metadata.

```go
type Route struct {
	Method      string
	Path        string
	Name        string
	Consumes    []string
	Produces    []string
	Controller  ComponentID
	MethodName  string
	Annotations []annotation.Annotation
}
```

Router adapters convert route metadata into registration code.

## 33.1 Initial router

The first supported router should be Chi.

## 33.2 Standard library adapter

A `net/http` adapter should be supported when route parameter capabilities are sufficient.

## 33.3 Future adapters

```text
Gin
Echo
Fiber
```

Adapters must not leak router-specific APIs into controller business methods unless an explicit escape hatch is used.

---

# 34. Authorization

## 34.1 Annotation

```go
// @Authorize(roles=["users.read"], mode="any")
```

Arguments:

```text
roles       []string
permissions []string
mode        any | all
expression  string, future
```

## 34.2 Authorizer interface

```go
type Authorizer interface {
	Authorize(
		ctx context.Context,
		request AuthorizationRequest,
	) error
}
```

```go
type AuthorizationRequest struct {
	Roles       []string
	Permissions []string
	Mode        AuthorizationMode
	Resource    string
	Action      string
}
```

## 34.3 Principal

```go
type Principal interface {
	Subject() string
	Roles() []string
	Claims() map[string]any
}
```

The authenticated principal must be stored in request context through a typed context API.

## 34.4 Default behavior

Endpoints without security annotations must follow configurable policy:

```text
permit
authenticated
deny
```

The recommended production default is explicit policy configuration.

---

# 35. Observability

## 35.1 Tracing

```go
// @Traced(name="users.create")
```

Generated tracing must:

* create a span;
* attach component and method names;
* propagate context;
* record errors;
* close the span.

## 35.2 Metrics

```go
// @Timed(name="users.create.duration")
```

Generated metrics should include:

* call count;
* error count;
* duration;
* active calls.

High-cardinality arguments must not be included automatically.

## 35.3 Structured logging

```go
// @Logged(level="info")
```

Sensitive method arguments must not be logged by default.

Arguments may be explicitly exposed in a future version.

## 35.4 Audit

```go
// @Audit(action="user.create", resource="user")
```

Audit records must be sent through an interface:

```go
type AuditSink interface {
	Write(ctx context.Context, event AuditEvent) error
}
```

Audit sink failures must have a configurable policy:

```text
fail-open
fail-closed
```

---

# 36. Resilience

## 36.1 Retry

```go
// @Retry(
//   maxAttempts=3,
//   delay="100ms",
//   multiplier=2.0,
//   maxDelay="2s"
// )
```

Retry must respect context cancellation.

## 36.2 Timeout

```go
// @Timeout("2s")
```

The generated proxy creates a child context with timeout.

The target method must receive `context.Context` for proper cancellation.

## 36.3 Circuit breaker

```go
// @CircuitBreaker(name="payments")
```

The runtime must define an adapter-neutral interface.

## 36.4 Rate limiting

```go
// @RateLimit(name="users-read")
```

The key strategy must be supplied by the adapter or configuration.

## 36.5 MVP scope

Resilience annotations are not required in version 0.1.

The proxy architecture must nevertheless be designed so they can be added without breaking generated interfaces.

---

# 37. Compiler pipeline

## 37.1 Phase 1: configuration loading

Load:

* `goboot.yaml`;
* command-line flags;
* build tags;
* target package patterns;
* active plugins.

## 37.2 Phase 2: package loading

Use `golang.org/x/tools/go/packages`.

Required mode:

```go
packages.NeedName |
	packages.NeedFiles |
	packages.NeedCompiledGoFiles |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedSyntax |
	packages.NeedTypes |
	packages.NeedTypesInfo |
	packages.NeedModule
```

The loader must respect:

* Go modules;
* workspace files;
* build tags;
* platform constraints;
* test package inclusion configuration.

## 37.3 Phase 3: annotation scanning

The scanner must associate comments with:

* declarations;
* types;
* functions;
* methods;
* interfaces;
* fields where supported.

The scanner must preserve exact source positions.

## 37.4 Phase 4: annotation parsing

Parse comment content into annotation AST.

Multiple adjacent annotation comments must be supported.

Non-annotation documentation comments must be preserved but ignored by the annotation parser.

## 37.5 Phase 5: semantic analysis

Build:

* components;
* constructors;
* controllers;
* routes;
* advice handlers;
* repositories;
* configuration properties;
* lifecycle hooks;
* intercepted methods.

## 37.6 Phase 6: dependency resolution

Resolve constructor arguments and build the dependency graph.

## 37.7 Phase 7: validation

Run all core and plugin validators.

## 37.8 Phase 8: generation

Generate source files into an isolated temporary output directory.

## 37.9 Phase 9: formatting

Run:

```text
go/format
imports.Process
```

## 37.10 Phase 10: atomic output replacement

Generated files must replace previous output atomically.

Failed generation must not leave partially updated output.

## 37.11 Phase 11: optional compile verification

When enabled:

```bash
go test ./...
```

or:

```bash
go test <affected-packages>
```

---

# 38. Intermediate application model

```go
type Application struct {
	Name            string
	RootPackage     string
	Components      []*Component
	Controllers     []*Controller
	Repositories    []*Repository
	Configurations  []*Configuration
	Advice          []*Advice
	Routes          []*Route
	Graph           *graph.Graph
	Diagnostics     []Diagnostic
	Plugins         []PluginMetadata
}
```

The intermediate model must not depend directly on concrete router or database implementations.

Generators consume this model.

---

# 39. Diagnostics

## 39.1 Diagnostic structure

```go
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
	Position token.Position
	Notes    []DiagnosticNote
}
```

## 39.2 Severity levels

```text
info
warning
error
```

## 39.3 Error format

```text
internal/controller/user_controller.go:24:1:
GOBHTTP004: duplicate route GET /api/v1/users/{id}

Previously declared at:
internal/controller/admin_user_controller.go:17:1
```

## 39.4 Required diagnostic categories

```text
GOBANNxxx   annotation errors
GOBDIxxx    dependency injection errors
GOBHTTPxxx  HTTP controller errors
GOBREPxxx   repository errors
GOBCFGxxx   configuration errors
GOBLIFxxx   lifecycle errors
GOBPRXxxx   proxy errors
GOBPLGxxx   plugin errors
```

## 39.5 Strict mode

Strict mode converts selected warnings to errors.

---

# 40. Generated file layout

Recommended generated package:

```text
internal/generated/
├── zz_goboot_application.gen.go
├── zz_goboot_components.gen.go
├── zz_goboot_routes.gen.go
├── zz_goboot_proxies.gen.go
├── zz_goboot_repositories.gen.go
├── zz_goboot_configuration.gen.go
├── zz_goboot_lifecycle.gen.go
└── zz_goboot_metadata.gen.go
```

Large applications may use subpackages:

```text
internal/generated/
├── application/
├── components/
├── controllers/
├── proxies/
├── repositories/
└── configuration/
```

The generator must avoid import cycles.

---

# 41. Naming rules

Generated symbols must be:

* deterministic;
* collision-safe;
* valid Go identifiers;
* stable across builds.

Example:

```text
github.com/acme/users/internal/service.UserService
```

may produce:

```go
userService
newUserServiceProxy
```

When collisions occur, use a stable package-derived suffix.

```go
userService_service
userService_adminservice
```

Hash suffixes may be used only when readable resolution is impossible.

---

# 42. Incremental generation

## 42.1 Input hashing

Hash inputs:

* source files containing relevant annotations;
* package type information;
* configuration;
* SQL files;
* templates;
* plugin versions;
* compiler version.

## 42.2 Cache

Cache location:

```text
.goboot/cache/
```

Cache must not contain required source-of-truth data.

It must be safe to delete.

## 42.3 Generation strategy

MVP may perform full generation.

Incremental generation can be added after stable correctness is achieved.

---

# 43. CLI specification

## 43.1 Initialize project

```bash
goboot init
```

Creates:

```text
goboot.yaml
internal/generated/
example application declaration
go:generate directive
```

## 43.2 Generate

```bash
goboot generate ./...
```

Flags:

```text
--config
--output
--strict
--clean
--tags
--profile
--verify
--verbose
```

## 43.3 Validate

```bash
goboot validate ./...
```

Performs compilation and semantic validation without writing generated files.

## 43.4 Graph

```bash
goboot graph ./... --format mermaid
```

Formats:

```text
mermaid
dot
json
text
```

## 43.5 Doctor

```bash
goboot doctor
```

Checks:

* Go version;
* configuration validity;
* writable output directory;
* module structure;
* dependency versions;
* plugin compatibility;
* stale generated files.

## 43.6 Clean

```bash
goboot clean
```

Removes only files containing the generated marker and belonging to Goboot.

## 43.7 Version

```bash
goboot version
```

Displays:

* CLI version;
* compiler version;
* runtime compatibility version;
* Go version.

---

# 44. `go generate` integration

Recommended project directive:

```go
//go:generate go run github.com/<organization>/goboot/cmd/goboot generate ./...
```

Alternatively:

```go
//go:generate goboot generate ./...
```

The first approach provides stronger version pinning through `go.mod`.

---

# 45. Project configuration

Example `goboot.yaml`:

```yaml
version: v1

application:
  name: users-service
  packages:
    - ./cmd/users
    - ./internal/...

generation:
  output: ./internal/generated
  package: generated
  clean: true
  strict: true
  verifyCompile: true

http:
  adapter: chi
  defaultConsumes:
    - application/json
  defaultProduces:
    - application/json
  errorFormat: problem-json
  defaultSecurity: authenticated

database:
  adapter: pgx
  repositoryGeneration: true
  transactions: true

configuration:
  files:
    - application.yaml
    - application.${profile}.yaml
  environmentPrefix: USERS
  rejectUnknownFields: true

observability:
  tracing: otel
  metrics: prometheus
  logging: slog

lifecycle:
  shutdownTimeout: 30s

features:
  controllers: true
  repositories: true
  configurationProperties: true
  lifecycle: true
  transactions: true
  openapi: false
```

---

# 46. Plugin system

## 46.1 Plugin interface

```go
type Plugin interface {
	Name() string
	Version() string

	Annotations() []annotation.Definition

	Analyze(
		ctx context.Context,
		application *model.Application,
	) []model.Diagnostic

	Generate(
		ctx context.Context,
		application *model.Application,
		output generator.Output,
	) error
}
```

## 46.2 Plugin registration

Plugins should be compiled into the CLI through normal Go imports.

```go
func main() {
	compiler := goboot.NewCompiler(
		httpchi.Plugin(),
		pgxplugin.Plugin(),
		otelplugin.Plugin(),
	)

	os.Exit(compiler.Run(os.Args))
}
```

A custom CLI may be built by applications that require custom plugins.

## 46.3 External plugin processes

A future version may support external code generators through a versioned JSON protocol.

Dynamic `.so` Go plugins must not be the primary plugin model because of portability and version compatibility concerns.

## 46.4 Plugin isolation

Plugins:

* must not mutate core models without declared APIs;
* must return diagnostics rather than panic;
* must use deterministic output;
* must declare compatibility versions.

---

# 47. Runtime compatibility

Generated code and runtime package versions must be compatible.

The generated metadata must include:

```go
const GeneratedByGobootVersion = "0.1.0"
const RequiredRuntimeVersion = "0.1"
```

The runtime should expose a compile-time-compatible API where possible.

Breaking runtime changes require a new compatibility version.

---

# 48. Testing strategy

## 48.1 Unit tests

Required coverage areas:

* annotation lexer;
* annotation parser;
* schema validation;
* comment association;
* constructor discovery;
* interface implementation matching;
* qualifier resolution;
* cycle detection;
* route building;
* method signature validation;
* SQL named parameter parsing;
* code formatting;
* deterministic symbol naming.

## 48.2 Golden tests

Generators must use golden-file tests.

Input:

```text
internal/testdata/controller/basic/
```

Expected output:

```text
internal/testdata/controller/basic/golden/
```

Golden tests must compare complete generated source.

## 48.3 Compile tests

Each important example must compile.

Test command:

```bash
go test ./...
```

Generated test fixtures should include:

* valid applications;
* missing dependencies;
* cycles;
* duplicate routes;
* invalid annotations;
* ambiguous interfaces;
* transaction proxy errors.

## 48.4 Integration tests

Integration tests must cover:

* HTTP request to controller;
* validation failure;
* authorization failure;
* mapped domain error;
* internal server error;
* transaction commit;
* transaction rollback;
* lifecycle startup;
* lifecycle rollback;
* graceful shutdown.

## 48.5 Repository integration tests

Use ephemeral PostgreSQL instances where possible.

Test:

* query parameter binding;
* entity reads;
* no-row behavior;
* exec affected rows;
* transactions;
* rollback;
* SQL file loading.

## 48.6 Fuzz testing

Fuzz targets:

* annotation lexer;
* annotation parser;
* SQL named parameter parser;
* path template parser;
* configuration key parser.

The parsers must never panic on arbitrary input.

---

# 49. Quality requirements

## 49.1 Static analysis

CI must run:

```bash
go vet ./...
staticcheck ./...
golangci-lint run
```

## 49.2 Race detection

```bash
go test -race ./...
```

## 49.3 Formatting

```bash
gofmt
goimports
```

## 49.4 Test coverage

Recommended minimum:

```text
annotation parser:      90%
dependency resolver:    90%
graph algorithms:       90%
generators:             80%
runtime:                80%
overall:                80%
```

Coverage percentage alone is not sufficient; compile and golden tests are mandatory.

## 49.5 Performance targets

For a project containing approximately:

* 200 packages;
* 2,000 Go files;
* 500 components;
* 300 routes;

initial full generation should target:

```text
under 10 seconds on a modern development machine
```

Cached generation should target:

```text
under 2 seconds
```

These are design targets, not MVP release blockers.

---

# 50. Security requirements

The framework must:

* never log secrets by default;
* prevent generated SQL from concatenating untrusted parameters;
* use query placeholders;
* escape JSON responses through standard encoders;
* enforce request body size limits;
* support HTTP server timeouts;
* avoid exposing internal error messages in production;
* support panic recovery;
* preserve context cancellation;
* validate external SQL file paths;
* prevent generated output path traversal;
* avoid executing source code during analysis;
* treat annotation content as untrusted compiler input;
* avoid shell invocation where direct Go APIs are available.

Generated source code must not include secret configuration values.

---

# 51. Backward compatibility

## 51.1 Annotation versioning

Configuration must contain:

```yaml
version: v1
```

Annotation behavior must remain stable within the same major specification version.

## 51.2 Deprecation policy

Deprecated annotations should:

1. remain functional for at least one minor release cycle;
2. generate warnings;
3. include replacement guidance;
4. be removed only in a major release.

## 51.3 Generated code compatibility

Generated files do not require source compatibility across compiler versions.

The correct upgrade process is regeneration.

User-authored interfaces and runtime contracts should follow semantic versioning.

---

# 52. Documentation requirements

The project must include:

* getting started guide;
* installation guide;
* annotation reference;
* dependency injection guide;
* HTTP controller guide;
* repository guide;
* transaction guide;
* error handling guide;
* configuration guide;
* lifecycle guide;
* plugin authoring guide;
* migration guide;
* troubleshooting guide;
* generated code explanation;
* architecture decision records.

Each annotation reference must include:

* valid targets;
* argument schema;
* defaults;
* example;
* generated behavior;
* common errors.

---

# 53. Example application

## 53.1 Domain interface

```go
type UserUseCase interface {
	GetUser(
		ctx context.Context,
		id uuid.UUID,
	) (*User, error)

	CreateUser(
		ctx context.Context,
		command CreateUserCommand,
	) (*User, error)
}
```

## 53.2 Repository

```go
// @Repository(generate=true, table="users")
type UserRepository interface {
	// @Query(`
	//   SELECT id, name, email, created_at
	//   FROM users
	//   WHERE id = :id
	// `)
	FindByID(
		ctx context.Context,
		id uuid.UUID,
	) (*User, error)

	// @Exec(`
	//   INSERT INTO users(id, name, email, created_at)
	//   VALUES (:id, :name, :email, :createdAt)
	// `)
	Create(
		ctx context.Context,
		id uuid.UUID,
		name string,
		email string,
		createdAt time.Time,
	) error
}
```

## 53.3 Service

```go
// @Service(implements="UserUseCase")
type UserService struct {
	repository UserRepository
}

func NewUserService(
	repository UserRepository,
) *UserService {
	return &UserService{
		repository: repository,
	}
}
```

```go
func (s *UserService) GetUser(
	ctx context.Context,
	id uuid.UUID,
) (*User, error) {
	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	return user, nil
}
```

```go
// @Transactional
// @Traced(name="users.create")
// @Timed(name="users.create")
func (s *UserService) CreateUser(
	ctx context.Context,
	command CreateUserCommand,
) (*User, error) {
	user := NewUser(command.Name, command.Email)

	if err := s.repository.Create(
		ctx,
		user.ID,
		user.Name,
		user.Email,
		user.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}
```

## 53.4 Controller

```go
// @RestController
// @RequestMapping(path="/api/v1/users")
type UserController struct {
	service UserUseCase
}

func NewUserController(
	service UserUseCase,
) *UserController {
	return &UserController{
		service: service,
	}
}
```

```go
type GetUserRequest struct {
	ID uuid.UUID `path:"id" validate:"required"`
}
```

```go
// @GetMapping(path="/{id}")
// @Authorize(roles=["users.read"])
// @Response(status=200)
// @Response(status=404, error="user_not_found")
func (c *UserController) GetUser(
	ctx context.Context,
	request GetUserRequest,
) (*UserResponse, error) {
	user, err := c.service.GetUser(ctx, request.ID)
	if err != nil {
		return nil, err
	}

	return NewUserResponse(user), nil
}
```

```go
// @PostMapping(path="")
// @Authorize(roles=["users.write"])
// @Response(status=201)
// @Response(status=400, error="validation_error")
// @Response(status=409, error="email_already_exists")
func (c *UserController) CreateUser(
	ctx context.Context,
	request CreateUserRequest,
) (*UserResponse, error) {
	user, err := c.service.CreateUser(
		ctx,
		request.ToCommand(),
	)
	if err != nil {
		return nil, err
	}

	return NewUserResponse(user), nil
}
```

---

# 54. MVP scope

## 54.1 Version 0.1

Version 0.1 must implement:

### Annotation parser

* Java-style annotation syntax;
* named arguments;
* strings;
* integers;
* booleans;
* arrays;
* multiline annotations;
* source positions;
* schema validation.

### Compiler

* `go/packages` loading;
* AST scanning;
* type analysis;
* constructor discovery;
* dependency graph;
* cycle detection;
* deterministic diagnostics.

### Core annotations

```text
@Application
@Component
@Service
@Repository
@Configuration
@Bean
@RestController
@RequestMapping
@GetMapping
@PostMapping
@ControllerAdvice
@ExceptionHandler
@ConfigurationProperties
@PostConstruct
@PreDestroy
```

### Dependency injection

* singleton components;
* constructor injection;
* interface resolution;
* primary components;
* bean functions;
* generated application wiring.

### HTTP

* Chi adapter;
* GET and POST routes;
* path, query, header, and JSON body binding;
* validation abstraction;
* response serialization;
* centralized error handling;
* controller advice;
* panic recovery.

### Configuration

* YAML;
* environment overrides;
* defaults;
* typed configuration properties;
* strict unknown field mode.

### Lifecycle

* startup hooks;
* shutdown hooks;
* reverse-order shutdown;
* startup rollback;
* graceful HTTP shutdown.

### CLI

```text
init
generate
validate
graph
doctor
clean
version
```

### Testing

* unit tests;
* golden tests;
* compile tests;
* HTTP integration tests.

## 54.2 Explicit exclusions from version 0.1

Not included:

* generated SQL repositories;
* transactions;
* service proxies;
* retries;
* circuit breakers;
* authorization implementation;
* OpenTelemetry implementation;
* OpenAPI generation;
* multiple HTTP adapters;
* dynamic profiles;
* conditional beans;
* incremental generation.

The version 0.1 architecture must remain compatible with adding them later.

---

# 55. Version 0.2 scope

Version 0.2 should add:

```text
@Transactional
@Traced
@Timed
@Authorize
@Authenticated
@PermitAll
@Profile
@ConditionalOnProperty
@ConditionalOnBean
@ConditionalOnMissingBean
```

Additional capabilities:

* generated interface-based service proxies;
* pgx transaction adapter;
* authorization abstraction;
* OpenTelemetry tracing adapter;
* Prometheus metrics adapter;
* conditional component resolution;
* profiles;
* generated OpenAPI metadata.

---

# 56. Version 0.3 scope

Version 0.3 should add:

```text
@Query
@Exec
@Retry
@Timeout
@CircuitBreaker
@RateLimit
@Audit
```

Additional capabilities:

* generated pgx repository implementations;
* external SQL files;
* named SQL parameters;
* row mapper support;
* resilience proxy adapters;
* audit sink;
* repository integration testing toolkit.

---

# 57. Version 1.0 readiness criteria

Version 1.0 requires:

* stable annotation specification;
* stable compiler plugin API;
* stable runtime compatibility contract;
* deterministic generation;
* production-ready lifecycle handling;
* production-ready error model;
* transactions;
* repository generation;
* HTTP adapter abstraction;
* Chi adapter;
* OpenAPI support;
* configuration validation;
* security review;
* benchmark suite;
* migration documentation;
* at least three complete reference applications;
* no known critical correctness defects.

---

# 58. Implementation milestones

## Milestone 1: annotation language

Deliverables:

* lexer;
* parser;
* value model;
* schemas;
* diagnostics;
* fuzz tests.

Acceptance criteria:

* all supported syntax parses correctly;
* malformed annotations return source-aware errors;
* parser never panics on arbitrary input.

## Milestone 2: package scanner

Deliverables:

* `go/packages` loader;
* AST comment scanner;
* type lookup;
* declaration association.

Acceptance criteria:

* annotations are correctly associated with declarations;
* source positions are accurate;
* build tags are respected.

## Milestone 3: component model and DI

Deliverables:

* component discovery;
* constructor discovery;
* dependency resolver;
* graph;
* cycles;
* wiring generator.

Acceptance criteria:

* a multi-package example application compiles;
* ambiguous and missing dependencies produce useful diagnostics;
* generated order is deterministic.

## Milestone 4: HTTP controllers

Deliverables:

* controller analyzer;
* route model;
* Chi generator;
* request binder;
* response writer;
* error handler.

Acceptance criteria:

* GET and POST endpoints work;
* validation and mapped errors work;
* duplicate routes are rejected.

## Milestone 5: configuration and lifecycle

Deliverables:

* configuration property generator;
* YAML and environment loading;
* startup and shutdown hooks;
* graceful shutdown.

Acceptance criteria:

* configuration is type-safe;
* startup failure performs rollback;
* shutdown order is correct.

## Milestone 6: service proxy architecture

Deliverables:

* interface proxy analyzer;
* proxy generator;
* interceptor ordering;
* transaction abstraction.

Acceptance criteria:

* intercepted service interfaces compile;
* concrete proxy injection errors are detected;
* transaction wrapper behavior is tested.

## Milestone 7: repositories

Deliverables:

* repository interface analyzer;
* SQL annotation parser;
* pgx implementation generator;
* named parameter compiler.

Acceptance criteria:

* generated repositories compile;
* query and exec methods work against PostgreSQL;
* transaction context is respected.

## Milestone 8: production hardening

Deliverables:

* benchmarks;
* security review;
* documentation;
* examples;
* compatibility checks;
* plugin API.

---

# 59. Acceptance criteria

The project is accepted when the following scenario works.

A developer can:

1. Install or pin the Goboot CLI.
2. Add annotation comments to Go types and methods.
3. Run:

```bash
go generate ./...
```

4. Receive generated, formatted Go code.
5. Build the project with:

```bash
go build ./...
```

6. Run an HTTP application where:

* dependencies are injected automatically;
* routes are registered automatically;
* requests are bound and validated;
* controller errors are centrally mapped;
* lifecycle hooks are invoked;
* shutdown is graceful.

7. Receive compile-time diagnostics for:

* missing dependencies;
* circular dependencies;
* duplicate routes;
* invalid annotations;
* invalid controller signatures;
* ambiguous implementations.

8. Inspect all generated code without requiring a hidden runtime scanner.

For later releases, acceptance additionally requires:

* service methods wrapped by generated proxies;
* transactions committed and rolled back correctly;
* repositories generated from annotated interfaces;
* observability and security interceptors applied predictably.

---

# 60. Final architectural decision

The framework must be implemented as a compile-time application compiler, not as a runtime dependency injection container.

Annotations describe application intent.

The compiler validates this intent and converts it into a semantic application model.

Generators transform the model into ordinary Go code.

The runtime package provides only the minimal reusable abstractions required by generated code:

* lifecycle;
* HTTP binding;
* error handling;
* transactions;
* authorization;
* observability;
* resilience.

This approach provides a Spring Boot–like development experience while retaining Go’s static type system, transparency, fast startup, predictable behavior, and compatibility with the standard Go toolchain.

