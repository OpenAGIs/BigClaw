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
	goRoot := flag.String("go-root", "", "repo root")
	reportPath := flag.String("report-path", "docs/reports/mixed-workload-matrix-report.json", "report path")
	timeoutSeconds := flag.Int("timeout-seconds", 240, "task timeout")
	autostart := flag.Bool("autostart", true, "autostart bigclawd")
	baseURL := flag.String("base-url", "", "existing BigClaw base URL")
	flag.Parse()

	root, err := resolveMixedRepoRoot(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, exitCode, err := reporting.RunMixedWorkloadMatrix(reporting.MixedWorkloadMatrixOptions{
		GoRoot:         root,
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
		BaseURL:        *baseURL,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(contents))
	os.Exit(exitCode)
}

func resolveMixedRepoRoot(value string) (string, error) {
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
