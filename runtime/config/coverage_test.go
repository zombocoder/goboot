package config

import (
	"strings"
	"testing"
	"time"
)

// allTypes exercises every setField conversion branch.
type allTypes struct {
	Name    string        `config:"name"`
	Enabled bool          `config:"enabled"`
	Count   int           `config:"count"`
	Big     int64         `config:"big"`
	Workers uint          `config:"workers"`
	Ratio   float64       `config:"ratio"`
	Timeout time.Duration `config:"timeout"`
	Tags    []string      `config:"tags"`
	Nested  struct {
		Port int `config:"port"`
	} `config:"nested"`
}

func TestBindAllScalarTypes(t *testing.T) {
	src := MapSource{
		"name":        "svc",
		"enabled":     "true",
		"count":       "7",
		"big":         "9000000000",
		"workers":     "4",
		"ratio":       "2.5",
		"timeout":     "3s",
		"tags":        "a, b ,,c",
		"nested.port": "8080",
	}
	var c allTypes
	if err := Bind("", src, &c); err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if c.Name != "svc" || !c.Enabled || c.Count != 7 || c.Big != 9_000_000_000 ||
		c.Workers != 4 || c.Ratio != 2.5 || c.Timeout != 3*time.Second ||
		c.Nested.Port != 8080 {
		t.Errorf("bound = %+v", c)
	}
	if len(c.Tags) != 3 || c.Tags[0] != "a" || c.Tags[2] != "c" {
		t.Errorf("tags = %v", c.Tags)
	}
}

func TestBindConversionErrors(t *testing.T) {
	cases := map[string]MapSource{
		"bad bool":     {"enabled": "notabool"},
		"bad int":      {"count": "x"},
		"bad uint":     {"workers": "-1"},
		"bad float":    {"ratio": "nan!!"},
		"bad duration": {"timeout": "3 fortnights"},
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			var c allTypes
			if err := Bind("", src, &c); err == nil {
				t.Errorf("expected an error binding %v", src)
			}
		})
	}
}

func TestScalarStringTypes(t *testing.T) {
	yaml := "" +
		"s: hello\n" +
		"b: true\n" +
		"i: 42\n" +
		"f: 3.14\n" +
		"empty: null\n"
	flat, err := FlattenYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("FlattenYAML: %v", err)
	}
	want := map[string]string{"s": "hello", "b": "true", "i": "42", "empty": ""}
	for k, v := range want {
		if flat[k] != v {
			t.Errorf("flat[%q] = %q, want %q", k, flat[k], v)
		}
	}
	if !strings.HasPrefix(flat["f"], "3.14") {
		t.Errorf("flat[f] = %q, want 3.14…", flat["f"])
	}
}
