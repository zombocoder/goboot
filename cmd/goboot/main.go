// Command goboot is the annotation-driven compiler CLI (§43). It is a thin
// wrapper over the importable cli package and injects no plugins; a project that
// needs plugins builds its own main that calls cli.Main(pluginA.New(), ...), or
// relies on the self-bootstrap flow that generates one from goboot.yaml (§46.2).
package main

import (
	"os"

	"github.com/zombocoder/goboot/cli"
)

func main() {
	os.Exit(cli.Main())
}
