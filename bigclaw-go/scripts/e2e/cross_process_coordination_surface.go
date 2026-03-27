package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/crossprocesscoordination"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("cross_process_coordination_surface", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	output := flags.String("output", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "output path")
	pretty := flags.Bool("pretty", false, "pretty print")
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
	report, err := crossprocesscoordination.BuildReport(crossprocesscoordination.BuildOptions{
		RepoRoot: absRepoRoot,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := crossprocesscoordination.WriteReport(filepath.Join(absRepoRoot, *output), report, true); err != nil {
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
