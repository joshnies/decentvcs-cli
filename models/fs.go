package models

type ProjectFileData struct {
	ProjectID       string `json:"project_id,omitempty"`
	CurrentBranchID string `json:"current_branch_id,omitempty"`
}
