package models

type CommitState struct {
	// File hash used for determining changes
	Hash string `json:"hash"`
	// ID of latest commit that modified the file
	HostCommitId string `json:"host_commit_id"`
}

type Commit struct {
	ID            string                 `json:"_id,omitempty"`
	CreatedAt     int64                  `json:"created_at,omitempty"`
	LastCommitID  string                 `json:"last_commit_id,omitempty"`
	Message       string                 `json:"message,omitempty"`
	ProjectID     string                 `json:"project_id,omitempty"`
	BranchID      string                 `json:"branch_id,omitempty"`
	SnapshotPaths []string               `json:"snapshot_paths,omitempty"`
	PatchPaths    []string               `json:"patch_paths,omitempty"`
	DeletedPaths  []string               `json:"deleted_paths,omitempty"`
	State         map[string]CommitState `json:"state,omitempty"`
	// TODO: Add user ID
}
