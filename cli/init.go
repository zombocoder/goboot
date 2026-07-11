package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// configTemplate is the scaffold written by `goboot init` (§43.1, §45).
const configTemplate = `version: v1

application:
  name: my-service
  packages:
    - ./...

generation:
  output: internal/generated
  package: generated
  clean: true
  strict: false

# Compile-time plugins to link into the CLI (native drivers, generators,
# annotations). goboot builds a plugin-aware binary from this list. Use the
# shorthand "module@version", or the explicit form for a custom import/constructor.
# plugins:
#   - github.com/acme/goboot-plugin-pgx@v0.2.0
#   - module: github.com/acme/goboot-redis
#     version: v1.3.1
#     import: github.com/acme/goboot-redis/gobootx
#     new: New
`

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		dir   = fs.String("dir", ".", "working directory")
		force = fs.Bool("force", false, "overwrite an existing goboot.yaml")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	path := filepath.Join(*dir, configFileName)
	if _, err := os.Stat(path); err == nil && !*force {
		fmt.Fprintf(stderr, "goboot: %s already exists (use -force to overwrite)\n", configFileName)
		return 1
	}
	if err := os.WriteFile(path, []byte(configTemplate), 0o644); err != nil {
		fmt.Fprintf(stderr, "goboot: writing %s: %v\n", configFileName, err)
		return 1
	}
	fmt.Fprintf(stdout, "goboot: wrote %s\n", path)
	fmt.Fprintln(stdout, "Add this directive to a Go file, then run `go generate ./...`:")
	fmt.Fprintln(stdout, "\t//go:generate go run github.com/zombocoder/goboot/cmd/goboot generate ./...")
	return 0
}
