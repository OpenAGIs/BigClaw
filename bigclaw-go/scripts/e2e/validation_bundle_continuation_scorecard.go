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
	repoRoot := flag.String("repo-root", "", "repository root")
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	pretty := flag.Bool("pretty", false, "print rendered json")
	flag.Parse()

	report, err := reporting.BuildValidationBundleContinuationScorecard(reporting.ValidationBundleContinuationScorecardOptions{
		RepoRoot: *repoRoot,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteValidationBundleContinuationReport(resolveOutputPath(*repoRoot, *output), report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
}

func resolveOutputPath(repoRoot string, output string) string {
	if filepath.IsAbs(output) {
		return output
	}
	resolvedRoot, err := inferRepoRoot(repoRoot)
	if err != nil {
		return output
	}
	return filepath.Join(resolvedRoot, output)
}

func inferRepoRoot(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(explicit)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(cwd, "bigclaw-go", "go.mod")); err == nil {
		return cwd, nil
	}
	if filepath.Base(cwd) == "bigclaw-go" {
		if _, err := os.Stat(filepath.Join(cwd, "go.mod")); err == nil {
			return filepath.Dir(cwd), nil
		}
	}
	return cwd, nil
}
