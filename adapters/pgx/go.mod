module github.com/zombocoder/goboot/adapters/pgx

go 1.25.0

require (
	github.com/jackc/pgx/v5 v5.7.2
	github.com/zombocoder/goboot v0.1.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.21.0 // indirect
)

// In-repo development resolves the core from this checkout; released consumers
// ignore this replace and fetch the required version above.
replace github.com/zombocoder/goboot => ../..
