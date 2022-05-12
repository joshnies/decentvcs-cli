package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Soft-delete the specified branch.
// Does NOT effect the branch's commits.
func DeleteBranch(c *cli.Context) error {
	gc := auth.Validate()

	// Get the branch name
	branchName := c.Args().First()
	if branchName == "" {
		return cli.Exit("You must specify a branch name", 1)
	}

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	// Get specified branch
	apiUrl := api.BuildURLf("projects/%s/branches/%s?join_commit=true", projectConfig.ProjectID, branchName)
	branchRes, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	// Parse response
	var branch models.BranchWithCommit
	err = json.NewDecoder(branchRes.Body).Decode(&branch)
	if err != nil {
		return err
	}

	// Prevent deletion of current branch
	if branch.ID == projectConfig.CurrentBranchID {
		return console.Error("You cannot delete the current branch. Please switch to another branch first.")
	}

	// Ask for confirmation
	if !c.Bool("no-confirm") {
		console.Warning("Are you sure you want to delete the branch \"%s\"?", branchName)
		var answer string
		fmt.Scanln(&answer)
	}

	// Soft-delete branch
	apiUrl = api.BuildURLf("projects/%s/branches/%s", projectConfig.ProjectID, branchName)
	_, err = httpw.Delete(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		return err
	}

	console.Success("Branch deleted")
	return nil
}
