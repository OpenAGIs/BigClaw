package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	defaultSubscriberTakeoverOutputPath   = "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json"
	defaultSubscriberTakeoverTemplatePath = "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json"
	legacySubscriberTakeoverScriptPath    = "scripts/e2e/subscriber_takeover_fault_matrix.py"
	goSubscriberTakeoverScriptPath        = "scripts/e2e/subscriber_takeover_fault_matrix.go"
	legacyMultiNodeSharedQueuePath        = "scripts/e2e/multi_node_shared_queue.py"
	goMultiNodeSharedQueuePath            = "scripts/e2e/multi_node_shared_queue.go"
)

func main() {
	outputPath := flag.String("output", defaultSubscriberTakeoverOutputPath, "output path")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := repoRootFromSubscriberTakeoverScript(subscriberTakeoverScriptFilePath())
	if err != nil {
		panic(err)
	}
	report, err := buildSubscriberTakeoverReport(repoRoot, defaultSubscriberTakeoverTemplatePath, time.Now().UTC())
	if err != nil {
		panic(err)
	}
	targetPath := resolveSubscriberTakeoverRepoPath(repoRoot, *outputPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		panic(err)
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(targetPath, append(body, '\n'), 0o644); err != nil {
		panic(err)
	}
	if *pretty {
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
}

func buildSubscriberTakeoverReport(repoRoot, templatePath string, generatedAt time.Time) (map[string]any, error) {
	if repoRoot == "" {
		return nil, errors.New("empty repo root")
	}
	sourcePath := resolveSubscriberTakeoverRepoPath(repoRoot, templatePath)
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	report := map[string]any{}
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	report = normalizeSubscriberTakeoverValue(report).(map[string]any)
	report["generated_at"] = utcSubscriberTakeoverISO(generatedAt)
	return report, nil
}

func normalizeSubscriberTakeoverValue(value any) any {
	switch cast := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(cast))
		for key, item := range cast {
			out[key] = normalizeSubscriberTakeoverValue(item)
		}
		return out
	case []any:
		out := make([]any, len(cast))
		for index, item := range cast {
			out[index] = normalizeSubscriberTakeoverValue(item)
		}
		return out
	case string:
		if cast == legacySubscriberTakeoverScriptPath {
			return goSubscriberTakeoverScriptPath
		}
		if cast == legacyMultiNodeSharedQueuePath {
			return goMultiNodeSharedQueuePath
		}
		return cast
	default:
		return value
	}
}

func resolveSubscriberTakeoverRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func repoRootFromSubscriberTakeoverScript(path string) (string, error) {
	if path == "" {
		return "", errors.New("empty script path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(path), "../../..")), nil
}

func subscriberTakeoverScriptFilePath() string {
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return path
}

func utcSubscriberTakeoverISO(moment time.Time) string {
	if moment.IsZero() {
		moment = time.Now().UTC()
	}
	return moment.UTC().Format(time.RFC3339Nano)
}
