package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/internal/reporting"
)

func main() {
	goRoot := flag.String("go-root", "", "go repo root")
	reportPath := flag.String("report-path", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "shared queue report path")
	takeoverReportPath := flag.String("takeover-report-path", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json", "live takeover report path")
	takeoverArtifactDir := flag.String("takeover-artifact-dir", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts", "live takeover artifact directory")
	takeoverTTLSeconds := flag.Float64("takeover-ttl-seconds", 1.0, "takeover ttl seconds")
	count := flag.Int("count", 200, "task count")
	submitWorkers := flag.Int("submit-workers", 8, "concurrent submit workers")
	timeoutSeconds := flag.Int("timeout-seconds", 180, "timeout in seconds")
	flag.Parse()

	root := *goRoot
	var err error
	if root == "" {
		root, err = reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
		if err != nil {
			root, err = reporting.FindRepoRoot(".")
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	sharedQueueReport, liveTakeoverReport, err := reporting.RunMultiNodeSharedQueue(reporting.MultiNodeSharedQueueOptions{
		GoRoot:              root,
		ReportPath:          *reportPath,
		TakeoverReportPath:  *takeoverReportPath,
		TakeoverArtifactDir: *takeoverArtifactDir,
		Count:               *count,
		SubmitWorkers:       *submitWorkers,
		TimeoutSeconds:      *timeoutSeconds,
		TakeoverTTL:         time.Duration(*takeoverTTLSeconds * float64(time.Second)),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, err := json.MarshalIndent(map[string]any{
		"shared_queue_report":  sharedQueueReport,
		"live_takeover_report": liveTakeoverReport,
	}, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(contents))
	sharedQueueOK, _ := sharedQueueReport["all_ok"].(bool)
	failingScenarios := 0
	if summary, ok := liveTakeoverReport["summary"].(map[string]any); ok {
		switch typed := summary["failing_scenarios"].(type) {
		case int:
			failingScenarios = typed
		case float64:
			failingScenarios = int(typed)
		}
	}
	if !sharedQueueOK || failingScenarios != 0 {
		os.Exit(1)
	}
}
