module github.com/zombocoder/goboot/plugins/lint

go 1.25.0

require github.com/zombocoder/goboot v0.1.0

require (
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

// In-repo development resolves the core from this checkout; released consumers
// ignore this replace and fetch the required version above.
replace github.com/zombocoder/goboot => ../..
