package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	brokerStubReportPath            = "docs/reports/broker-failover-stub-report.json"
	brokerStubArtifactRoot          = "docs/reports/broker-failover-stub-artifacts"
	brokerStubCheckpointSummaryPath = "docs/reports/broker-checkpoint-fencing-proof-summary.json"
	brokerStubRetentionSummaryPath  = "docs/reports/broker-retention-boundary-proof-summary.json"
)

func runAutomationBrokerFailoverStubMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e broker-failover-stub-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	output := flags.String("output", brokerStubReportPath, "output path")
	artifactRoot := flags.String("artifact-root", brokerStubArtifactRoot, "artifact root")
	checkpointSummary := flags.String("checkpoint-fencing-summary-output", brokerStubCheckpointSummaryPath, "checkpoint fencing summary output")
	retentionSummary := flags.String("retention-boundary-summary-output", brokerStubRetentionSummaryPath, "retention boundary summary output")
	asJSON := flags.Bool("json", true, "json")
	pretty := flags.Bool("pretty", false, "pretty-print copied report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e broker-failover-stub-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, err := automationBrokerFailoverStubMatrix(absPath(*goRoot), *output, *artifactRoot, *checkpointSummary, *retentionSummary)
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return emit(report, *asJSON, 0)
}

func automationBrokerFailoverStubMatrix(goRoot, outputPath, artifactRoot, checkpointSummaryPath, retentionSummaryPath string) (map[string]any, error) {
	root := absPath(goRoot)
	reportSource := filepath.Join(root, brokerStubReportPath)
	checkpointSource := filepath.Join(root, brokerStubCheckpointSummaryPath)
	retentionSource := filepath.Join(root, brokerStubRetentionSummaryPath)
	artifactSourceRoot := filepath.Join(root, brokerStubArtifactRoot)

	report, err := e2eReadJSONMap(reportSource)
	if err != nil {
		return nil, fmt.Errorf("read canonical stub report: %w", err)
	}
	checkpointSummary, err := e2eReadJSONMap(checkpointSource)
	if err != nil {
		return nil, fmt.Errorf("read checkpoint fencing summary: %w", err)
	}
	retentionSummary, err := e2eReadJSONMap(retentionSource)
	if err != nil {
		return nil, fmt.Errorf("read retention boundary summary: %w", err)
	}

	if err := e2eWriteJSON(filepath.Join(root, outputPath), report); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(filepath.Join(root, checkpointSummaryPath), checkpointSummary); err != nil {
		return nil, err
	}
	if err := e2eWriteJSON(filepath.Join(root, retentionSummaryPath), retentionSummary); err != nil {
		return nil, err
	}
	if err := copyDirContents(artifactSourceRoot, filepath.Join(root, artifactRoot)); err != nil {
		return nil, err
	}
	return report, nil
}

func copyDirContents(sourceRoot, destRoot string) error {
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return err
	}
	return filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destRoot, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(source, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
