package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zombocoder/goboot/generator/di"
)

func cmdClean(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("clean", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dir := flags.String("dir", ".", "working directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig(*dir)
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	removed, err := cleanTree(filepath.Join(*dir, cfg.Generation.Output))
	if err != nil {
		fmt.Fprintf(stderr, "goboot: %v\n", err)
		return 1
	}
	for _, f := range removed {
		fmt.Fprintf(stdout, "goboot: removed %s\n", f)
	}
	fmt.Fprintf(stdout, "goboot: removed %d file(s)\n", len(removed))
	return 0
}

// cleanDir removes goboot-generated files from a single directory (non
// recursive). A missing directory is not an error.
func cleanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if isGeneratedFile(path) {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
	}
	return nil
}

// cleanTree walks a directory tree removing every goboot-generated file and
// returns the paths removed. It removes only files that carry the generated
// marker, so hand-written files are never touched (§43.6).
func cleanTree(root string) ([]string, error) {
	var removed []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() || !isGeneratedFile(path) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		removed = append(removed, path)
		return nil
	})
	if os.IsNotExist(err) {
		return removed, nil
	}
	return removed, err
}

// isGeneratedFile reports whether a file is a goboot-generated Go file: it must
// have the generated-file prefix and contain the generated marker.
func isGeneratedFile(path string) bool {
	if !strings.HasPrefix(filepath.Base(path), generatedFilePrefix) {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(di.GeneratedMarker))
}
