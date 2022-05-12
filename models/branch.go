package models

type Branch struct {
	ID        string `json:"_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	CommitID  string `json:"commit_id,omitempty"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
}

type BranchWithCommit struct {
	ID        string `json:"_id,omitempty"`
	CreatedAt int64  `json:"created_at,omitempty"`
	Name      string `json:"name,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Commit    Commit `json:"commit,omitempty"`
	DeletedAt int64  `json:"deleted_at,omitempty"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}
