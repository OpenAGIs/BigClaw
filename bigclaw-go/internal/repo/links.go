package repo

import (
	"fmt"
	"sort"
)

var validRunCommitRoles = map[string]struct{}{
	"source":    {},
	"candidate": {},
	"closeout":  {},
	"accepted":  {},
}

type RunCommitBinding struct {
	Links []RunCommitLink `json:"links"`
}

func (b RunCommitBinding) AcceptedCommitHash() string {
	for _, link := range b.Links {
		if link.Role == "accepted" {
			return link.CommitHash
		}
	}
	return ""
}

func ValidateRunCommitRoles(links []RunCommitLink) error {
	invalidSet := map[string]struct{}{}
	for _, link := range links {
		if _, ok := validRunCommitRoles[link.Role]; !ok {
			invalidSet[link.Role] = struct{}{}
		}
	}
	if len(invalidSet) == 0 {
		return nil
	}
	invalid := make([]string, 0, len(invalidSet))
	for role := range invalidSet {
		invalid = append(invalid, role)
	}
	sort.Strings(invalid)
	return fmt.Errorf("unsupported run commit roles: %s", joinCSV(invalid))
}

func BindRunCommits(links []RunCommitLink) (RunCommitBinding, error) {
	if err := ValidateRunCommitRoles(links); err != nil {
		return RunCommitBinding{}, err
	}
	return RunCommitBinding{Links: append([]RunCommitLink(nil), links...)}, nil
}

func joinCSV(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	}
	out := values[0]
	for _, value := range values[1:] {
		out += ", " + value
	}
	return out
}
