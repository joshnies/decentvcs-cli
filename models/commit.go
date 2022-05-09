package models

type Commit struct {
	ID           string `json:"_id,omitempty"`
	CreatedAt    int64  `json:"created_at,omitempty"`
	Index        int    `json:"index,omitempty"`
	LastCommitID string `json:"last_commit_id,omitempty"`
	Message      string `json:"message,omitempty"`
	ProjectID    string `json:"project_id,omitempty"`
	BranchID     string `json:"branch_id,omitempty"`
	// Array of fs paths to created files (uploaded as snapshots)
	CreatedFiles []string `json:"created_files,omitempty"`
	// Array of fs paths to modified files (uploaded as snapshots)
	ModifiedFiles []string `json:"modified_files,omitempty"`
	// Array of fs paths to deleted files
	DeletedFiles []string `json:"deleted_files,omitempty"`
	// Map of file path to hash
	HashMap map[string]string `json:"hash_map,omitempty"`
	// TODO: Add user ID
}
