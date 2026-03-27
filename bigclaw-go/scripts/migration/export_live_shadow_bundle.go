package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/liveshadowbundle"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("export_live_shadow_bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "go root")
	shadowCompareReport := flags.String("shadow-compare-report", "docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrixReport := flags.String("shadow-matrix-report", "docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	scorecardReport := flags.String("scorecard-report", "docs/reports/live-shadow-mirror-scorecard.json", "scorecard report path")
	bundleRoot := flags.String("bundle-root", "docs/reports/live-shadow-runs", "bundle root")
	summaryPath := flags.String("summary-path", "docs/reports/live-shadow-summary.json", "summary path")
	indexPath := flags.String("index-path", "docs/reports/live-shadow-index.md", "index path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-shadow-index.json", "manifest path")
	rollupPath := flags.String("rollup-path", "docs/reports/live-shadow-drift-rollup.json", "rollup path")
	runID := flags.String("run-id", "", "run id override")
	generatedAt := flags.String("generated-at", "", "override generated_at timestamp (RFC3339/RFC3339Nano)")
	rollupGeneratedAt := flags.String("rollup-generated-at", "", "override rollup generated_at timestamp (RFC3339/RFC3339Nano)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	absGoRoot, err := filepath.Abs(*goRoot)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	parsedGeneratedAt, err := parseOptionalTime(*generatedAt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	parsedRollupGeneratedAt, err := parseOptionalTime(*rollupGeneratedAt)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	result, err := liveshadowbundle.Export(liveshadowbundle.ExportOptions{
		GoRoot:              absGoRoot,
		ShadowCompareReport: *shadowCompareReport,
		ShadowMatrixReport:  *shadowMatrixReport,
		ScorecardReport:     *scorecardReport,
		BundleRoot:          *bundleRoot,
		SummaryPath:         *summaryPath,
		IndexPath:           *indexPath,
		ManifestPath:        *manifestPath,
		RollupPath:          *rollupPath,
		RunID:               *runID,
		GeneratedAt:         parsedGeneratedAt,
		RollupGeneratedAt:   parsedRollupGeneratedAt,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	body, err := json.MarshalIndent(result.Manifest, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = os.Stdout.Write(append(body, '\n'))
	return 0
}

func parseOptionalTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, strings.Replace(value, "Z", "+00:00", 1))
}
