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
	scorecard := flag.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flag.Float64("max-latest-age-hours", 72, "maximum age of latest bundle in hours")
	minRecentBundles := flag.Int("min-recent-bundles", 2, "minimum recent bundle count")
	requireRepeatedLaneCoverage := flag.Bool("require-repeated-lane-coverage", true, "require repeated recent lane coverage")
	allowPartialLaneHistory := flag.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flag.String("enforcement-mode", "", "review, hold, or fail")
	legacyEnforce := flag.Bool("enforce", false, "compatibility alias for fail mode")
	pretty := flag.Bool("pretty", false, "print rendered json")
	flag.Parse()

	requireRepeated := *requireRepeatedLaneCoverage && !*allowPartialLaneHistory
	report, err := reporting.BuildValidationBundleContinuationPolicyGate(reporting.ValidationBundleContinuationPolicyGateOptions{
		RepoRoot:                    *repoRoot,
		ScorecardPath:               *scorecard,
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: requireRepeated,
		EnforcementMode:             *enforcementMode,
		LegacyEnforceContinuation:   *legacyEnforce,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteValidationBundleContinuationReport(resolveCLIPath(*repoRoot, *output), report); err != nil {
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
	os.Exit(report.Enforcement.ExitCode)
}

func resolveCLIPath(repoRoot string, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	resolvedRoot, err := inferPolicyRepoRoot(repoRoot)
	if err != nil {
		return path
	}
	return filepath.Join(resolvedRoot, path)
}

func inferPolicyRepoRoot(explicit string) (string, error) {
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
