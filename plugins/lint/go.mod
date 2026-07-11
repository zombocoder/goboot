module github.com/zombocoder/goboot/plugins/lint

go 1.25.0

require github.com/zombocoder/goboot v0.0.0

require (
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// In-repo development: resolve the core from the parent checkout. Consumers get
// the tagged version via the require above; this replace is ignored downstream.
replace github.com/zombocoder/goboot => ../..
