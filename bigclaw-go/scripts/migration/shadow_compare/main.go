package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"bigclaw-go/internal/reporting"
)

func main() {
	primary := flag.String("primary", "", "primary BigClaw endpoint")
	shadow := flag.String("shadow", "", "shadow BigClaw endpoint")
	taskFile := flag.String("task-file", "", "task file")
	timeoutSeconds := flag.Int("timeout-seconds", 180, "task timeout in seconds")
	healthTimeoutSeconds := flag.Int("health-timeout-seconds", 60, "health timeout in seconds")
	reportPath := flag.String("report-path", "", "optional report output path")
	flag.Parse()

	report, err := reporting.RunShadowCompare(reporting.ShadowCompareOptions{
		PrimaryBaseURL: *primary,
		ShadowBaseURL:  *shadow,
		TaskPath:       *taskFile,
		Timeout:        time.Duration(*timeoutSeconds) * time.Second,
		HealthTimeout:  time.Duration(*healthTimeoutSeconds) * time.Second,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *reportPath != "" {
		if err := reporting.WriteJSON(*reportPath, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	contents, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(contents))
	diff := map[string]any{}
	if typed, ok := report["diff"].(map[string]any); ok {
		diff = typed
	}
	stateEqual, _ := diff["state_equal"].(bool)
	eventTypesEqual, _ := diff["event_types_equal"].(bool)
	if !stateEqual || !eventTypesEqual {
		os.Exit(1)
	}
}
