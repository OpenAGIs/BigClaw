package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"bigclaw-go/internal/refill"
)

type localIssueStoreFlags struct {
	repoRoot  *string
	storePath *string
	asJSON    *bool
}

func newLocalIssueStoreFlagSet(command string) (*flag.FlagSet, localIssueStoreFlags) {
	flags := flag.NewFlagSet("local-issues "+command, flag.ContinueOnError)
	return flags, localIssueStoreFlags{
		repoRoot:  flags.String("repo", "..", "repo root"),
		storePath: flags.String("local-issues", "", "local issue store path"),
		asJSON:    flags.Bool("json", false, "json"),
	}
}

func loadLocalIssueStore(repoRoot string, storePath string) (*refill.LocalIssueStore, string, error) {
	resolvedRepoRoot := absPath(repoRoot)
	resolvedStorePath := resolvePathAgainstRepoRoot(resolvedRepoRoot, storePath)
	resolvedLocalIssuesPath := resolvedLocalIssueStorePath(resolvedRepoRoot, resolvedStorePath)
	store, err := refill.LoadLocalIssueStore(resolvedLocalIssuesPath)
	if err != nil {
		return nil, "", err
	}
	return store, absPath(resolvedLocalIssuesPath), nil
}

func parseLabels(labelsCSV string) []string {
	labels := []string{}
	for _, label := range splitCSV(labelsCSV) {
		if trimmed := trim(label); trimmed != "" {
			labels = append(labels, trimmed)
		}
	}
	return labels
}

func runLocalIssuesList(args []string) error {
	flags, options := newLocalIssueStoreFlagSet("list")
	statesCSV := flags.String("states", "", "comma-separated state filter")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues list [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	store, resolvedStorePath, err := loadLocalIssueStore(*options.repoRoot, *options.storePath)
	if err != nil {
		return err
	}
	stateFilter := map[string]struct{}{}
	for _, state := range splitCSV(*statesCSV) {
		if trimmed := trim(state); trimmed != "" {
			stateFilter[trimmed] = struct{}{}
		}
	}
	filtered := make([]refill.LocalIssue, 0, len(store.Issues()))
	for _, issue := range store.Issues() {
		if len(stateFilter) != 0 {
			if _, ok := stateFilter[issue.State]; !ok {
				continue
			}
		}
		filtered = append(filtered, issue)
	}
	if *options.asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"local_issues": resolvedStorePath,
			"issues":       filtered,
		})
	}
	for _, issue := range filtered {
		fmt.Printf("%s\t%s\t%s\n", issue.Identifier, issue.State, issue.Title)
	}
	return nil
}

func runLocalIssuesCreate(args []string) error {
	flags, options := newLocalIssueStoreFlagSet("create")
	id := flags.String("id", "", "issue id (defaults to lowercased identifier)")
	identifier := flags.String("identifier", "", "issue identifier (e.g. BIG-PAR-104)")
	title := flags.String("title", "", "issue title")
	description := flags.String("description", "", "issue description")
	stateName := flags.String("state", "Todo", "state name")
	priority := flags.Int("priority", 3, "priority (1=urgent, 4=low)")
	labelsCSV := flags.String("labels", "", "comma-separated labels")
	assigned := flags.Bool("assigned-to-worker", true, "assigned to worker")
	createdAt := flags.String("created-at", "", "RFC3339 timestamp")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues create [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	store, resolvedStorePath, err := loadLocalIssueStore(*options.repoRoot, *options.storePath)
	if err != nil {
		return err
	}
	when, err := parseOptionalTime(*createdAt)
	if err != nil {
		return err
	}
	created, err := store.CreateIssue(refill.LocalIssueCreateParams{
		ID:               *id,
		Identifier:       *identifier,
		Title:            *title,
		Description:      *description,
		State:            *stateName,
		Priority:         *priority,
		Labels:           parseLabels(*labelsCSV),
		AssignedToWorker: *assigned,
		CreatedAt:        when,
	})
	if err != nil {
		return err
	}
	return emit(map[string]any{
		"status":       "ok",
		"backend":      "local",
		"issue":        created.Identifier,
		"state":        created.State,
		"local_issues": resolvedStorePath,
	}, *options.asJSON, 0)
}

func runLocalIssuesEnsure(args []string) error {
	flags, options := newLocalIssueStoreFlagSet("ensure")
	id := flags.String("id", "", "issue id (defaults to lowercased identifier)")
	identifier := flags.String("identifier", "", "issue identifier (e.g. BIG-PAR-104)")
	title := flags.String("title", "", "issue title (defaults to identifier)")
	description := flags.String("description", "", "issue description")
	stateName := flags.String("state", "Todo", "state name")
	priority := flags.Int("priority", 3, "priority (1=urgent, 4=low)")
	labelsCSV := flags.String("labels", "", "comma-separated labels")
	assigned := flags.Bool("assigned-to-worker", true, "assigned to worker")
	createdAt := flags.String("created-at", "", "RFC3339 timestamp")
	setStateIfExists := flags.Bool("set-state-if-exists", false, "update the issue state when the issue already exists")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues ensure [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	trimmedIdentifier := trim(*identifier)
	if trimmedIdentifier == "" {
		return errors.New("identifier is required")
	}
	store, resolvedStorePath, err := loadLocalIssueStore(*options.repoRoot, *options.storePath)
	if err != nil {
		return err
	}
	when, err := parseOptionalTime(*createdAt)
	if err != nil {
		return err
	}
	state := trim(*stateName)
	if state == "" {
		state = "Todo"
	}
	existing, found := store.FindIssue(trimmedIdentifier)
	action := "created"
	if found {
		action = "exists"
		if *setStateIfExists && existing.State != state {
			if when.IsZero() {
				when = time.Now()
			}
			updatedState, err := store.UpdateIssueState(trimmedIdentifier, state, when)
			if err != nil {
				return err
			}
			existing.State = updatedState
			action = "updated"
		}
		return emit(map[string]any{
			"status":       "ok",
			"backend":      "local",
			"action":       action,
			"issue":        existing.Identifier,
			"state":        existing.State,
			"local_issues": resolvedStorePath,
		}, *options.asJSON, 0)
	}
	if trim(*title) == "" {
		*title = trimmedIdentifier
	}
	created, err := store.CreateIssue(refill.LocalIssueCreateParams{
		ID:               *id,
		Identifier:       trimmedIdentifier,
		Title:            *title,
		Description:      *description,
		State:            state,
		Priority:         *priority,
		Labels:           parseLabels(*labelsCSV),
		AssignedToWorker: *assigned,
		CreatedAt:        when,
	})
	if err != nil {
		return err
	}
	return emit(map[string]any{
		"status":       "ok",
		"backend":      "local",
		"action":       action,
		"issue":        created.Identifier,
		"state":        created.State,
		"local_issues": resolvedStorePath,
	}, *options.asJSON, 0)
}

func runLocalIssuesSetState(args []string) error {
	flags, options := newLocalIssueStoreFlagSet("set-state")
	issueRef := flags.String("issue", "", "issue id or identifier")
	stateName := flags.String("state", "", "state name")
	createdAt := flags.String("created-at", "", "RFC3339 timestamp")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues set-state [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*issueRef) == "" {
		return errors.New("issue is required")
	}
	if trim(*stateName) == "" {
		return errors.New("state is required")
	}
	when, err := parseOptionalTime(*createdAt)
	if err != nil {
		return err
	}
	store, resolvedStorePath, err := loadLocalIssueStore(*options.repoRoot, *options.storePath)
	if err != nil {
		return err
	}
	updatedState, err := store.UpdateIssueState(*issueRef, *stateName, when)
	if err != nil {
		return err
	}
	return emit(map[string]any{
		"status":       "ok",
		"backend":      "local",
		"issue":        *issueRef,
		"state":        updatedState,
		"local_issues": resolvedStorePath,
	}, *options.asJSON, 0)
}

func runLocalIssuesComment(args []string) error {
	flags, options := newLocalIssueStoreFlagSet("comment")
	issueRef := flags.String("issue", "", "issue id or identifier")
	author := flags.String("author", "codex", "comment author")
	body := flags.String("body", "", "comment body")
	createdAt := flags.String("created-at", "", "RFC3339 timestamp")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl local-issues comment [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*issueRef) == "" {
		return errors.New("issue is required")
	}
	store, resolvedStorePath, err := loadLocalIssueStore(*options.repoRoot, *options.storePath)
	if err != nil {
		return err
	}
	when, err := parseOptionalTime(*createdAt)
	if err != nil {
		return err
	}
	if err := store.AddComment(*issueRef, refill.LocalIssueComment{Author: *author, Body: *body, CreatedAt: when}); err != nil {
		return err
	}
	return emit(map[string]any{
		"status":       "ok",
		"backend":      "local",
		"issue":        *issueRef,
		"author":       *author,
		"body":         *body,
		"local_issues": resolvedStorePath,
	}, *options.asJSON, 0)
}
