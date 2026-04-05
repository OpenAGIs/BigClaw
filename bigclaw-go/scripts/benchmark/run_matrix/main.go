package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bigclaw-go/internal/reporting"
)

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func main() {
	goRoot := flag.String("go-root", "", "repo root")
	reportPath := flag.String("report-path", "docs/reports/benchmark-matrix-report.json", "matrix report path")
	timeoutSeconds := flag.Int("timeout-seconds", 180, "soak timeout")
	var scenarios multiFlag
	flag.Var(&scenarios, "scenario", "count:workers scenario")
	flag.Parse()

	root, err := resolveBenchmarkRepoRoot(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	report, err := reporting.RunBenchmarkMatrix(reporting.BenchmarkMatrixOptions{
		GoRoot:         root,
		ReportPath:     *reportPath,
		TimeoutSeconds: *timeoutSeconds,
		Scenarios:      scenarios,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(contents))
}

func resolveBenchmarkRepoRoot(value string) (string, error) {
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
