package cmd

import (
	"encoding/json"
	"time"

	"github.com/joshnies/qc-cli/config"
	"github.com/joshnies/qc-cli/constants"
	"github.com/joshnies/qc-cli/lib/api"
	"github.com/joshnies/qc-cli/lib/auth"
	"github.com/joshnies/qc-cli/lib/console"
	"github.com/joshnies/qc-cli/lib/httpw"
	"github.com/joshnies/qc-cli/models"
	"github.com/urfave/cli/v2"
)

// Pull changes from remote
func Pull(c *cli.Context) error {
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

	// Get specified commit ID from args
	commitID := c.Args().Get(0)
	if commitID == "" {
		// Get latest commit on current branch
		commitRes, err := httpw.Get(api.BuildURLf("projects/%s/branches/%s/commit", projectConfig.ProjectID, projectConfig.CurrentBranchID), gc.Auth.AccessToken)
		if err != nil {
			return err
		}
		defer commitRes.Body.Close()

		// Parse commit
		var commit models.Commit
		err = json.NewDecoder(commitRes.Body).Decode(&commit)
		if err != nil {
			return console.Error(constants.ErrMsgInternal)
		}

		// Return if commit is the same as current commit
		if commit.ID == projectConfig.CurrentCommitID {
			console.Info("No changes to pull")
			return nil
		}

		// Pull changes up to latest commit (inclusive)
		return pullFwd(gc, projectConfig, commit.ID)
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

	// Get specified commit
	commitRes, err = httpw.Get(api.BuildURLf("projects/%s/commits/%s", projectConfig.ProjectID, commitID), gc.Auth.AccessToken)
	if err != nil {
		return err
	}
	defer commitRes.Body.Close()

	// Parse commit
	var specifiedCommit models.Commit
	err = json.NewDecoder(commitRes.Body).Decode(&specifiedCommit)
	if err != nil {
		return console.Error(constants.ErrMsgInternal)
	}

	// Return if commit is the same as current commit
	if specifiedCommit.ID == currentCommit.ID {
		console.Info("No changes to pull")
	}

	// Pull changes up or down depending on which commit is newer
	if time.Unix(specifiedCommit.CreatedAt, 0).After(time.Unix(currentCommit.CreatedAt, 0)) {
		return pullFwd(gc, projectConfig, specifiedCommit.ID)
	}

	return pullBwd(gc, projectConfig, specifiedCommit.ID)
}

// Pull a commit that's after the current commit
func pullFwd(gc models.GlobalConfig, projectConfig models.ProjectConfig, afterCommitId string) error {
	// Get newer commits from remote for current branch
	apiUrl := api.BuildURLf("projects/%s/branches/%s/commits?after=%s", projectConfig.ProjectID, projectConfig.CurrentBranchID, afterCommitId)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		console.Verbose("Error fetching commits: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Parse response
	var commits []models.Commit
	err = json.NewDecoder(res.Body).Decode(&commits)
	if err != nil {
		console.Verbose("Error parsing commits from API response: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Return if no new commits found
	if len(commits) == 0 {
		console.Info("No changes to pull.")
		return nil
	}

	// TODO: Implement

	console.Success("Successful")
	return nil
}

// Pull a commit that's before the current commit
func pullBwd(gc models.GlobalConfig, projectConfig models.ProjectConfig, beforeCommitId string) error {
	// Get older commits from remote for current branch
	apiUrl := api.BuildURLf("projects/%s/branches/%s/commits?before=%s", projectConfig.ProjectID, projectConfig.CurrentBranchID, beforeCommitId)
	res, err := httpw.Get(apiUrl, gc.Auth.AccessToken)
	if err != nil {
		console.Verbose("Error fetching commits: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Parse response
	var commits []models.Commit
	err = json.NewDecoder(res.Body).Decode(&commits)
	if err != nil {
		console.Verbose("Error parsing commits from API response: %s", err)
		return console.Error("Failed to fetch commits")
	}

	// Return if no new commits found
	if len(commits) == 0 {
		console.Info("No changes to pull.")
		return nil
	}

	// TODO: Implement

	console.Success("Successful")
	return nil
}
