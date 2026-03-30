package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultReportPath            = "bigclaw-go/docs/reports/broker-failover-stub-report.json"
	defaultCheckpointSummaryPath = "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json"
	defaultRetentionSummaryPath  = "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json"
)

func main() {
	goRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	flags := flag.NewFlagSet("broker-failover-stub-matrix", flag.ExitOnError)
	outputPath := flags.String("output", defaultReportPath, "Path for the broker stub report")
	checkpointSummaryPath := flags.String("checkpoint-summary-output", defaultCheckpointSummaryPath, "Path for the checkpoint fencing summary")
	retentionSummaryPath := flags.String("retention-summary-output", defaultRetentionSummaryPath, "Path for the retention-boundary summary")
	pretty := flags.Bool("pretty", false, "Print the report to stdout")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	reportBody, err := os.ReadFile(resolveRepoPath(repoRoot, defaultReportPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	checkpointBody, err := os.ReadFile(resolveRepoPath(repoRoot, defaultCheckpointSummaryPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	retentionBody, err := os.ReadFile(resolveRepoPath(repoRoot, defaultRetentionSummaryPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := writeJSONCopy(resolveRepoPath(repoRoot, *outputPath), reportBody); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeJSONCopy(resolveRepoPath(repoRoot, *checkpointSummaryPath), checkpointBody); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := writeJSONCopy(resolveRepoPath(repoRoot, *retentionSummaryPath), retentionBody); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		var prettyBody any
		if err := json.Unmarshal(reportBody, &prettyBody); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		rendered, err := json.MarshalIndent(prettyBody, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(rendered))
	}
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(repoRoot, path)
}

func writeJSONCopy(path string, body []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}
