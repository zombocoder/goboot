// Package sqlgen compiles named-parameter SQL into driver-specific positional
// SQL (§27.4). It is the seam at which database drivers differ: everything else
// in the repository pipeline is driver-neutral, and a driver adapter or plugin
// contributes only a Dialect. The package is build-time only and imports no
// driver.
package sqlgen

import "strconv"

// Dialect renders a positional placeholder for a database driver's SQL syntax.
type Dialect interface {
	// Name identifies the dialect, e.g. "postgres".
	Name() string
	// Placeholder renders the placeholder for the given 1-based position.
	Placeholder(index int) string
}

// postgresDialect renders $1, $2, ... (PostgreSQL, pgx).
type postgresDialect struct{}

func (postgresDialect) Name() string             { return "postgres" }
func (postgresDialect) Placeholder(i int) string { return "$" + strconv.Itoa(i) }

// questionDialect renders ? (MySQL, SQLite).
type questionDialect struct{}

func (questionDialect) Name() string           { return "question" }
func (questionDialect) Placeholder(int) string { return "?" }

var (
	// Postgres is the default dialect: $1, $2, ...
	Postgres Dialect = postgresDialect{}
	// Question uses ? placeholders, for MySQL- and SQLite-style drivers.
	Question Dialect = questionDialect{}
)

// DialectByName returns a built-in dialect by name and whether it exists.
func DialectByName(name string) (Dialect, bool) {
	switch name {
	case "", "postgres":
		return Postgres, true
	case "question", "mysql", "sqlite":
		return Question, true
	default:
		return nil, false
	}
}
