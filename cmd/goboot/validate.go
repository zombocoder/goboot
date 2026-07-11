package main

import (
	"flag"
	"fmt"
	"io"
)

func cmdValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		dir    = fs.String("dir", ".", "working directory containing go.mod and goboot.yaml")
		tags   = fs.String("tags", "", "comma-separated build tags")
		strict = fs.Bool("strict", false, "treat warnings as errors")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	patterns := resolvePatterns(fs.Args(), cfg)
	strictMode := *strict || cfg.Generation.Strict

	res, errCount := analyzeCommon(*dir, patterns, *tags, strictMode, stderr)
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
