package models

type Branch struct {
	ID        string `json:"_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	CommitID  string `json:"commit_id,omitempty"`
}

type BranchWithCommit struct {
	ID        string `json:"_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	Commit    Commit `json:"commit,omitempty"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}
