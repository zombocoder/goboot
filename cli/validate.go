package cli

import (
	"flag"
	"fmt"
	"io"
)

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		dir      = fs.String("dir", ".", "working directory containing go.mod and goboot.yaml")
		tags     = fs.String("tags", "", "comma-separated build tags")
		strict   = fs.Bool("strict", false, "treat warnings as errors")
		profile  = fs.String("profile", "", "comma-separated active profiles (§29.3)")
		property = fs.String("property", "", "comma-separated key=value pairs for @ConditionalOnProperty")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	if code, done := maybeBootstrap(*dir, cfg, stdout, stderr); done {
		return code
	}
	patterns := resolvePatterns(fs.Args(), cfg)
	strictMode := *strict || cfg.Generation.Strict

	genPkg := generatedPackagePath(*dir, cfg.Generation.Output)
	res, _, errCount := analyzeCommon(*dir, patterns, *tags, strictMode, conditionOptions(*profile, *property), genPkg, stderr)
	if res == nil {
		return 1
	}
	if errCount > 0 {
		fmt.Fprintf(stderr, "goboot: validation failed with %d error(s)\n", errCount)
		return 1
	}
	fmt.Fprintf(stdout, "goboot: ok — %d component(s), %d route(s)\n",
		len(res.App.Components), len(res.App.Routes))
	return 0
}
