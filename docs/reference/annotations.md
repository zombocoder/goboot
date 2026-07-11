# Annotation reference

Annotations are `// @Name(arg=value, ...)` doc comments attached to a declaration.
Values may be strings, ints, floats, bools, `null`, identifiers (`singleton`,
`TimeUnit.MINUTES`), arrays `[...]`, or objects `{k=v}`. Raw strings use
backticks (`` @Query(`SELECT ...`) ``).

!!! note
    An annotation must be the entire comment line — a name mentioned in prose
    (e.g. "the `@Transactional` method") is not parsed as an annotation.

## Components & DI

| Annotation | Target | Key arguments |
| ---------- | ------ | ------------- |
| `@Application` | struct | `name` (req), `scan`, `profiles`, `configuration` |
| `@Service` | struct | `name`, `scope` (singleton\|prototype), `implements` |
| `@Component` | struct | `name` |
| `@Configuration` | struct | — |
| `@Nut` | func/method | `name` |
| `@Primary` | struct/func | — |
| `@Named` | struct/func | positional string |
| `@Scope` | struct | positional (singleton\|prototype) |

## HTTP

| Annotation | Target | Key arguments |
| ---------- | ------ | ------------- |
| `@RestController` | struct | — |
| `@RequestMapping` | struct | `path`, `host`, `headers` |
| `@GetMapping` `@PostMapping` `@PutMapping` `@PatchMapping` `@DeleteMapping` | method | `path`, `name`, `consumes`, `produces`, `timeout`, `status` |
| `@Response` | method (repeatable) | `status`, `type`, `error`, `contentType` |
| `@ResponseStatus` | method | positional int |
| `@Consumes` / `@Produces` | method | positional `[]string` |
| `@ControllerAdvice` | struct | — |
| `@ExceptionHandler` | method | `type` (optional; caught type read from the 2nd param) |

## Repositories

| Annotation | Target | Notes |
| ---------- | ------ | ----- |
| `@Repository` | struct or interface | `name`, `entity`, `table`, `generate` |
| `@Query` | interface method | positional SQL (`:name`, `:arg.Field`) |
| `@Exec` | interface method | positional SQL |
| `@Batch` | interface method | runs per slice element |
| `@Call` | interface method | stored procedure/function |

## Interception (service proxies)

Require `@Service(implements="Iface")`. Chain order: timeout → tracing → logging
→ audit → metrics → bulkhead → circuit breaker → rate limit → authorize → retry →
transaction → target.

| Annotation | Key arguments |
| ---------- | ------------- |
| `@Transactional` | `readOnly`, `isolation`, `propagation`, `timeout` |
| `@Traced` | `name` |
| `@Timed` | `name` |
| `@Logged` | `level` (debug\|info\|warn\|error) |
| `@Audit` | `action`, `resource` |
| `@Timeout` | positional duration (`"2s"`) |
| `@Retry` | `maxAttempts`, `delay`, `multiplier`, `maxDelay` |
| `@CircuitBreaker` | `name`, `failureThreshold`, `resetTimeout`, `halfOpenMax` |
| `@RateLimit` | `name`, `limit`, `period`, `burst` |
| `@Bulkhead` | `name`, `maxConcurrent`, `maxWait` |
| `@Authorize` | `roles`, `permissions`, `mode` (any\|all) |
| `@RolesAllowed` | positional `[]string` |

## Config, lifecycle & scheduling

| Annotation | Target | Key arguments |
| ---------- | ------ | ------------- |
| `@ConfigurationProperties` | struct | `prefix` (req) |
| `@PostConstruct` / `@PreDestroy` | method | — |
| `@Scheduled` | method | `fixedRate`, `fixedDelay`, `initialDelay`, `timeUnit` |

## Conditions & profiles

| Annotation | Key arguments |
| ---------- | ------------- |
| `@Profile` | positional `[]string` |
| `@ConditionalOnProperty` | `name` (req), `havingValue`, `matchIfMissing` |
| `@ConditionalOnNut` / `@ConditionalOnMissingNut` | `type` (req) |
