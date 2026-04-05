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
	goRoot := flag.String("go-root", "", "bigclaw-go root")
	shadowCompare := flag.String("shadow-compare-report", "docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrix := flag.String("shadow-matrix-report", "docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	scorecard := flag.String("scorecard-report", "docs/reports/live-shadow-mirror-scorecard.json", "scorecard report path")
	bundleRoot := flag.String("bundle-root", "docs/reports/live-shadow-runs", "bundle root")
	summaryPath := flag.String("summary-path", "docs/reports/live-shadow-summary.json", "summary path")
	indexPath := flag.String("index-path", "docs/reports/live-shadow-index.md", "index path")
	manifestPath := flag.String("manifest-path", "docs/reports/live-shadow-index.json", "manifest path")
	rollupPath := flag.String("rollup-path", "docs/reports/live-shadow-drift-rollup.json", "rollup path")
	runID := flag.String("run-id", "", "run id")
	flag.Parse()

	root, err := resolveGoRoot(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	manifest, _, err := reporting.ExportLiveShadowBundle(root, reporting.LiveShadowBundleOptions{
		ShadowCompareReportPath: *shadowCompare,
		ShadowMatrixReportPath:  *shadowMatrix,
		ScorecardReportPath:     *scorecard,
		BundleRootPath:          *bundleRoot,
		SummaryPath:             *summaryPath,
		IndexPath:               *indexPath,
		ManifestPath:            *manifestPath,
		RollupPath:              *rollupPath,
		RunID:                   *runID,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, _ := json.MarshalIndent(manifest, "", "  ")
	fmt.Println(string(contents))
}

func resolveGoRoot(value string) (string, error) {
	if value != "" {
		if filepath.IsAbs(value) {
			return value, nil
		}
		return filepath.Abs(value)
	}
	root, err := reporting.FindRepoRoot(".")
	if err == nil {
		return root, nil
	}
	return reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
}
