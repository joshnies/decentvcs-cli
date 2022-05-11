package cmd

import (
	"bytes"
	"encoding/json"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Create a new branch.
func NewBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get branch name from args
	branchName := c.Args().First()
	if branchName == "" {
		return cli.Exit("Branch name is required", 1)
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Create branch
	apiUrl := api.BuildURLf("projects/%s/branches", projectConfig.ProjectID)
	bodyJson, err := json.Marshal(models.BranchCreateDTO{
		Name:        branchName,
		CommitIndex: projectConfig.CurrentCommitIndex,
	})
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	body := bytes.NewBuffer(bodyJson)
	branchRes, err := httpw.Post(apiUrl, body, gc.Auth.AccessToken)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Parse response
	var branch models.Branch
	err = json.NewDecoder(branchRes.Body).Decode(&branch)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Set current branch
	projectConfig.CurrentBranchID = branch.ID
	_, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	console.Info("Created and switched to branch %s", branch.Name)
	return nil
}
