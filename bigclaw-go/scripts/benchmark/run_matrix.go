package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/benchmarkmatrix"
)

type scenarioFlags []string

func (s *scenarioFlags) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *scenarioFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("run_matrix", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "go root")
	reportPath := flags.String("report-path", "docs/reports/benchmark-matrix-report.json", "report path")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "timeout seconds")
	var scenarios scenarioFlags
	flags.Var(&scenarios, "scenario", "count:workers benchmark scenario")
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
	report, err := benchmarkmatrix.BuildReport(absRoot, scenarios, *timeoutSeconds, nil)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := benchmarkmatrix.WriteReport(filepath.Join(absRoot, *reportPath), report); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	_, _ = os.Stdout.Write(append(body, '\n'))
	return 0
}
