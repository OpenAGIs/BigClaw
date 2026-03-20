package repo

type RepoCommit struct {
	CommitHash   string         `json:"commit_hash"`
	Title        string         `json:"title"`
	Author       string         `json:"author,omitempty"`
	ParentHashes []string       `json:"parent_hashes,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type CommitLineage struct {
	RootHash string              `json:"root_hash"`
	Lineage  []RepoCommit        `json:"lineage,omitempty"`
	Children map[string][]string `json:"children,omitempty"`
	Leaves   []string            `json:"leaves,omitempty"`
}

type CommitDiff struct {
	LeftHash     string `json:"left_hash"`
	RightHash    string `json:"right_hash"`
	FilesChanged int    `json:"files_changed"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
	Summary      string `json:"summary,omitempty"`
}
