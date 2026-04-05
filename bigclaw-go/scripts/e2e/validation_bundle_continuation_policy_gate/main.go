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
	scorecard := flag.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flag.Float64("max-latest-age-hours", 72.0, "max latest bundle age in hours")
	minRecentBundles := flag.Int("min-recent-bundles", 2, "minimum recent bundle count")
	requireRepeatedLaneCoverage := flag.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartialLaneHistory := flag.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flag.String("enforcement-mode", "", "review, hold, or fail")
	enforce := flag.Bool("enforce", false, "legacy alias for fail mode")
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
	report, err := reporting.BuildValidationBundleContinuationPolicyGate(repoRoot, reporting.ContinuationPolicyGateOptions{
		ScorecardPath:               normalizeInputPath(repoRoot, *scorecard),
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: !*allowPartialLaneHistory && *requireRepeatedLaneCoverage,
		EnforcementMode:             *enforcementMode,
		LegacyEnforce:               *enforce,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	outputPath := normalizeInputPath(repoRoot, *output)
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(repoRoot, outputPath)
	}
	if err := reporting.WriteJSON(outputPath, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		contents, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(contents))
	}
	os.Exit(report.Enforcement.ExitCode)
}

func normalizeInputPath(repoRoot string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	if filepath.Base(repoRoot) == "bigclaw-go" {
		return strings.TrimPrefix(value, "bigclaw-go/")
	}
	return value
}
