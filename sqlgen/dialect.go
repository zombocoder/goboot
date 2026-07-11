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

// questionDialect renders ? (the generic positional-placeholder style, SQLite).
type questionDialect struct{}

func (questionDialect) Name() string           { return "question" }
func (questionDialect) Placeholder(int) string { return "?" }

// mysqlDialect renders ? placeholders (MySQL, go-sql-driver). It is a distinct
// named dialect from the generic question style for clearer diagnostics.
type mysqlDialect struct{}

func (mysqlDialect) Name() string           { return "mysql" }
func (mysqlDialect) Placeholder(int) string { return "?" }

// sqlServerDialect renders @p1, @p2, ... (Microsoft SQL Server).
type sqlServerDialect struct{}

func (sqlServerDialect) Name() string             { return "sqlserver" }
func (sqlServerDialect) Placeholder(i int) string { return "@p" + strconv.Itoa(i) }

var (
	// Postgres is the default dialect: $1, $2, ...
	Postgres Dialect = postgresDialect{}
	// Question uses ? placeholders, the generic positional style (SQLite).
	Question Dialect = questionDialect{}
	// MySQL uses ? placeholders (MySQL, go-sql-driver).
	MySQL Dialect = mysqlDialect{}
	// SQLServer uses @p1, @p2, ... placeholders (Microsoft SQL Server).
	SQLServer Dialect = sqlServerDialect{}
)

// DialectByName returns a built-in dialect by name and whether it exists.
func DialectByName(name string) (Dialect, bool) {
	switch name {
	case "", "postgres":
		return Postgres, true
	case "question", "sqlite":
		return Question, true
	case "mysql":
		return MySQL, true
	case "sqlserver", "mssql":
		return SQLServer, true
	default:
		return nil, false
	}
}
