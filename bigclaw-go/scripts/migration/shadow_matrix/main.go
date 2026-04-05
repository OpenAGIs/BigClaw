package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"bigclaw-go/internal/reporting"
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	primary := flag.String("primary", "", "primary BigClaw endpoint")
	shadow := flag.String("shadow", "", "shadow BigClaw endpoint")
	var taskFiles multiFlag
	flag.Var(&taskFiles, "task-file", "task file")
	corpusManifest := flag.String("corpus-manifest", "", "corpus manifest path")
	replayCorpusSlices := flag.Bool("replay-corpus-slices", false, "submit replayable corpus slices")
	timeoutSeconds := flag.Int("timeout-seconds", 180, "task timeout in seconds")
	healthTimeoutSeconds := flag.Int("health-timeout-seconds", 60, "health timeout in seconds")
	reportPath := flag.String("report-path", "", "optional report output path")
	flag.Parse()

	report, err := reporting.RunShadowMatrix(reporting.ShadowMatrixOptions{
		PrimaryBaseURL:     *primary,
		ShadowBaseURL:      *shadow,
		TaskFiles:          taskFiles,
		CorpusManifestPath: *corpusManifest,
		ReplayCorpusSlices: *replayCorpusSlices,
		Timeout:            time.Duration(*timeoutSeconds) * time.Second,
		HealthTimeout:      time.Duration(*healthTimeoutSeconds) * time.Second,
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
	results, _ := report["results"].([]map[string]any)
	if !shadowMatrixMatched(results) {
		os.Exit(1)
	}
}

func shadowMatrixMatched(results []map[string]any) bool {
	for _, item := range results {
		diff, _ := item["diff"].(map[string]any)
		stateEqual, _ := diff["state_equal"].(bool)
		eventTypesEqual, _ := diff["event_types_equal"].(bool)
		if !stateEqual || !eventTypesEqual {
			return false
		}
	}
	return true
}
