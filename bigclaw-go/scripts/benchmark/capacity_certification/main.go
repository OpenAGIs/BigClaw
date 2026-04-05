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

func main() {
	benchmarkReport := flag.String("benchmark-report", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "benchmark report path")
	mixedWorkloadReport := flag.String("mixed-workload-report", "bigclaw-go/docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	var supplementalSoakReports multiFlag
	flag.Var(&supplementalSoakReports, "supplemental-soak-report", "additional soak report paths")
	output := flag.String("output", "bigclaw-go/docs/reports/capacity-certification-matrix.json", "json output path")
	markdownOutput := flag.String("markdown-output", "bigclaw-go/docs/reports/capacity-certification-report.md", "markdown output path")
	pretty := flag.Bool("pretty", false, "print the report to stdout")
	flag.Parse()

	root, err := reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
	if err != nil {
		root, err = reporting.FindRepoRoot(".")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	report, markdown, err := reporting.BuildCapacityCertification(root, reporting.CapacityCertificationOptions{
		BenchmarkReportPath:         *benchmarkReport,
		MixedWorkloadReportPath:     *mixedWorkloadReport,
		SupplementalSoakReportPaths: supplementalSoakReports,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := reporting.WriteJSON(resolvePath(root, *output), report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(resolvePath(root, *markdownOutput), []byte(markdown), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		contents, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(contents))
	}
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func resolvePath(root string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	if filepath.Base(root) == "bigclaw-go" {
		return filepath.Join(root, strings.TrimPrefix(value, "bigclaw-go/"))
	}
	return filepath.Join(root, value)
}
