package cmd

import (
	"encoding/json"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Sync local project to a commit
func Sync(c *cli.Context) error {
	gc := auth.Validate()

	// Get project config
	projectConfig, err := config.GetProjectConfig()
	if err != nil {
		return err
	}

	if projectConfig.CurrentCommitID == "" {
		// TODO: Add option for user to sync local project with remote, no matter what commit they're on
		return console.Error("Current commit ID is invalid. Please check your project config file.")
	}

	// Get current commit
	commitRes, err := httpw.Get(api.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, projectConfig.CurrentCommitID), gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	// Parse commit
	var currentCommit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&currentCommit)
	if err != nil {
		return console.Error(constants.ErrMsgInternal)
	}

	// Get specified commit ID from args; default to latest commit
	var toCommit models.Commit
	commitID := c.Args().Get(0)

	if commitID == "" {
		// Get latest commit on current branch
		commitRes, err := httpw.Get(api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer commitRes.Body.Close()

		// Parse commit
		err = json.NewDecoder(commitRes.Body).Decode(&toCommit)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}
	} else {
		// Get user-specified commit
		commitRes, err = httpw.Get(api.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, commitID), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer commitRes.Body.Close()

		// Parse commit
		err = json.NewDecoder(commitRes.Body).Decode(&toCommit)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}
	}

	// Return if commit is the same as current commit
	if toCommit.ID == projectConfig.CurrentCommitID {
		console.Info("You are already on this commit")
	}

	// TODO: Download new files
	// TODO: Delete deleted files

	return nil
}
