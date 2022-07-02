package models

// Model for the DecentVCS project config file.
type ProjectConfig struct {
	ProjectID          string `yaml:"project" validate:"required"`
	CurrentBranchID    string `yaml:"branch" validate:"required"`
	CurrentCommitIndex int    `yaml:"commit" validate:"required,gt=0"`
}
