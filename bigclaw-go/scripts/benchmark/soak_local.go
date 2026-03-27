package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/soaklocal"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("soak_local", flag.ContinueOnError)
	count := flags.Int("count", 50, "task count")
	workers := flags.Int("workers", 8, "worker count")
	baseURL := flags.String("base-url", "http://127.0.0.1:8080", "base URL")
	goRoot := flags.String("go-root", ".", "go root")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	autostart := flags.Bool("autostart", false, "autostart the service")
	reportPath := flags.String("report-path", "docs/reports/soak-local-report.json", "report path")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	absRoot, err := filepath.Abs(*goRoot)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	report, logPath, err := soaklocal.Run(soaklocal.Options{
		Count:          *count,
		Workers:        *workers,
		BaseURL:        *baseURL,
		GoRoot:         absRoot,
		TimeoutSeconds: *timeoutSeconds,
		Autostart:      *autostart,
		ReportPath:     *reportPath,
	})
	body, marshalErr := json.MarshalIndent(report, "", "  ")
	if marshalErr == nil {
		_, _ = os.Stdout.Write(append(body, '\n'))
	}
	if logPath != "" {
		_, _ = fmt.Fprintf(os.Stdout, "service-log: %s\n", logPath)
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
