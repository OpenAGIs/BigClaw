package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/reporting"
)

func main() {
	output := flag.String("output", "bigclaw-go/docs/reports/broker-failover-stub-report.json", "report output path")
	artifactRoot := flag.String("artifact-root", "bigclaw-go/docs/reports/broker-failover-stub-artifacts", "artifact root")
	checkpointSummary := flag.String("checkpoint-fencing-summary-output", "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json", "checkpoint fencing summary output")
	retentionSummary := flag.String("retention-boundary-summary-output", "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json", "retention boundary summary output")
	pretty := flag.Bool("pretty", false, "compatibility flag; outputs stay indented")
	flag.Parse()
	_ = pretty

	root, err := resolveBrokerRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteBrokerFailoverStubArtifacts(root, reporting.BrokerStubOptions{
		Output:                         *output,
		ArtifactRoot:                   *artifactRoot,
		CheckpointFencingSummaryOutput: *checkpointSummary,
		RetentionBoundarySummaryOutput: *retentionSummary,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveBrokerRepoRoot() (string, error) {
	root, err := reporting.FindRepoRoot(".")
	if err == nil {
		return root, nil
	}
	return reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
}
