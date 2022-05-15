package cmd

import (
	"bytes"
	"encoding/json"

	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/console"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/models"
	"github.com/urfave/cli/v2"
)

// Create a new branch.
func NewBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get branch name from args
	branchName := c.Args().First()
	if branchName == "" {
		return console.Error("Branch name is required")
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Create branch
	apiUrl := api.BuildURLf("projects/%s/branches", projectConfig.ProjectID)
	bodyJson, err := json.Marshal(models.BranchCreateDTO{
		Name:        branchName,
		CommitIndex: projectConfig.CurrentCommitIndex,
	})
	if err != nil {
		return err
	}

	body := bytes.NewBuffer(bodyJson)
	branchRes, err := httpw.Post(apiUrl, body, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse response
	var branch models.Branch
	err = json.NewDecoder(branchRes.Body).Decode(&branch)
	if err != nil {
		return err
	}

	// Set current branch
	projectConfig.CurrentBranchID = branch.ID
	_, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	console.Info("Created and switched to branch %s", branch.Name)
	return nil
}
