package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultBrokerFailoverReportPath           = "bigclaw-go/docs/reports/broker-failover-stub-report.json"
	defaultBrokerFailoverArtifactRoot         = "bigclaw-go/docs/reports/broker-failover-stub-artifacts"
	defaultBrokerCheckpointFencingSummaryPath = "bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json"
	defaultBrokerRetentionBoundarySummaryPath = "bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json"
	legacyBrokerFailoverStubScriptPath        = "scripts/e2e/broker_failover_stub_matrix.py"
	goBrokerFailoverStubScriptPath            = "scripts/e2e/broker_failover_stub_matrix.go"
)

func main() {
	outputPath := flag.String("output", defaultBrokerFailoverReportPath, "output path")
	artifactRoot := flag.String("artifact-root", defaultBrokerFailoverArtifactRoot, "artifact root")
	checkpointFencingSummaryOutput := flag.String("checkpoint-fencing-summary-output", defaultBrokerCheckpointFencingSummaryPath, "checkpoint fencing summary output")
	retentionBoundarySummaryOutput := flag.String("retention-boundary-summary-output", defaultBrokerRetentionBoundarySummaryPath, "retention boundary summary output")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := repoRootFromBrokerFailoverScript(brokerFailoverScriptFilePath())
	if err != nil {
		panic(err)
	}
	report, err := buildBrokerFailoverReport(repoRoot)
	if err != nil {
		panic(err)
	}
	checkpointSummary, err := buildBrokerCheckpointFencingSummary(repoRoot)
	if err != nil {
		panic(err)
	}
	retentionSummary, err := buildBrokerRetentionBoundarySummary(repoRoot)
	if err != nil {
		panic(err)
	}
	if err := writeBrokerFailoverReport(
		repoRoot,
		report,
		checkpointSummary,
		retentionSummary,
		*outputPath,
		*artifactRoot,
		*checkpointFencingSummaryOutput,
		*retentionBoundarySummaryOutput,
	); err != nil {
		panic(err)
	}
	if *pretty {
		body, err := json.MarshalIndent(stripBrokerRawArtifacts(report), "", "  ")
		if err != nil {
			panic(err)
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
}

func buildBrokerFailoverReport(repoRoot string) (map[string]any, error) {
	report := map[string]any{}
	if err := readBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, defaultBrokerFailoverReportPath), &report); err != nil {
		return nil, err
	}
	report = normalizeBrokerFailoverValue(report).(map[string]any)

	rawArtifacts := map[string]any{}
	scenarios := asBrokerFailoverSlice(report["scenarios"])
	for _, item := range scenarios {
		scenario := asBrokerFailoverMap(item)
		scenarioID, _ := scenario["scenario_id"].(string)
		artifacts := asBrokerFailoverMap(scenario["artifacts"])
		rawArtifacts[scenarioID] = map[string]any{
			"publish_attempt_ledger":    mustReadBrokerFailoverJSON(repoRoot, artifacts["publish_attempt_ledger"]),
			"replay_capture":            mustReadBrokerFailoverJSON(repoRoot, artifacts["replay_capture"]),
			"checkpoint_transition_log": mustReadBrokerFailoverJSON(repoRoot, artifacts["checkpoint_transition_log"]),
			"fault_timeline":            mustReadBrokerFailoverJSON(repoRoot, artifacts["fault_timeline"]),
			"backend_health_snapshot":   mustReadBrokerFailoverJSON(repoRoot, artifacts["backend_health_snapshot"]),
		}
	}
	report["raw_artifacts"] = rawArtifacts
	return report, nil
}

func buildBrokerCheckpointFencingSummary(repoRoot string) (map[string]any, error) {
	summary := map[string]any{}
	if err := readBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, defaultBrokerCheckpointFencingSummaryPath), &summary); err != nil {
		return nil, err
	}
	return normalizeBrokerFailoverValue(summary).(map[string]any), nil
}

func buildBrokerRetentionBoundarySummary(repoRoot string) (map[string]any, error) {
	summary := map[string]any{}
	if err := readBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, defaultBrokerRetentionBoundarySummaryPath), &summary); err != nil {
		return nil, err
	}
	return normalizeBrokerFailoverValue(summary).(map[string]any), nil
}

func writeBrokerFailoverReport(
	repoRoot string,
	report map[string]any,
	checkpointSummary map[string]any,
	retentionSummary map[string]any,
	outputPath string,
	artifactRoot string,
	checkpointSummaryPath string,
	retentionSummaryPath string,
) error {
	if err := writeBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, outputPath), stripBrokerRawArtifacts(report)); err != nil {
		return err
	}
	if err := writeBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, checkpointSummaryPath), checkpointSummary); err != nil {
		return err
	}
	if err := writeBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, retentionSummaryPath), retentionSummary); err != nil {
		return err
	}
	rawArtifacts := asBrokerFailoverMap(report["raw_artifacts"])
	for scenarioID, value := range rawArtifacts {
		scenarioArtifacts := asBrokerFailoverMap(value)
		targetDir := resolveBrokerFailoverRepoPath(repoRoot, filepath.Join(artifactRoot, scenarioID))
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		for filename, key := range map[string]string{
			"publish-attempt-ledger.json":    "publish_attempt_ledger",
			"replay-capture.json":            "replay_capture",
			"checkpoint-transition-log.json": "checkpoint_transition_log",
			"fault-timeline.json":            "fault_timeline",
			"backend-health.json":            "backend_health_snapshot",
		} {
			if err := writeBrokerFailoverJSON(filepath.Join(targetDir, filename), scenarioArtifacts[key]); err != nil {
				return err
			}
		}
	}
	return nil
}

func stripBrokerRawArtifacts(report map[string]any) map[string]any {
	stripped := make(map[string]any, len(report))
	for key, value := range report {
		if key == "raw_artifacts" {
			continue
		}
		stripped[key] = value
	}
	return stripped
}

func mustReadBrokerFailoverJSON(repoRoot string, rawPath any) any {
	path, _ := rawPath.(string)
	value := any(nil)
	if err := readBrokerFailoverJSON(resolveBrokerFailoverRepoPath(repoRoot, path), &value); err != nil {
		panic(err)
	}
	return normalizeBrokerFailoverValue(value)
}

func readBrokerFailoverJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func writeBrokerFailoverJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func normalizeBrokerFailoverValue(value any) any {
	switch cast := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(cast))
		for key, item := range cast {
			out[key] = normalizeBrokerFailoverValue(item)
		}
		return out
	case []any:
		out := make([]any, len(cast))
		for index, item := range cast {
			out[index] = normalizeBrokerFailoverValue(item)
		}
		return out
	case string:
		if cast == legacyBrokerFailoverStubScriptPath {
			return goBrokerFailoverStubScriptPath
		}
		return cast
	default:
		return value
	}
}

func resolveBrokerFailoverRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func repoRootFromBrokerFailoverScript(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty script path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), "../../..")), nil
}

func brokerFailoverScriptFilePath() string {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return path
}

func asBrokerFailoverMap(value any) map[string]any {
	if cast, ok := value.(map[string]any); ok {
		return cast
	}
	return map[string]any{}
}

func asBrokerFailoverSlice(value any) []any {
	if cast, ok := value.([]any); ok {
		return cast
	}
	return nil
}
