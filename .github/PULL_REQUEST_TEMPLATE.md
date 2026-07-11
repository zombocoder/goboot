<!-- Thanks for contributing to goboot! -->

## Summary

<!-- What does this change do and why? Reference the spec section(s) if applicable, e.g. §24.4. -->

## Type of change

- [ ] Bug fix
- [ ] New feature
- [ ] Refactor / internal change
- [ ] Documentation
- [ ] Other:

## Checklist

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes
- [ ] `gofmt -l .` prints nothing
- [ ] `go test -race ./...` passes
- [ ] If a generator changed: golden files and committed `internal/*e2e` wiring were regenerated (not hand-edited)
- [ ] Added/updated tests (unit, golden/compile, integration, and/or fuzz as appropriate)
- [ ] Updated documentation where relevant
- [ ] Preserved the core invariants (compile-time only, `go/types`-based, deterministic output, diagnostics-not-panics, adapters out of core)

## Notes for reviewers

<!-- Anything that needs special attention, trade-offs, or follow-ups. -->
