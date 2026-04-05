package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/reporting"
)

func main() {
	options, pretty, err := reporting.ParseTaskSmokeCLIFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if options.GoRoot == "" {
		options.GoRoot, err = reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
		if err != nil {
			options.GoRoot, err = reporting.FindRepoRoot(".")
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	result, err := reporting.RunTaskSmoke(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, marshalErr := json.MarshalIndent(result.Report, "", "  ")
	if marshalErr != nil {
		fmt.Fprintln(os.Stderr, marshalErr)
		os.Exit(1)
	}
	if result.ExitCode == 0 {
		fmt.Println(string(contents))
	} else {
		fmt.Fprintln(os.Stderr, string(contents))
	}
	if result.ServiceLogPath != "" {
		fmt.Fprintf(os.Stderr, "bigclawd log: %s\n", result.ServiceLogPath)
	}
	if pretty && result.ExitCode == 0 {
		return
	}
	os.Exit(result.ExitCode)
}
