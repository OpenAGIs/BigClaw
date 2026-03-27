package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
)

var benchmarkLinePattern = regexp.MustCompile(`^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$`)

type automationBenchmarkMatrixOptions struct {
	GoRoot         string
	ReportPath     string
	TimeoutSeconds int
	Scenarios      []string
	RunCommand     func(name string, args []string, dir string) ([]byte, error)
	RunSoak        func(count int, workers int, timeoutSeconds int, reportPath string) (map[string]any, error)
}

func runAutomationBenchmarkMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation benchmark run-matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "soak timeout seconds")
	var scenarios multiStringFlag
	flags.Var(&scenarios, "scenario", "count:workers scenario; repeatable")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation benchmark run-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, exitCode, err := automationBenchmarkMatrix(automationBenchmarkMatrixOptions{
		GoRoot:         absPath(*goRoot),
		ReportPath:     trim(*reportPath),
		TimeoutSeconds: *timeoutSeconds,
		Scenarios:      scenarios.valuesOr([]string{"50:8", "100:12"}),
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	return err
}

func automationBenchmarkMatrix(opts automationBenchmarkMatrixOptions) (map[string]any, int, error) {
	runCommand := opts.RunCommand
	if runCommand == nil {
		runCommand = func(name string, args []string, dir string) ([]byte, error) {
			cmd := exec.Command(name, args...)
			cmd.Dir = dir
			return cmd.CombinedOutput()
		}
	}
	runSoak := opts.RunSoak
	if runSoak == nil {
		runSoak = func(count int, workers int, timeoutSeconds int, reportPath string) (map[string]any, error) {
			report, _, err := automationSoakLocal(automationSoakLocalOptions{
				Count:          count,
				Workers:        workers,
				BaseURL:        "http://127.0.0.1:8080",
				GoRoot:         opts.GoRoot,
				TimeoutSeconds: timeoutSeconds,
				Autostart:      true,
				ReportPath:     reportPath,
			})
			if err != nil {
				return nil, err
			}
			return structToMap(report), nil
		}
	}

	benchOutput, err := runCommand("go", []string{"test", "-bench", ".", "./internal/queue", "./internal/scheduler"}, opts.GoRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("run benchmarks: %w (%s)", err, string(benchOutput))
	}

	soakResults := make([]any, 0, len(opts.Scenarios))
	for _, scenario := range opts.Scenarios {
		count, workers, err := parseBenchmarkScenario(scenario)
		if err != nil {
			return nil, 0, err
		}
		scenarioReportPath := filepath.ToSlash(filepath.Join("docs", "reports", fmt.Sprintf("soak-local-%dx%d.json", count, workers)))
		result, err := runSoak(count, workers, opts.TimeoutSeconds, scenarioReportPath)
		if err != nil {
			return nil, 0, err
		}
		soakResults = append(soakResults, map[string]any{
			"scenario": map[string]any{
				"count":   count,
				"workers": workers,
			},
			"report_path": scenarioReportPath,
			"result":      result,
		})
	}

	report := map[string]any{
		"benchmark": map[string]any{
			"stdout": string(benchOutput),
			"parsed": parseBenchmarkStdout(string(benchOutput)),
		},
		"soak_matrix": soakResults,
	}
	if err := automationWriteReport(opts.GoRoot, opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func parseBenchmarkStdout(stdout string) map[string]any {
	parsed := map[string]any{}
	for _, line := range bytes.Split([]byte(stdout), []byte("\n")) {
		match := benchmarkLinePattern.FindSubmatch(bytes.TrimSpace(line))
		if len(match) != 3 {
			continue
		}
		nsPerOp, err := strconv.ParseFloat(string(match[2]), 64)
		if err != nil {
			continue
		}
		parsed[string(match[1])] = map[string]any{"ns_per_op": nsPerOp}
	}
	return parsed
}

func parseBenchmarkScenario(value string) (int, int, error) {
	parts := splitCSVWithSeparator(value, ':')
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid scenario %q, expected count:workers", value)
	}
	count, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid scenario count %q: %w", parts[0], err)
	}
	workers, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid scenario workers %q: %w", parts[1], err)
	}
	if count <= 0 || workers <= 0 {
		return 0, 0, fmt.Errorf("invalid scenario %q, count and workers must be > 0", value)
	}
	return count, workers, nil
}

type multiStringFlag struct {
	values []string
}

func (m *multiStringFlag) String() string {
	return ""
}

func (m *multiStringFlag) Set(value string) error {
	m.values = append(m.values, value)
	return nil
}

func (m *multiStringFlag) valuesOr(defaults []string) []string {
	if len(m.values) == 0 {
		return defaults
	}
	return m.values
}

func splitCSVWithSeparator(value string, separator rune) []string {
	items := []string{}
	current := ""
	for _, r := range value {
		if r == separator {
			items = append(items, trim(current))
			current = ""
			continue
		}
		current += string(r)
	}
	items = append(items, trim(current))
	return items
}
