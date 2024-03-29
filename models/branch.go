package models

import "time"

type Branch struct {
	ID        string            `json:"_id,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	DeletedAt time.Time         `json:"deleted_at,omitempty"`
	Name      string            `json:"name,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	CommitID  string            `json:"commit_id,omitempty"`
	Locks     map[string]string `json:"locks,omitempty"`
}

type BranchWithCommit struct {
	ID        string            `json:"_id,omitempty"`
	CreatedAt time.Time         `json:"created_at,omitempty"`
	DeletedAt time.Time         `json:"deleted_at,omitempty"`
	Name      string            `json:"name,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	Commit    Commit            `json:"commit,omitempty"`
	Locks     map[string]string `json:"locks,omitempty"`
}

type BranchCreateDTO struct {
	Name        string `json:"name,omitempty"`
	CommitIndex int    `json:"commit_index,omitempty"`
}
