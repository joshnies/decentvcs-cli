package models

type ProjectConfig struct {
	ProjectID          string `json:"project_id,omitempty"`
	CurrentBranchID    string `json:"current_branch_id,omitempty"`
	CurrentCommitIndex int    `json:"current_commit_index,omitempty"`
}

// Merge new project config into the old one.
func MergeProjectConfigs(oldData ProjectConfig, newData ProjectConfig) ProjectConfig {
	merged := oldData

	if newData.ProjectID != "" {
		merged.ProjectID = newData.ProjectID
	}

	if newData.CurrentBranchID != "" {
		merged.CurrentBranchID = newData.CurrentBranchID
	}

	if newData.CurrentCommitIndex != 0 {
		merged.CurrentCommitIndex = newData.CurrentCommitIndex
	}

	return merged
}
