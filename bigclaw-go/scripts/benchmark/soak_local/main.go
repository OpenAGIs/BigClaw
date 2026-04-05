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
	count := flag.Int("count", 50, "number of tasks")
	workers := flag.Int("workers", 8, "submit workers")
	baseURL := flag.String("base-url", "http://127.0.0.1:8080", "BigClaw base URL")
	goRoot := flag.String("go-root", "", "repo root")
	timeoutSeconds := flag.Int("timeout-seconds", 180, "task timeout")
	autostart := flag.Bool("autostart", false, "autostart bigclawd")
	reportPath := flag.String("report-path", "docs/reports/soak-local-report.json", "report path")
	flag.Parse()

	root, err := resolveRepoRoot(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, exitCode, err := reporting.RunLocalSoak(reporting.LocalSoakOptions{
		Count:          *count,
		Workers:        *workers,
		BaseURL:        *baseURL,
		GoRoot:         root,
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
		ReportPath:     *reportPath,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(contents))
	os.Exit(exitCode)
}

func resolveRepoRoot(value string) (string, error) {
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
