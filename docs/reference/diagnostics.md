# Diagnostics

goboot reports problems as source-positioned diagnostics with **stable codes** —
never panics. Warnings are advisory; `-strict` promotes them to errors.

| Prefix | Area |
| ------ | ---- |
| `GOBANN*` | annotation syntax / schema (unknown annotation, wrong target, bad argument, non-repeatable) |
| `GOBDI*` | dependency injection (missing dependency, ambiguous dependency, cycle, invalid constructor) |
| `GOBHTTP*` | HTTP (invalid handler signature, duplicate route, invalid `@ExceptionHandler`) |
| `GOBREP*` | repositories (invalid query signature, unknown SQL parameter, missing `@Query`/`@Exec`) |
| `GOBCFG*` / `GOBLIF*` | configuration / lifecycle |
| `GOBPRX*` | service proxies (concrete injection of a proxied service, missing/unimplemented interface) |
| `GOBSCH*` | scheduling |
| `GOBPLG*` | plugins |

## Common ones

| Code | Meaning | Fix |
| ---- | ------- | --- |
| `GOBANN002` | unknown annotation (warning) | typo, or register it via a plugin `AnnotationProvider` |
| `GOBANN003` | annotation on the wrong target | move it to a valid declaration kind |
| `GOBDI001` | missing dependency | provide a component/constructor for the type |
| `GOBDI` (ambiguous) | more than one candidate | add `@Primary` or `@Named` |
| `GOBDI` (cycle) | dependency cycle | break the cycle (introduce an interface / rethink ownership) |
| `GOBHTTP004` | duplicate route | two handlers map to the same method + path |
| `GOBPRX001` | concrete injection of a proxied service | inject the interface it declares via `implements=` |
| `GOBREP002` | unknown SQL parameter | a `:name` has no matching method argument |

Run `goboot validate ./...` to see all diagnostics without writing files.
