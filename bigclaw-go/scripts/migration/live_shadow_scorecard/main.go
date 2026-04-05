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
	shadowCompare := flag.String("shadow-compare-report", "bigclaw-go/docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrix := flag.String("shadow-matrix-report", "bigclaw-go/docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	output := flag.String("output", "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json", "output path")
	pretty := flag.Bool("pretty", false, "print the report to stdout")
	flag.Parse()

	root, err := reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
	if err != nil {
		root, err = reporting.FindRepoRoot(".")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	report, err := reporting.BuildLiveShadowScorecard(root, reporting.LiveShadowScorecardOptions{
		ShadowCompareReportPath: *shadowCompare,
		ShadowMatrixReportPath:  *shadowMatrix,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteJSON(resolveOutputPath(root, *output), report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		contents, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(contents))
	}
}

func resolveOutputPath(root string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	if filepath.Base(root) == "bigclaw-go" {
		return filepath.Join(root, strings.TrimPrefix(value, "bigclaw-go/"))
	}
	return filepath.Join(root, value)
}
