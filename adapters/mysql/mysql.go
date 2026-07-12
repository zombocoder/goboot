// Package mysql binds goboot's driver-neutral db abstraction to MySQL via
// go-sql-driver/mysql (§27). Because that is a standard database/sql driver, the
// provider and transaction manager reuse the databasesql adapter; this package
// adds the driver registration and a DSN helper. Pair it with the `mysql` SQL
// dialect (`goboot generate -dialect mysql`).
//
// Wire it into the composition root:
//
//	pool, err := mysql.Open("user:pass@tcp(localhost:3306)/app")
//	dbProvider := mysql.NewProvider(pool)
//	proxyDeps.Transactions = mysql.NewTransactionManager(pool)
//
// It is a separate module so the MySQL driver stays out of the core.
package mysql

import (
	"database/sql"
	"fmt"
	"time"

	driver "github.com/go-sql-driver/mysql"
	"github.com/zombocoder/goboot/adapters/databasesql"
	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

// Option tunes the connection pool Open returns. Without any options the pool
// keeps database/sql's defaults — no magic numbers are imposed.
type Option func(*sql.DB)

// WithMaxOpenConns caps the number of open connections (0 = unlimited).
func WithMaxOpenConns(n int) Option { return func(p *sql.DB) { p.SetMaxOpenConns(n) } }

// WithMaxIdleConns caps the idle connection pool.
func WithMaxIdleConns(n int) Option { return func(p *sql.DB) { p.SetMaxIdleConns(n) } }

// WithConnMaxLifetime bounds how long a connection may be reused.
func WithConnMaxLifetime(d time.Duration) Option { return func(p *sql.DB) { p.SetConnMaxLifetime(d) } }

// WithConnMaxIdleTime bounds how long a connection may sit idle.
func WithConnMaxIdleTime(d time.Duration) Option { return func(p *sql.DB) { p.SetConnMaxIdleTime(d) } }

// Open opens a MySQL connection pool for dsn (go-sql-driver format,
// "user:pass@tcp(host:port)/dbname?params"). It forces parseTime so DATETIME /
// TIMESTAMP columns scan into time.Time — which generated repositories rely on —
// and applies any pool options. It does not verify connectivity; call pool.Ping.
func Open(dsn string, opts ...Option) (*sql.DB, error) {
	cfg, err := driver.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: parsing DSN: %w", err)
	}
	cfg.ParseTime = true
	connector, err := driver.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("mysql: building connector: %w", err)
	}
	pool := sql.OpenDB(connector)
	for _, o := range opts {
		o(pool)
	}
	return pool, nil
}

// NewProvider returns a db.DBProvider backed by the MySQL pool. It joins the
// active transaction from the context when one is present.
func NewProvider(pool *sql.DB) db.DBProvider {
	return databasesql.NewProvider(pool)
}

// NewTransactionManager returns a runtime.TransactionManager over the pool, so
// @Transactional methods run inside a MySQL transaction.
func NewTransactionManager(pool *sql.DB) goruntime.TransactionManager {
	return databasesql.NewTransactionManager(pool)
}
