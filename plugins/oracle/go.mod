module github.com/zombocoder/goboot/plugins/oracle

go 1.25.0

require github.com/zombocoder/goboot v0.0.0

// In-repo development: resolve the core from the parent checkout. Consumers get
// the tagged version via the require above; this replace is ignored downstream
// (replace directives only apply to a build's main module).
replace github.com/zombocoder/goboot => ../..
