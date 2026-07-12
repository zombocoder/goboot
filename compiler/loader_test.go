package compiler

import "testing"

func TestIsNotYetGeneratedError(t *testing.T) {
	const out = "github.com/acme/app/internal/generated"
	tests := []struct {
		name    string
		pkgPath string
		msg     string
		ignore  string
		want    bool
	}{
		{"no ignore configured", "github.com/acme/app", `could not import ` + out, "", false},
		{"the output package's own empty-dir error", out, `invalid package name: ""`, out, true},
		{"consumer cannot import the output package", "github.com/acme/app",
			`could not import ` + out + ` (invalid package name: "")`, out, true},
		{"consumer sees no Go files in output", "github.com/acme/app",
			`no Go files in ` + out, out, true},
		{"unrelated real error is kept", "github.com/acme/app",
			`undefined: somethingElse`, out, false},
		{"real error that merely names the path is kept", "github.com/acme/app",
			`type ` + out + `.Foo has no field Bar`, out, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotYetGeneratedError(tc.pkgPath, tc.msg, tc.ignore); got != tc.want {
				t.Errorf("isNotYetGeneratedError(%q, %q, %q) = %v, want %v",
					tc.pkgPath, tc.msg, tc.ignore, got, tc.want)
			}
		})
	}
}
