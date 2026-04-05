package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/reporting"
)

func main() {
	output := flag.String("output", "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json", "report output path")
	pretty := flag.Bool("pretty", false, "compatibility flag; outputs stay indented")
	flag.Parse()
	_ = pretty

	root, err := resolveTakeoverRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteSubscriberTakeoverArtifacts(root, reporting.SubscriberTakeoverOptions{
		Output: *output,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveTakeoverRepoRoot() (string, error) {
	root, err := reporting.FindRepoRoot(".")
	if err == nil {
		return root, nil
	}
	return reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
}
