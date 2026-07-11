# HTTP controllers

A `@RestController` becomes an HTTP component; its mapping methods become routes.
goboot generates a handler per route that binds the request, validates it,
authorizes, invokes your method, and writes the response — plus centralized
RFC-7807 error handling.

```go
// @RestController
// @RequestMapping(path="/widgets")
type WidgetController struct{ svc WidgetUseCase }

// @GetMapping(path="/{id}")
func (c *WidgetController) Get(ctx context.Context, req GetRequest) (*Widget, error) {
    return c.svc.Get(ctx, req.ID)
}
```

## Verbs & status

`@GetMapping` `@PostMapping` `@PutMapping` `@PatchMapping` `@DeleteMapping`.
Default success statuses: GET/PUT/PATCH `200`, POST `201`, DELETE `204` (no body).
Override with `@ResponseStatus(202)` or the mapping's `status` argument.

## Request binding

goboot binds fields of the request struct from the request via tags:

| Tag | Source |
| --- | ------ |
| `path:"id"` | path parameter (`/{id}`) |
| `query:"expand"` | query string |
| `header:"X-Request-ID"` | header |
| `cookie:"session"` | cookie |
| `json:"name"` | JSON body |

```go
type UpdateRequest struct {
    ID    string `path:"id"`
    Title string `json:"title"`
    Force bool   `query:"force"`
}
```

## Handler signatures

The first parameter is `context.Context`; the last result is `error`. Supported
forms: `(ctx, req) (*Res, error)`, `(ctx) (*Res, error)`, `(ctx, req) error`,
`(ctx) error`.

## Errors

Return an `error`; goboot maps it to an RFC-7807 `Problem` body. Use
`runtime.NewError(status, code, message)` to control the status/code:

```go
return nil, runtime.NewError(404, "widget_not_found", "no such widget")
```

For typed error mapping, add a `@ControllerAdvice` with `@ExceptionHandler`
methods:

```go
// @ControllerAdvice
type Advice struct{}

// @ExceptionHandler
func (a *Advice) NotFound(ctx context.Context, err *NotFoundError) error {
    return runtime.NewError(404, "not_found", err.Error())
}
```

The caught type is the handler's second parameter (matched via `errors.As`); an
`err error` parameter is a catch-all, tried after concrete handlers.

## Content negotiation

`@Consumes`/`@Produces` (or the `consumes`/`produces` mapping arguments) enforce
media types: an unsupported request `Content-Type` yields **415**, an unacceptable
`Accept` yields **406**, both before binding.

## Wiring

```go
mux := http.NewServeMux()
generated.RegisterRoutes(mux, components, runtime.DefaultHTTPHandlerDependencies())
```

Or let `generated.NewApplication(...)` build the mux and server for you.
