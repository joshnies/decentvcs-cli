package models

// Model for the DecentVCS project config file.
type ProjectConfig struct {
	ProjectSlug        string `yaml:"project" validate:"required"`
	CurrentBranchName  string `yaml:"branch" validate:"required"`
	CurrentCommitIndex int    `yaml:"commit" validate:"required,gt=0"`
}
