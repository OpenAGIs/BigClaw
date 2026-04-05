package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/reporting"
)

func main() {
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	pretty := flag.Bool("pretty", false, "print the generated report to stdout")
	flag.Parse()

	repoRoot, err := reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
	if err != nil {
		repoRoot, err = reporting.FindRepoRoot(".")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := reporting.BuildValidationBundleContinuationScorecard(repoRoot, reporting.ContinuationScorecardOptions{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outputPath := resolveOutputPath(repoRoot, *output)
	if err := reporting.WriteJSON(outputPath, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		contents, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(contents))
	}
}

func resolveOutputPath(repoRoot string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	trimmed := value
	if filepath.Base(repoRoot) == "bigclaw-go" {
		trimmed = strings.TrimPrefix(trimmed, "bigclaw-go/")
	}
	return filepath.Join(repoRoot, trimmed)
}
