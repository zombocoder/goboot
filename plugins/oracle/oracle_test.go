package oracle

import "testing"

func TestPluginIdentity(t *testing.T) {
	p := New()
	if p.Name() != "oracle" || p.Version() != "0.1.0" {
		t.Errorf("identity = %s/%s", p.Name(), p.Version())
	}
}

func TestDialectRegistered(t *testing.T) {
	d, ok := New().Dialects()["oracle"]
	if !ok {
		t.Fatal("oracle dialect not registered")
	}
	if d.Name() != "oracle" {
		t.Errorf("dialect name = %q", d.Name())
	}
	if got := d.Placeholder(1) + " " + d.Placeholder(2); got != ":1 :2" {
		t.Errorf("placeholders = %q, want :1 :2", got)
	}
}
