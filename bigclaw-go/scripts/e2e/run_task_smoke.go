package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"bigclaw-go/internal/runtasksmoke"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("run_task_smoke", flag.ContinueOnError)
	executor := flags.String("executor", "", "executor")
	title := flags.String("title", "", "title")
	entrypoint := flags.String("entrypoint", "", "entrypoint")
	image := flags.String("image", "", "image")
	baseURL := flags.String("base-url", getenv("BIGCLAW_ADDR", "http://127.0.0.1:8080"), "base URL")
	goRoot := flags.String("go-root", ".", "Go repo root")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	pollInterval := flags.Float64("poll-interval", 1.0, "poll interval seconds")
	runtimeEnvJSON := flags.String("runtime-env-json", "", "runtime env JSON")
	metadataJSON := flags.String("metadata-json", "", "metadata JSON")
	reportPath := flags.String("report-path", "", "report path")
	autostart := flags.Bool("autostart", false, "autostart bigclawd")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *executor == "" || *title == "" || *entrypoint == "" {
		_, _ = fmt.Fprintln(os.Stderr, "--executor, --title, and --entrypoint are required")
		return 2
	}

	runtimeEnv, err := decodeJSONMap(*runtimeEnvJSON)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}
	metadata, err := decodeJSONMap(*metadataJSON)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 2
	}

	report, exitCode, err := runtasksmoke.Run(runtasksmoke.Options{
		Executor:     *executor,
		Title:        *title,
		Entrypoint:   *entrypoint,
		Image:        *image,
		BaseURL:      *baseURL,
		GoRoot:       *goRoot,
		Timeout:      time.Duration(*timeoutSeconds) * time.Second,
		PollInterval: time.Duration(*pollInterval * float64(time.Second)),
		RuntimeEnv:   runtimeEnv,
		Metadata:     metadata,
		ReportPath:   *reportPath,
		Autostart:    *autostart,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := os.Stdout.Write(append(body, '\n')); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if logPath, ok := report["service_log"].(string); ok && logPath != "" {
		_, _ = fmt.Fprintf(os.Stderr, "bigclawd log: %s\n", logPath)
	}
	return exitCode
}

func decodeJSONMap(raw string) (map[string]any, error) {
	if raw == "" {
		return nil, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("invalid JSON object: %w", err)
	}
	return decoded, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
