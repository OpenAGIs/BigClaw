package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/scripts/e2e/validationbundle"
)

func main() {
	output := flag.String("output", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "output path")
	pretty := flag.Bool("pretty", false, "print the generated report")
	flag.Parse()

	repoRoot, err := resolveRepoRoot()
	if err != nil {
		panic(err)
	}
	report, err := validationbundle.BuildScorecard(repoRoot, time.Now().UTC())
	if err != nil {
		panic(err)
	}
	if err := validationbundle.WriteJSON(resolvePath(repoRoot, *output), report); err != nil {
		panic(err)
	}
	if *pretty {
		body, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(body))
	}
}

func resolveRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if filepath.Base(wd) == "bigclaw-go" {
		return filepath.Dir(wd), nil
	}
	return wd, nil
}

func resolvePath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}
