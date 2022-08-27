package models

import "time"

type Commit struct {
	ID        string    `json:"_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Index     int       `json:"index,omitempty"`
	Message   string    `json:"message,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	BranchID  string    `json:"branch_id,omitempty"`
	// Array of relative fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty"`
	// Array of relative fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty"`
	// Array of relative fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty"`
	// Map of relative fs paths to their associated data
	Files map[string]FileData `json:"files,omitempty"`
	// ID of the user who made the commit.
	// If empty, then the system created it.
	AuthorID string `json:"author_id,omitempty"`
}

type CommitWithBranch struct {
	ID        string    `json:"_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Index     int       `json:"index,omitempty"`
	Message   string    `json:"message,omitempty"`
	ProjectID string    `json:"project_id,omitempty"`
	Branch    Branch    `json:"branch,omitempty"`
	// Array of relative fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty"`
	// Array of relative fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty"`
	// Array of relative fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty"`
	// Map of relative fs paths to their associated data
	Files map[string]FileData `json:"files,omitempty"`
	// ID of the user who made the commit.
	// If empty, then the system created it.
	AuthorID string `json:"author_id,omitempty"`
}
