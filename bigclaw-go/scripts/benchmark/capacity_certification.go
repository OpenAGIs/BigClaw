package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/capacitycert"
)

type soakReportFlags []string

func (s *soakReportFlags) String() string {
	return fmt.Sprintf("%v", []string(*s))
}

func (s *soakReportFlags) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("capacity_certification", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	benchmarkReport := flags.String("benchmark-report", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "benchmark report path")
	mixedWorkloadReport := flags.String("mixed-workload-report", "bigclaw-go/docs/reports/mixed-workload-matrix-report.json", "mixed workload report path")
	output := flags.String("output", "bigclaw-go/docs/reports/capacity-certification-matrix.json", "json output path")
	markdownOutput := flags.String("markdown-output", "bigclaw-go/docs/reports/capacity-certification-report.md", "markdown output path")
	pretty := flags.Bool("pretty", false, "print json report to stdout")
	var soakReports soakReportFlags
	flags.Var(&soakReports, "supplemental-soak-report", "additional soak report path")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	absRepoRoot, err := filepath.Abs(*repoRoot)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	report, markdown, err := capacitycert.BuildReport(capacitycert.BuildOptions{
		RepoRoot:                    absRepoRoot,
		BenchmarkReportPath:         *benchmarkReport,
		MixedWorkloadReportPath:     *mixedWorkloadReport,
		SupplementalSoakReportPaths: soakReports,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := capacitycert.WriteOutputs(
		filepath.Join(absRepoRoot, *output),
		filepath.Join(absRepoRoot, *markdownOutput),
		report,
		markdown,
	); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return 1
		}
		_, _ = os.Stdout.Write(append(body, '\n'))
	}

	return 0
}
