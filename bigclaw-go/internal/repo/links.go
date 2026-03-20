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
	Links              []RunCommitLink `json:"links,omitempty"`
	AcceptedCommitHash string          `json:"accepted_commit_hash,omitempty"`
}

func BindRunCommits(links []RunCommitLink) (RunCommitBinding, error) {
	if err := ValidateRunCommitRoles(links); err != nil {
		return RunCommitBinding{}, err
	}
	return RunCommitBinding{
		Links:              append([]RunCommitLink(nil), links...),
		AcceptedCommitHash: acceptedCommitHash(links),
	}, nil
}

func ValidateRunCommitRoles(links []RunCommitLink) error {
	invalid := make(map[string]struct{})
	for _, link := range links {
		if _, ok := validRunCommitRoles[link.Role]; !ok {
			invalid[link.Role] = struct{}{}
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	roles := make([]string, 0, len(invalid))
	for role := range invalid {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return fmt.Errorf("unsupported run commit roles: %s", joinStrings(roles, ", "))
}

func acceptedCommitHash(links []RunCommitLink) string {
	for _, link := range links {
		if link.Role == "accepted" {
			return link.CommitHash
		}
	}
	return ""
}
