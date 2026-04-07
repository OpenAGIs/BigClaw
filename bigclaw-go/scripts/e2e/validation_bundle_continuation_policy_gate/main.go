package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/scripts/e2e/validationbundle"
)

func main() {
	scorecard := flag.String("scorecard", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "scorecard path")
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json", "output path")
	maxLatestAgeHours := flag.Float64("max-latest-age-hours", 72.0, "maximum age in hours for the latest bundle")
	minRecentBundles := flag.Int("min-recent-bundles", 2, "minimum number of recent bundles")
	requireRepeatedLaneCoverage := flag.Bool("require-repeated-lane-coverage", true, "require repeated lane coverage")
	allowPartialLaneHistory := flag.Bool("allow-partial-lane-history", false, "allow partial lane coverage history")
	enforcementMode := flag.String("enforcement-mode", "", "review, hold, or fail")
	enforce := flag.Bool("enforce", false, "legacy alias for fail mode")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := resolveRepoRoot()
	if err != nil {
		panic(err)
	}
	report, err := validationbundle.BuildGate(
		repoRoot,
		*scorecard,
		*maxLatestAgeHours,
		*minRecentBundles,
		!*allowPartialLaneHistory && *requireRepeatedLaneCoverage,
		*enforcementMode,
		*enforce,
		time.Now().UTC(),
	)
	if err != nil {
		panic(err)
	}
	if err := validationbundle.WriteJSON(resolvePath(repoRoot, *output), report); err != nil {
		panic(err)
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(body))
	}
	os.Exit(report.Enforcement.ExitCode)
}

func resolveRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if filepath.Base(wd) == "bigclaw-go" {
		return filepath.Dir(wd), nil
	}
	return wd, nil
}

func resolvePath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}
