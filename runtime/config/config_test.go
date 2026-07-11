package config

import (
	"testing"
	"time"
)

type ServerProperties struct {
	Host            string        `config:"host" default:"0.0.0.0"`
	Port            int           `config:"port" default:"8080"`
	ReadTimeout     time.Duration `config:"read-timeout" default:"15s"`
	ShutdownTimeout time.Duration `config:"shutdown-timeout" default:"30s"`
	Debug           bool          `config:"debug"`
	Tags            []string      `config:"tags"`
}

func TestBindDefaults(t *testing.T) {
	var p ServerProperties
	if err := Bind("server", MapSource{}, &p); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if p.Host != "0.0.0.0" || p.Port != 8080 {
		t.Errorf("defaults not applied: %+v", p)
	}
	if p.ReadTimeout != 15*time.Second || p.ShutdownTimeout != 30*time.Second {
		t.Errorf("duration defaults wrong: %+v", p)
	}
}

func TestBindOverrides(t *testing.T) {
	src := MapSource{
		"server.host":         "127.0.0.1",
		"server.port":         "9090",
		"server.read-timeout": "5s",
		"server.debug":        "true",
		"server.tags":         "a, b ,c",
	}
	var p ServerProperties
	if err := Bind("server", src, &p); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if p.Host != "127.0.0.1" || p.Port != 9090 || p.ReadTimeout != 5*time.Second || !p.Debug {
		t.Errorf("overrides not applied: %+v", p)
	}
	if len(p.Tags) != 3 || p.Tags[0] != "a" || p.Tags[2] != "c" {
		t.Errorf("list binding wrong: %+v", p.Tags)
	}
}

func TestBindLayeredPrecedence(t *testing.T) {
	file := MapSource{"server.port": "8080", "server.host": "file-host"}
	env := EnvSource{Prefix: "APP", Getenv: func(k string) string {
		if k == "APP_SERVER_PORT" {
			return "9999"
		}
		return ""
	}}
	// Env overrides file (env listed first).
	src := Layered{env, file}
	var p ServerProperties
	if err := Bind("server", src, &p); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if p.Port != 9999 {
		t.Errorf("env should override file: port = %d", p.Port)
	}
	if p.Host != "file-host" {
		t.Errorf("file value should apply when env absent: host = %q", p.Host)
	}
}

func TestEnvName(t *testing.T) {
	if got := EnvName("USERS", "server.read-timeout"); got != "USERS_SERVER_READ_TIMEOUT" {
		t.Errorf("EnvName = %q", got)
	}
	if got := EnvName("", "server.port"); got != "SERVER_PORT" {
		t.Errorf("EnvName no prefix = %q", got)
	}
}

func TestFlattenYAML(t *testing.T) {
	yaml := []byte(`
server:
  host: 0.0.0.0
  port: 8080
  read-timeout: 15s
  tags:
    - api
    - web
database:
  url: postgres://localhost
`)
	src, err := FlattenYAML(yaml)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	checks := map[string]string{
		"server.host":         "0.0.0.0",
		"server.port":         "8080",
		"server.read-timeout": "15s",
		"server.tags":         "api,web",
		"database.url":        "postgres://localhost",
	}
	for k, want := range checks {
		if got, _ := src.Get(k); got != want {
			t.Errorf("flattened %q = %q, want %q", k, got, want)
		}
	}
}

func TestBindFromYAML(t *testing.T) {
	src, err := FlattenYAML([]byte("server:\n  host: example.com\n  port: 443\n  tags: [x, y]\n"))
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	var p ServerProperties
	if err := Bind("server", src, &p); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if p.Host != "example.com" || p.Port != 443 || len(p.Tags) != 2 {
		t.Errorf("yaml bind wrong: %+v", p)
	}
}

func TestBindNestedStruct(t *testing.T) {
	type DB struct {
		URL  string `config:"url"`
		Pool int    `config:"pool" default:"10"`
	}
	type App struct {
		Name     string `config:"name"`
		Database DB     `config:"database"`
	}
	src := MapSource{"app.name": "svc", "app.database.url": "postgres://x"}
	var a App
	if err := Bind("app", src, &a); err != nil {
		t.Fatalf("bind: %v", err)
	}
	if a.Name != "svc" || a.Database.URL != "postgres://x" || a.Database.Pool != 10 {
		t.Errorf("nested bind wrong: %+v", a)
	}
}

func TestBindRequired(t *testing.T) {
	type C struct {
		Key string `config:"key" required:"true"`
	}
	var c C
	err := Bind("", MapSource{}, &c)
	if err == nil {
		t.Fatal("expected error for missing required property")
	}
}

func TestBindInvalidValue(t *testing.T) {
	type C struct {
		Port int `config:"port"`
	}
	var c C
	if err := Bind("", MapSource{"port": "notanumber"}, &c); err == nil {
		t.Fatal("expected error for invalid integer")
	}
}

func TestBindRejectsNonPointer(t *testing.T) {
	if err := Bind("", MapSource{}, ServerProperties{}); err == nil {
		t.Fatal("expected error binding to non-pointer")
	}
}
