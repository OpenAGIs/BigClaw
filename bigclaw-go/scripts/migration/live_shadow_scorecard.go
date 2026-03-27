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

	"bigclaw-go/internal/liveshadowscorecard"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("live_shadow_scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	shadowCompareReport := flags.String("shadow-compare-report", "bigclaw-go/docs/reports/shadow-compare-report.json", "shadow compare report path")
	shadowMatrixReport := flags.String("shadow-matrix-report", "bigclaw-go/docs/reports/shadow-matrix-report.json", "shadow matrix report path")
	output := flags.String("output", "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json", "output path")
	generatedAt := flags.String("generated-at", "", "override generated_at timestamp (RFC3339/RFC3339Nano)")
	pretty := flags.Bool("pretty", false, "pretty print")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	absRepoRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	parsedGeneratedAt := time.Time{}
	if strings.TrimSpace(*generatedAt) != "" {
		parsedValue, err := time.Parse(time.RFC3339Nano, strings.Replace(*generatedAt, "Z", "+00:00", 1))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 1
		}
		parsedGeneratedAt = parsedValue
	}

	report, err := liveshadowscorecard.BuildReport(liveshadowscorecard.BuildOptions{
		RepoRoot:                absRepoRoot,
		ShadowCompareReportPath: *shadowCompareReport,
		ShadowMatrixReportPath:  *shadowMatrixReport,
		GeneratedAt:             parsedGeneratedAt,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := liveshadowscorecard.WriteReport(filepath.Join(absRepoRoot, *output), report); err != nil {
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
	return 0
}
