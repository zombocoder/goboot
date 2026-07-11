# CLI reference

```bash
go install github.com/zombocoder/goboot/cmd/goboot@latest
```

## Commands

| Command | Description |
| ------- | ----------- |
| `goboot init` | scaffold a `goboot.yaml` |
| `goboot generate ./...` | generate wiring (+ plugin artifacts) into the output package |
| `goboot validate ./...` | analyze and print diagnostics; write nothing |
| `goboot graph ./... --format mermaid` | print the dependency graph (`text`\|`mermaid`\|`dot`\|`json`) |
| `goboot plugins` | list configured vs. linked plugins (`plugins sync` writes a tool main) |
| `goboot clean` | remove generated files |
| `goboot doctor` | environment checks |
| `goboot version` | print version + runtime compatibility |

## Flags (generate / validate)

| Flag | Meaning |
| ---- | ------- |
| `-dir <path>` | working directory (contains `go.mod` + `goboot.yaml`) |
| `-output <dir>` | output directory (overrides `goboot.yaml`) |
| `-package <name>` | generated package name |
| `-dialect <name>` | `postgres` (default), `mysql`, `sqlserver`, `question` |
| `-profile a,b` | active profiles for `@Profile` / conditionals |
| `-property k=v` | property values for `@ConditionalOnProperty` |
| `-strict` | treat warnings as errors |
| `-tags <list>` | build tags |
| `-clean` | remove existing generated files first |

## goboot.yaml

```yaml
application:
  name: my-service
  packages:
    - ./internal/...
generation:
  output: internal/generated
  package: generated
  clean: true
  strict: false
  dialect: postgres
plugins:
  - github.com/zombocoder/goboot/plugins/openapi@v0.1.0
```

## Environment

| Variable | Effect |
| -------- | ------ |
| `GOBOOT_BOOTSTRAP=off` | disable plugin self-bootstrap (run the plugin-free binary) |
