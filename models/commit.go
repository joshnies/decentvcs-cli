package models

type Commit struct {
	ID            string   `json:"_id,omitempty"`
	CreatedAt     int64    `json:"created_at,omitempty"`
	Name          string   `json:"name,omitempty"`
	ProjectID     string   `json:"project_id,omitempty"`
	SnapshotPaths []string `json:"snapshot_paths,omitempty"`
	PatchPaths    []string `json:"patch_paths,omitempty"`
}