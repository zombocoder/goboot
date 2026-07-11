package sqlgen

import (
	"strings"
	"testing"
)

func TestCompilePostgres(t *testing.T) {
	c := Compile("SELECT id FROM users WHERE id = :id AND org = :orgID", Postgres)
	if c.SQL != "SELECT id FROM users WHERE id = $1 AND org = $2" {
		t.Errorf("SQL = %q", c.SQL)
	}
	if len(c.Params) != 2 || c.Params[0] != "id" || c.Params[1] != "orgID" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestCompileQuestion(t *testing.T) {
	c := Compile("SELECT id FROM users WHERE id = :id AND org = :orgID", Question)
	if c.SQL != "SELECT id FROM users WHERE id = ? AND org = ?" {
		t.Errorf("SQL = %q", c.SQL)
	}
}

func TestCompileSQLServer(t *testing.T) {
	c := Compile("SELECT id FROM users WHERE id = :id AND org = :orgID", SQLServer)
	if c.SQL != "SELECT id FROM users WHERE id = @p1 AND org = @p2" {
		t.Errorf("SQL = %q", c.SQL)
	}
	if len(c.Params) != 2 || c.Params[0] != "id" || c.Params[1] != "orgID" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestDialectByNameSQLServer(t *testing.T) {
	for _, name := range []string{"sqlserver", "mssql"} {
		d, ok := DialectByName(name)
		if !ok || d.Name() != "sqlserver" {
			t.Errorf("DialectByName(%q) = %v, %v; want the sqlserver dialect", name, d, ok)
		}
	}
}

func TestCompileMySQL(t *testing.T) {
	c := Compile("SELECT id FROM users WHERE id = :id AND org = :orgID", MySQL)
	if c.SQL != "SELECT id FROM users WHERE id = ? AND org = ?" {
		t.Errorf("SQL = %q", c.SQL)
	}
	if d, ok := DialectByName("mysql"); !ok || d.Name() != "mysql" {
		t.Errorf("DialectByName(mysql) = %v, %v; want the mysql dialect", d, ok)
	}
}

func TestCompileRepeatedParam(t *testing.T) {
	// A name used twice produces two placeholders and two param entries.
	c := Compile("WHERE a = :id OR b = :id", Postgres)
	if c.SQL != "WHERE a = $1 OR b = $2" {
		t.Errorf("SQL = %q", c.SQL)
	}
	if len(c.Params) != 2 || c.Params[0] != "id" || c.Params[1] != "id" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestCompileFieldReference(t *testing.T) {
	c := Compile("INSERT INTO users(id, email) VALUES (:user.ID, :user.Email)", Postgres)
	if c.SQL != "INSERT INTO users(id, email) VALUES ($1, $2)" {
		t.Errorf("SQL = %q", c.SQL)
	}
	if c.Params[0] != "user.ID" || c.Params[1] != "user.Email" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestCompileIgnoresStringLiterals(t *testing.T) {
	// A colon inside a string literal must not be treated as a parameter.
	c := Compile("SELECT ':not_a_param' AS x WHERE id = :id", Postgres)
	if !strings.Contains(c.SQL, "':not_a_param'") {
		t.Errorf("string literal altered: %q", c.SQL)
	}
	if len(c.Params) != 1 || c.Params[0] != "id" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestCompileEscapedQuote(t *testing.T) {
	// '' is an escaped quote; the parameter after the string must still bind.
	c := Compile("SELECT 'O''Brien' WHERE id = :id", Postgres)
	if len(c.Params) != 1 || c.Params[0] != "id" {
		t.Errorf("params = %v, sql=%q", c.Params, c.SQL)
	}
}

func TestCompilePostgresCast(t *testing.T) {
	// :: is a cast, not a parameter.
	c := Compile("SELECT id::text WHERE id = :id", Postgres)
	if !strings.Contains(c.SQL, "id::text") {
		t.Errorf("cast altered: %q", c.SQL)
	}
	if len(c.Params) != 1 || c.Params[0] != "id" {
		t.Errorf("params = %v", c.Params)
	}
}

func TestCompileNoParams(t *testing.T) {
	c := Compile("SELECT COUNT(*) FROM users", Postgres)
	if c.SQL != "SELECT COUNT(*) FROM users" || len(c.Params) != 0 {
		t.Errorf("unexpected compile: %q %v", c.SQL, c.Params)
	}
}

func TestDialectByName(t *testing.T) {
	for _, name := range []string{"", "postgres", "question", "mysql", "sqlite"} {
		if _, ok := DialectByName(name); !ok {
			t.Errorf("dialect %q should resolve", name)
		}
	}
	if _, ok := DialectByName("oracle"); ok {
		t.Errorf("unknown dialect should not resolve")
	}
}

func FuzzCompile(f *testing.F) {
	seeds := []string{
		"", ":", "::", ":id", "SELECT :a, :b.c FROM t WHERE x = ':lit'",
		"'unterminated :id", ":::", ":1abc", "a:b:c", "WHERE x = :id::text",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, sql string) {
		// Must never panic, for either dialect.
		_ = Compile(sql, Postgres)
		_ = Compile(sql, Question)
	})
}
