package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/internal/shadowcompare"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("shadow_compare", flag.ContinueOnError)
	primary := flags.String("primary", "", "primary base URL")
	shadow := flags.String("shadow", "", "shadow base URL")
	taskFile := flags.String("task-file", "", "task file")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	healthTimeoutSeconds := flags.Int("health-timeout-seconds", 60, "health timeout seconds")
	reportPath := flags.String("report-path", "", "report path")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *primary == "" || *shadow == "" || *taskFile == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--primary, --shadow, and --task-file are required")
		return 2
	}

	task, err := shadowcompare.LoadTask(*taskFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	report, err := shadowcompare.CompareTask(shadowcompare.CompareOptions{
		PrimaryBaseURL: *primary,
		ShadowBaseURL:  *shadow,
		Task:           task,
		Timeout:        time.Duration(*timeoutSeconds) * time.Second,
		HealthTimeout:  time.Duration(*healthTimeoutSeconds) * time.Second,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *reportPath != "" {
		absPath, err := filepath.Abs(*reportPath)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := shadowcompare.WriteReport(absPath, report); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = os.Stdout.Write(append(body, '\n'))
	return shadowcompare.ExitCode(report)
}
