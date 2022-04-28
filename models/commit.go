package models

type Commit struct {
	ID            string            `json:"_id,omitempty"`
	CreatedAt     int64             `json:"created_at,omitempty"`
	LastCommitID  string            `json:"last_commit_id,omitempty"`
	Message       string            `json:"message,omitempty"`
	ProjectID     string            `json:"project_id,omitempty"`
	SnapshotPaths []string          `json:"snapshot_paths,omitempty"`
	PatchPaths    []string          `json:"patch_paths,omitempty"`
	DeletedPaths  []string          `json:"deleted_paths,omitempty"`
	HashMap       map[string]string `json:"hash_map,omitempty"`
	// TODO: Add user ID
}
