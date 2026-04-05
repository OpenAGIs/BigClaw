package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/reporting"
)

func main() {
	output := flag.String("output", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "json output path")
	pretty := flag.Bool("pretty", false, "print the report to stdout")
	flag.Parse()

	root, err := resolveCoordinationRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := reporting.BuildCrossProcessCoordinationSurface(root, reporting.CrossProcessCoordinationSurfaceOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteJSON(resolveCoordinationPath(root, *output), report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		contents, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(contents))
	}
}

func resolveCoordinationRepoRoot() (string, error) {
	root, err := reporting.FindRepoRoot(".")
	if err == nil {
		return root, nil
	}
	return reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
}

func resolveCoordinationPath(root string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, value)
}
