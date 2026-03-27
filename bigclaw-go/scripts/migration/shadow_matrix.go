package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/internal/shadowmatrix"
)

type taskFiles []string

func (t *taskFiles) String() string {
	return fmt.Sprintf("%v", []string(*t))
}

func (t *taskFiles) Set(value string) error {
	*t = append(*t, value)
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("shadow_matrix", flag.ContinueOnError)
	primary := flags.String("primary", "", "primary base URL")
	shadow := flags.String("shadow", "", "shadow base URL")
	var files taskFiles
	flags.Var(&files, "task-file", "task file")
	corpusManifest := flags.String("corpus-manifest", "", "corpus manifest")
	replayCorpusSlices := flags.Bool("replay-corpus-slices", false, "replay corpus slices")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	healthTimeoutSeconds := flags.Int("health-timeout-seconds", 60, "health timeout seconds")
	reportPath := flags.String("report-path", "", "report path")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(files) == 0 && *corpusManifest == "" {
		_, _ = fmt.Fprintln(os.Stderr, "at least one --task-file or --corpus-manifest must be provided")
		return 2
	}
	report, err := shadowmatrix.BuildReport(shadowmatrix.BuildOptions{
		PrimaryBaseURL:     *primary,
		ShadowBaseURL:      *shadow,
		TaskFiles:          files,
		CorpusManifestPath: *corpusManifest,
		ReplayCorpusSlices: *replayCorpusSlices,
		Timeout:            time.Duration(*timeoutSeconds) * time.Second,
		HealthTimeout:      time.Duration(*healthTimeoutSeconds) * time.Second,
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
		if err := shadowmatrix.WriteReport(absPath, report); err != nil {
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
	return shadowmatrix.ExitCode(report)
}
