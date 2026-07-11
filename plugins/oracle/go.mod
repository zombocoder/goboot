module github.com/zombocoder/goboot/plugins/oracle

go 1.25.0

require github.com/zombocoder/goboot v0.1.0

// In-repo development resolves the core from this checkout; released consumers
// ignore this replace and fetch the required version above.
replace github.com/zombocoder/goboot => ../..
