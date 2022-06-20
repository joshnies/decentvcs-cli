package models

type ProjectConfig struct {
	ProjectID          string `yaml:"project"`
	CurrentBranchID    string `yaml:"branch"`
	CurrentCommitIndex int    `yaml:"commit"`
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
