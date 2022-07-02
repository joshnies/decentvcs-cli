package models

// Model for the DecentVCS project config file.
type ProjectConfig struct {
	ProjectID          string `yaml:"project" validate:"required;uuid4"`
	CurrentBranchID    string `yaml:"branch" validate:"required;uuid4"`
	CurrentCommitIndex int    `yaml:"commit" validate:"required;gt=0"`
}
