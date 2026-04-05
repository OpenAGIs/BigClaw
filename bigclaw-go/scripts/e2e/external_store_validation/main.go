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
	reportPath := flag.String("report-path", "bigclaw-go/docs/reports/external-store-validation-report.json", "report output path")
	timeoutSeconds := flag.Int("timeout-seconds", 120, "task timeout in seconds")
	pollInterval := flag.Float64("poll-interval", 0.5, "poll interval in seconds")
	retention := flag.String("retention", "2s", "event retention duration")
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
	report, err := reporting.RunExternalStoreValidation(reporting.ExternalStoreValidationOptions{
		GoRoot:         root,
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		PollInterval:   time.Duration(*pollInterval * float64(time.Second)),
		Retention:      *retention,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(contents))
}
