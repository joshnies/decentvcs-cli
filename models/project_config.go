package models

type ProjectConfig struct {
	ProjectID       string `json:"project_id,omitempty"`
	CurrentBranchID string `json:"current_branch_id,omitempty"`
}

func MergeProjectConfigs(existingData, newData ProjectConfig) ProjectConfig {
	merged := existingData

	if newData.ProjectID != "" {
		merged.ProjectID = newData.ProjectID
	}

	if newData.CurrentBranchID != "" {
		merged.CurrentBranchID = newData.CurrentBranchID
	}

	return merged
}
