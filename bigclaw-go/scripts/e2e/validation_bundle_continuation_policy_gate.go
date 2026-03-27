package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/continuationgate"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("validation_bundle_continuation_policy_gate", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", ".", "repository root")
	scorecard := flags.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flags.Float64("max-latest-age-hours", 72, "max latest bundle age in hours")
	minRecentBundles := flags.Int("min-recent-bundles", 2, "minimum recent bundles")
	requireRepeatedLaneCoverage := flags.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartialLaneHistory := flags.Bool("allow-partial-lane-history", false, "allow partial lane history")
	enforcementMode := flags.String("enforcement-mode", "", "review, hold, or fail")
	enforce := flags.Bool("enforce", false, "legacy fail-mode shortcut")
	pretty := flags.Bool("pretty", false, "pretty-print json to stdout")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return 2
	}
	absRepoRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	report, err := continuationgate.BuildReport(continuationgate.BuildOptions{
		RepoRoot:                    absRepoRoot,
		ScorecardPath:               *scorecard,
		MaxLatestAgeHours:           *maxLatestAgeHours,
		MinRecentBundles:            *minRecentBundles,
		RequireRepeatedLaneCoverage: *requireRepeatedLaneCoverage && !*allowPartialLaneHistory,
		EnforcementMode:             *enforcementMode,
		LegacyEnforceContinuation:   *enforce,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := continuationgate.WriteReport(filepath.Join(absRepoRoot, *output), report, true); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 1
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
	enforcement := report["enforcement"].(continuationgate.EnforcementSummary)
	return enforcement.ExitCode
}
