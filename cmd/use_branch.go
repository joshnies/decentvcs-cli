package cmd

import (
	"encoding/json"

	"github.com/joshnies/qc/config"
	"github.com/joshnies/qc/lib/api"
	"github.com/joshnies/qc/lib/auth"
	"github.com/joshnies/qc/lib/commits"
	"github.com/joshnies/qc/lib/httpw"
	"github.com/joshnies/qc/lib/projects"
	"github.com/joshnies/qc/models"
	"github.com/urfave/cli/v2"
)

// Switch to the specified branch.
// This will also sync to the latest commit on that branch.
func UseBranch(c *cli.Context) error {
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

	// Set the current branch in project config
	projectConfig.CurrentBranchID = branch.ID
	projectConfig, err = config.SaveProjectConfig(".", projectConfig)
	if err != nil {
		return err
	}

	// Reset local changes if specified branch points to a different commit than current
	if projectConfig.CurrentCommitIndex != branch.Commit.Index {
		// Reset local changes
		err = projects.ResetChanges(gc, !c.Bool("no-confirm"))
		if err != nil {
			return err
		}
	}

	// Sync
	err = commits.SyncToCommit(gc, projectConfig, branch.Commit.Index, true)
	if err != nil {
		return err
	}

	return nil
}
