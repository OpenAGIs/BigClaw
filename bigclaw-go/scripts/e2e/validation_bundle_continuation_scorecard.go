package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/continuationscorecard"
)

func main() {
	os.Exit(run())
}

func run() int {
	flags := flag.NewFlagSet("validation_bundle_continuation_scorecard", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", ".", "repository root")
	output := flags.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
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
	report, err := continuationscorecard.BuildReport(continuationscorecard.BuildOptions{RepoRoot: absRepoRoot})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := continuationscorecard.WriteReport(filepath.Join(absRepoRoot, *output), report, true); err != nil {
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
